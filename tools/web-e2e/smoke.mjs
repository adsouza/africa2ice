import { createServer } from "node:http";
import { readFile, stat } from "node:fs/promises";
import { extname, resolve } from "node:path";
import { chromium } from "playwright";

const root = resolve(import.meta.dirname, "../..");
const webRoot = resolve(root, "web");
for (const name of ["index.html", "main.wasm", "wasm_exec.js"]) {
  const info = await stat(resolve(webRoot, name));
  if (!info.isFile() || info.size === 0) throw new Error(`staged web artifact ${name} is missing or empty`);
}

// The host page owns CSS sizing only (DESIGN.md step 11): no script may read
// devicePixelRatio, assign a canvas backing size, or rescale input. Ebiten
// does all three, and the DPR checks below prove it.
const hostPage = await readFile(resolve(webRoot, "index.html"), "utf8");
for (const [pattern, meaning] of [[/devicePixelRatio/, "reads devicePixelRatio"], [/\.(width|height)\s*=(?!=)/, "assigns an element width or height"], [/getContext\s*\(/, "draws on a canvas itself"]]) {
  if (pattern.test(hostPage)) throw new Error(`web/index.html ${meaning}; canvas sizing and scale belong to Ebiten`);
}

const mime = new Map([
  [".html", "text/html; charset=utf-8"],
  [".js", "text/javascript; charset=utf-8"],
  [".wasm", "application/wasm"],
]);
const server = createServer(async (request, response) => {
  const url = new URL(request.url, "http://127.0.0.1");
  const name = url.pathname === "/" ? "index.html" : url.pathname.slice(1);
  if (!new Set(["index.html", "main.wasm", "wasm_exec.js"]).has(name)) {
    response.writeHead(404).end();
    return;
  }
  const body = await readFile(resolve(webRoot, name));
  response.writeHead(200, { "Content-Type": mime.get(extname(name)) ?? "application/octet-stream", "Cache-Control": "no-store" });
  response.end(body);
});
await new Promise(resolveListen => server.listen(0, "127.0.0.1", resolveListen));
const { port } = server.address();
const origin = `http://127.0.0.1:${port}`;
const browser = await chromium.launch({ headless: true });
const pressGameKey = (page, key) => page.keyboard.press(key, { delay: 40 });

// The web logging sink writes one JSON object per line to console.log.
// Records are kept per page so a check can ask what this page has logged.
const logRecords = new WeakMap();
const recordsOf = page => logRecords.get(page) ?? [];
const waitForRecord = async (page, description, matches, timeout = 5_000) => {
  const deadline = Date.now() + timeout;
  while (Date.now() < deadline) {
    if (recordsOf(page).some(matches)) return;
    await page.waitForTimeout(25);
  }
  throw new Error(`no console log record ${description}; saw ${recordsOf(page).map(record => record.msg).join(", ")}`);
};
const turnOf = async page => JSON.parse(await page.locator("html").getAttribute("data-africa2ice-summary")).turn;
const waitForTurn = async (page, turn, failure) => {
  try {
    await page.waitForFunction(expected => JSON.parse(document.documentElement.dataset.africa2iceSummary).turn === expected, turn, { timeout: 5_000 });
  } catch {
    throw new Error(`${failure} (turn is ${await turnOf(page)}, want ${turn})`);
  }
};

// End turn is the panel's bottom button. Its centre in the 1280x720 DIP
// presentation; FitPresentation letterboxes that presentation into any other
// window, so a larger window scales the point and offsets it by the bars.
const endTurnDIP = { x: 1084, y: 661 };
const presentationPoint = ({ width, height }, point) => {
  const scale = Math.min(width / 1280, height / 720);
  return { x: (width - 1280 * scale) / 2 + point.x * scale, y: (height - 720 * scale) / 2 + point.y * scale };
};
// With bands still waiting, the first End turn click only arms a
// confirmation; the second ends the turn. Ebiten samples input once a frame,
// so each press is held across a few frames rather than clicked instantly.
const clickEndTurnTwice = async (page, point, touch) => {
  const cdp = touch ? await page.context().newCDPSession(page) : null;
  if (!touch) await page.mouse.move(point.x, point.y, { steps: 4 });
  for (let attempt = 0; attempt < 2; attempt++) {
    if (touch) {
      await cdp.send("Input.dispatchTouchEvent", { type: "touchStart", touchPoints: [{ x: point.x, y: point.y }] });
      await page.waitForTimeout(150);
      await cdp.send("Input.dispatchTouchEvent", { type: "touchEnd", touchPoints: [] });
    } else {
      await page.mouse.down();
      await page.waitForTimeout(150);
      await page.mouse.up();
    }
    await page.waitForTimeout(300);
  }
};

const openGame = async (context, page) => {
  const failures = [];
  page.on("pageerror", error => failures.push(`pageerror: ${error.message}`));
  logRecords.set(page, []);
  page.on("console", message => {
    if (message.type() === "error") failures.push(`console.error: ${message.text()}`);
    if (message.type() !== "log" || !message.text().startsWith("{")) return;
    try {
      recordsOf(page).push(JSON.parse(message.text()));
    } catch {
      failures.push(`console log line is not one JSON object: ${message.text()}`);
    }
  });
  page.on("request", request => {
    if (!request.url().startsWith(origin)) failures.push(`unexpected network request: ${request.url()}`);
  });
  await page.addInitScript(() => {
    window.africa2iceE2EObserver = summary => {
      window.__africa2iceSummaries ??= [];
      window.__africa2iceSummaries.push(summary);
    };
  });
  await page.goto(`${origin}/?e2e=1`, { waitUntil: "load" });
  await page.waitForFunction(() => document.documentElement.dataset.africa2iceReady === "true", null, { timeout: 10_000 });
  await page.waitForFunction(() => Boolean(document.documentElement.dataset.africa2iceSummary), null, { timeout: 5_000 });
  await pressGameKey(page, "Enter");
  await page.waitForTimeout(50);
  if (failures.length > 0) throw new Error(failures.join("\n"));
  return failures;
};

try {
  const context = await browser.newContext({ viewport: { width: 1280, height: 720 }, deviceScaleFactor: 2 });
  let page = await context.newPage();
  let failures = await openGame(context, page);
  const initial = JSON.parse(await page.locator("html").getAttribute("data-africa2ice-summary"));
  const starts = recordsOf(page).filter(record => record.msg === "session.start");
  if (starts.length !== 1 || starts[0].target !== "web") throw new Error(`expected one web session.start record, saw ${JSON.stringify(starts)}`);

  let queued = false;
  for (const keys of [["ArrowUp"], ["ArrowRight"], ["ArrowDown"], ["ArrowLeft"], ["ArrowUp", "ArrowRight"], ["ArrowDown", "ArrowRight"], ["ArrowDown", "ArrowLeft"], ["ArrowUp", "ArrowLeft"]]) {
    await pressGameKey(page, "Escape");
    for (const key of keys) await pressGameKey(page, key);
    await pressGameKey(page, "Enter");
    await page.waitForTimeout(75);
    const summary = JSON.parse(await page.locator("html").getAttribute("data-africa2ice-summary"));
    if (summary.world_revision > initial.world_revision) {
      queued = true;
      break;
    }
  }
  if (!queued) throw new Error("could not queue a legal migration with the keyboard");
  await waitForRecord(page, "decorating the queued migration", record => record.msg === "action.dispatch" && record.command === "queue_migration");

  await pressGameKey(page, "Space");
  await page.waitForFunction(() => JSON.parse(document.documentElement.dataset.africa2iceSummary).turn === 1, null, { timeout: 5_000 });
  const saved = await page.locator("html").getAttribute("data-africa2ice-summary");
  await pressGameKey(page, "Control+S");
  await page.waitForTimeout(1_000);
  if (failures.length > 0) throw new Error(failures.join("\n"));

  await page.close();
  page = await context.newPage();
  failures = await openGame(context, page);
  await page.waitForFunction(() => JSON.parse(document.documentElement.dataset.africa2iceSummary).turn === 1, null, { timeout: 5_000 });
  const restored = await page.locator("html").getAttribute("data-africa2ice-summary");
  if (restored !== saved) throw new Error(`quick-save reload changed the semantic frame\nbefore ${saved}\nafter  ${restored}`);

  await pressGameKey(page, "F1");
  await page.waitForTimeout(1_000);
  await pressGameKey(page, "Space");
  await page.waitForFunction(() => JSON.parse(document.documentElement.dataset.africa2iceSummary).turn === 2, null, { timeout: 5_000 });
  await page.keyboard.down("Shift");
  await pressGameKey(page, "F1");
  await page.waitForTimeout(50);
  await page.keyboard.up("Shift");
  await page.waitForFunction(expected => document.documentElement.dataset.africa2iceSummary === expected, saved, { timeout: 5_000 });
  const manualRestored = await page.locator("html").getAttribute("data-africa2ice-summary");
  if (manualRestored !== saved) throw new Error(`manual-slot load changed the semantic frame\nbefore ${saved}\nafter  ${manualRestored}`);
  if (failures.length > 0) throw new Error(failures.join("\n"));
  await context.close();

  // An induced error must reach the console too: loading the never-written
  // manual slot 3 fails, and its completion is logged with outcome "error".
  {
    const errorContext = await browser.newContext({ viewport: { width: 1280, height: 720 } });
    const errorPage = await errorContext.newPage();
    await openGame(errorContext, errorPage);
    await errorPage.keyboard.down("Shift");
    await pressGameKey(errorPage, "F3");
    await errorPage.keyboard.up("Shift");
    await waitForRecord(errorPage, "for the failed load of empty slot 3", record => record.msg === "repository.completion" && record.slot === 3 && record.outcome === "error");
    await errorContext.close();
  }

  // Stable DIP geometry and matching pointer hits. Each DPR keeps a 1280x720
  // CSS viewport, so the same CSS point must hit End turn whatever the
  // backing scale. DPR 1.5 is what 150% browser zoom of a 1920x1080 window
  // presents. Ebiten's own canvas must take the full device-pixel backing,
  // which is what leaving DisableHiDPI false means.
  const viewport = { width: 1280, height: 720 };
  for (const [deviceScaleFactor, touch] of [[1, false], [1.25, false], [1.5, false], [2, false], [3, false], [2, true]]) {
    const hitContext = await browser.newContext({ viewport, deviceScaleFactor, hasTouch: touch });
    const hitPage = await hitContext.newPage();
    await openGame(hitContext, hitPage);
    const backing = await hitPage.evaluate(() => [...document.querySelectorAll("canvas")].filter(canvas => !canvas.id).map(canvas => [canvas.width, canvas.height]));
    const expected = [Math.round(1280 * deviceScaleFactor), Math.round(720 * deviceScaleFactor)];
    if (backing.length !== 1 || backing[0][0] !== expected[0] || backing[0][1] !== expected[1]) {
      throw new Error(`DPR ${deviceScaleFactor}: Ebiten canvas backing ${JSON.stringify(backing)}, want ${expected}`);
    }
    await clickEndTurnTwice(hitPage, presentationPoint(viewport, endTurnDIP), touch);
    await waitForTurn(hitPage, 1, `DPR ${deviceScaleFactor}${touch ? " touch" : ""}: two presses on End turn did not end the turn`);
    await hitContext.close();
  }

  // Resize: a larger window letterboxes the presentation, and the scaled
  // point must still hit End turn.
  {
    const resizeContext = await browser.newContext({ viewport });
    const resizePage = await resizeContext.newPage();
    await openGame(resizeContext, resizePage);
    const larger = { width: 1600, height: 900 };
    await resizePage.setViewportSize(larger);
    await resizePage.waitForTimeout(300);
    await clickEndTurnTwice(resizePage, presentationPoint(larger, endTurnDIP), false);
    await waitForTurn(resizePage, 1, "after resizing to 1600x900 the scaled End turn point missed");
    await resizeContext.close();
  }

  // 200% zoom of a 1920x1080 window and a portrait phone both present less
  // than 1280x720 DIP: gameplay must be gated, then restored once the window
  // is large enough again.
  for (const [label, small, deviceScaleFactor] of [["200% zoom", { width: 960, height: 540 }, 2], ["portrait", { width: 720, height: 1280 }, 3]]) {
    const gateContext = await browser.newContext({ viewport: small, deviceScaleFactor });
    const gatePage = await gateContext.newPage();
    await openGame(gateContext, gatePage);
    await pressGameKey(gatePage, "Space");
    await gatePage.waitForTimeout(500);
    if ((await turnOf(gatePage)) !== 0) throw new Error(`${label}: gameplay was not gated below 1280x720 DIP`);
    await gatePage.setViewportSize(viewport);
    await gatePage.waitForTimeout(300);
    // openGame's Enter was gated as well, so the title scene is still up.
    await pressGameKey(gatePage, "Enter");
    await gatePage.waitForTimeout(100);
    await pressGameKey(gatePage, "Space");
    await waitForTurn(gatePage, 1, `${label}: gameplay did not resume at 1280x720`);
    await gateContext.close();
  }
  process.stdout.write("browser smoke passed: boot, migration, turn, quick-save/reload, manual save/load, console logs, DPR 1/1.25/1.5/2/3 hits and backing, touch, resize, 200% zoom and portrait gating\n");
} finally {
  await browser.close();
  await new Promise(resolveClose => server.close(resolveClose));
}
