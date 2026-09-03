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

const openGame = async (context, page) => {
  const failures = [];
  page.on("pageerror", error => failures.push(`pageerror: ${error.message}`));
  page.on("console", message => {
    if (message.type() === "error") failures.push(`console.error: ${message.text()}`);
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

  for (const deviceScaleFactor of [1, 1.25, 3]) {
    const dpiContext = await browser.newContext({ viewport: { width: 1280, height: 720 }, deviceScaleFactor });
    const dpiPage = await dpiContext.newPage();
    await openGame(dpiContext, dpiPage);
    await dpiContext.close();
  }
  process.stdout.write("browser smoke passed: boot, migration, turn, quick-save/reload, manual save/load, DPR 1/1.25/2/3\n");
} finally {
  await browser.close();
  await new Promise(resolveClose => server.close(resolveClose));
}
