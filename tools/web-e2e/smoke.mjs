import { createServer } from "node:http";
import { mkdir, readFile, stat } from "node:fs/promises";
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
// Ebiten samples keys per frame. Give software-rendered high-DPI frames time
// to observe both edges, including the release between repeated PageDown keys.
const pressGameKey = async (page, key) => {
  await page.keyboard.press(key, { delay: 100 });
  await page.waitForTimeout(100);
};

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
  const recent = recordsOf(page).slice(-12).map(record => record.kind ? `${record.msg}(${record.kind})` : record.msg);
  throw new Error(`no console log record ${description}; most recent: ${recent.join(", ")}`);
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
// confirmation; a later one ends the turn. Ebiten samples input once a frame,
// and a software-rendered CI browser can take longer than any fixed delay to
// draw a high-DPR frame, so each press is held until the game logs that it
// saw it (ui.pointer) before it is released; a touch tap, which ebitenui
// reads without that record, is retried with a doubling hold instead. The first click also rebuilds
// the panel to show its "click again" notice, and a press that lands just
// before that rebuild is released onto the new button and never becomes a
// click, so a press is retried until the turn ends. Every intent the presses
// produce must be end-turn: anything else means the point missed the button.
const countRecords = (page, matches) => recordsOf(page).filter(matches).length;
const isPointer = record => record.msg === "ui.pointer";
const isIntent = record => record.msg === "ui.intent";
const clickEndTurnUntilTheTurnEnds = async (page, point, touch, label) => {
  const cdp = touch ? await page.context().newCDPSession(page) : null;
  if (!touch) await page.mouse.move(point.x, point.y, { steps: 4 });
  const intentsBefore = countRecords(page, isIntent);
  for (let press = 1; press <= 5 && (await turnOf(page)) === 0; press++) {
    const pointers = countRecords(page, isPointer);
    const intents = countRecords(page, isIntent);
    if (touch) {
      // ebitenui reads touches itself, without the mouse-press path that
      // logs ui.pointer, so a tap has no observable press. Its hold doubles
      // on each retry instead, until it spans a frame however slow the
      // browser renders.
      await cdp.send("Input.dispatchTouchEvent", { type: "touchStart", touchPoints: [{ x: point.x, y: point.y }] });
      await page.waitForTimeout(150 * 2 ** (press - 1));
      await cdp.send("Input.dispatchTouchEvent", { type: "touchEnd", touchPoints: [] });
    } else {
      await page.mouse.down();
      await waitForRecord(page, `${label}: showing press ${press} on End turn`, () => countRecords(page, isPointer) > pointers, 10_000);
      await page.mouse.up();
    }
    const deadline = Date.now() + 3_000;
    while (Date.now() < deadline && countRecords(page, isIntent) === intents && (await turnOf(page)) === 0) await page.waitForTimeout(25);
  }
  const produced = recordsOf(page).filter(isIntent).slice(intentsBefore);
  const strays = produced.filter(record => record.kind !== "end-turn");
  if (strays.length > 0) throw new Error(`${label}: presses on End turn produced ${strays.map(record => record.kind).join(", ")}`);
  if (produced.length < 2) throw new Error(`${label}: presses on End turn produced ${produced.length} end-turn intents, want the arming click and the confirming one`);
  await waitForTurn(page, 1, `${label}: presses on End turn did not end the turn`);
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
  // Departure must appear in the map's fixed strip, animate without another
  // campaign publication, and let planning continue while it is visible.
  for (const deviceScaleFactor of [1, 2]) {
    for (const reducedMotion of [false, true]) {
      const departureContext = await browser.newContext({ viewport: { width: 1280, height: 720 }, deviceScaleFactor });
      const departurePage = await departureContext.newPage();
      const departureFailures = await openGame(departureContext, departurePage);
      if (reducedMotion) {
        await pressGameKey(departurePage, "Escape");
        await pressGameKey(departurePage, "o");
        await departurePage.mouse.click(430, 355, { delay: 100 });
        await waitForRecord(departurePage, "departure reduced motion toggle", record => record.msg === "ui.intent" && record.kind === "toggle-reduced-motion");
        await pressGameKey(departurePage, "Escape");
        await pressGameKey(departurePage, "Escape");
      }
      await departurePage.mouse.move(50, 600);
      await departurePage.waitForTimeout(300);
      const initial = await departurePage.locator("html").getAttribute("data-africa2ice-summary");
      await pressGameKey(departurePage, "b");
      await departurePage.waitForFunction(previous => document.documentElement.dataset.africa2iceSummary !== previous, initial, { timeout: 5_000 });
      await waitForRecord(departurePage, "departure migration", record => record.msg === "action.dispatch" && record.command === "queue_migration");
      const queued = await departurePage.locator("html").getAttribute("data-africa2ice-summary");
      const departureClip = { x: 92, y: 156, width: 720, height: 176 };
      const first = await departurePage.screenshot({ clip: departureClip });
      if (process.env.AFRICA2ICE_DEPARTURE_PREVIEW_DIR && deviceScaleFactor === 1 && !reducedMotion) {
        await mkdir(process.env.AFRICA2ICE_DEPARTURE_PREVIEW_DIR, { recursive: true });
        await departurePage.screenshot({ path: resolve(process.env.AFRICA2ICE_DEPARTURE_PREVIEW_DIR, "departure.png") });
      }
      await departurePage.waitForTimeout(600);
      const second = await departurePage.screenshot({ clip: departureClip });
      if (first.equals(second) !== reducedMotion) throw new Error(`DPR ${deviceScaleFactor}: departure motion did not follow Reduced motion=${reducedMotion}`);
      if (await departurePage.locator("html").getAttribute("data-africa2ice-summary") !== queued) throw new Error("departure animation published a campaign change");
      if (reducedMotion) {
        await departurePage.waitForTimeout(3_600);
        const expired = await departurePage.screenshot({ clip: departureClip });
        if (second.equals(expired)) throw new Error("reduced-motion departure did not expire");
        await departurePage.waitForTimeout(300);
        if (!expired.equals(await departurePage.screenshot({ clip: departureClip }))) throw new Error("departure expiry left moving or stale pixels");
      } else {
        // Field Notes and End turn remain usable while the procession walks.
        await pressGameKey(departurePage, "f");
        if (await departurePage.locator("html").getAttribute("data-africa2ice-summary") !== queued) throw new Error("changing departure placement altered the campaign");
        await pressGameKey(departurePage, "Space");
        await waitForTurn(departurePage, 1, "departure blocked End turn");
      }
      if (departureFailures.length) throw new Error(departureFailures.join("\n"));
      await departureContext.close();
    }
  }

  // Camp must animate at both backing scales without publishing a campaign
  // change, then freeze under reduced motion and return via its mouse control.
  for (const deviceScaleFactor of [1, 2]) {
    const campContext = await browser.newContext({ viewport: { width: 1280, height: 720 }, deviceScaleFactor });
    const campPage = await campContext.newPage();
    const campFailures = await openGame(campContext, campPage);
    const summary = await campPage.locator("html").getAttribute("data-africa2ice-summary");
    const canvas = campPage.locator("canvas:not([id])");
    // The miniature lives at the bottom of the scrollable panel. Isolate its
    // pixels from map shimmer, then edit/discard without changing the frame.
    await pressGameKey(campPage, "PageDown");
    await pressGameKey(campPage, "PageDown");
    await campPage.mouse.move(1100, 550);
    await campPage.mouse.wheel(0, 600);
    await campPage.waitForTimeout(250);
    await campPage.mouse.move(50, 600);
    const miniatureClip = { x: 926, y: 452, width: 316, height: 178 };
    const miniature = await campPage.screenshot({ clip: miniatureClip });
    const miniatureSkyClip = { ...miniatureClip, height: 50 };
    const miniatureSky = await campPage.screenshot({ clip: miniatureSkyClip });
    await campPage.waitForTimeout(600);
    if (miniature.equals(await campPage.screenshot({ clip: miniatureClip }))) throw new Error(`DPR ${deviceScaleFactor}: Workforce miniature did not animate`);
    if (!miniatureSky.equals(await campPage.screenshot({ clip: miniatureSkyClip }))) throw new Error(`DPR ${deviceScaleFactor}: miniature stars did not stay steady`);
    await pressGameKey(campPage, "ArrowRight");
    await pressGameKey(campPage, "d");
    if (await campPage.locator("html").getAttribute("data-africa2ice-summary") !== summary) throw new Error("Workforce miniature or draft edit changed the campaign");
    await pressGameKey(campPage, "c");
    await campPage.waitForTimeout(250);
    const first = await canvas.screenshot();
    const starsClip = { x: 32, y: 120, width: 1216, height: 90 };
    const stars = await campPage.screenshot({ clip: starsClip });
    await campPage.waitForTimeout(600);
    if (first.equals(await canvas.screenshot())) throw new Error(`DPR ${deviceScaleFactor}: camp did not animate`);
    if (stars.equals(await campPage.screenshot({ clip: starsClip }))) throw new Error(`DPR ${deviceScaleFactor}: full-size stars did not twinkle`);
    for (const key of ["Space", "Tab", "n", "ArrowRight"]) await pressGameKey(campPage, key);
    if (await campPage.locator("html").getAttribute("data-africa2ice-summary") !== summary) throw new Error("camp keys changed the campaign");
    await pressGameKey(campPage, "Escape");
    await pressGameKey(campPage, "Escape");
    await pressGameKey(campPage, "o");
    await campPage.waitForTimeout(100);
    // Reduced motion is the second settings button; all points are CSS DIPs.
    await campPage.mouse.click(430, 355, { delay: 100 });
    await waitForRecord(campPage, "reduced motion toggle", record => record.msg === "ui.intent" && record.kind === "toggle-reduced-motion");
    await campPage.mouse.move(50, 600);
    await pressGameKey(campPage, "Escape");
    await pressGameKey(campPage, "Escape");
    await pressGameKey(campPage, "c");
    await campPage.waitForTimeout(200);
    const frozen = await canvas.screenshot();
    await campPage.waitForTimeout(600);
    if (!frozen.equals(await canvas.screenshot())) throw new Error(`DPR ${deviceScaleFactor}: reduced motion camp kept moving`);
    await pressGameKey(campPage, "Escape");
    await campPage.mouse.move(1100, 550);
    await campPage.mouse.wheel(0, 600);
    await campPage.waitForTimeout(250);
    await campPage.mouse.move(50, 600);
    const frozenMiniature = await campPage.screenshot({ clip: miniatureClip });
    await campPage.waitForTimeout(600);
    if (!frozenMiniature.equals(await campPage.screenshot({ clip: miniatureClip }))) throw new Error(`DPR ${deviceScaleFactor}: reduced motion Workforce miniature kept moving`);
    await pressGameKey(campPage, "c");
    const larger = { width: 1600, height: 1000 };
    await campPage.setViewportSize(larger);
    await campPage.waitForTimeout(200);
    const returnPoint = presentationPoint(larger, { x: 155, y: 656 });
    await campPage.mouse.click(returnPoint.x, returnPoint.y, { delay: 100 });
    await waitForRecord(campPage, "camp return button", record => record.msg === "ui.intent" && record.kind === "back");
    await pressGameKey(campPage, "Space");
    await waitForTurn(campPage, 1, "camp return button did not restore gameplay");
    if (campFailures.length) throw new Error(campFailures.join("\n"));
    await campContext.close();
  }

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
    await clickEndTurnUntilTheTurnEnds(hitPage, presentationPoint(viewport, endTurnDIP), touch, `DPR ${deviceScaleFactor}${touch ? " touch" : ""}`);
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
    await clickEndTurnUntilTheTurnEnds(resizePage, presentationPoint(larger, endTurnDIP), false, "resized to 1600x900");
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
  process.stdout.write("browser smoke passed: departure animation, frozen pose/expiry and live planning at DPR 1/2, camp animation, reduced motion, camp return/resize at DPR 1/2, boot, migration, turn, quick-save/reload, manual save/load, console logs, DPR 1/1.25/1.5/2/3 hits and backing, touch, resize, 200% zoom and portrait gating\n");
} finally {
  await browser.close();
  await new Promise(resolveClose => server.close(resolveClose));
}
