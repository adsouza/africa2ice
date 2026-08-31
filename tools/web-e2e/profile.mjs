import { createServer } from "node:http";
import { readFile, stat, writeFile } from "node:fs/promises";
import { arch, cpus, freemem, platform, release, totalmem } from "node:os";
import { extname, resolve } from "node:path";
import { chromium } from "playwright";

const root = resolve(import.meta.dirname, "../..");
const webRoot = resolve(root, "web");
const profileSave = await readFile(resolve(root, "testdata/performance_profile_save.json"), "utf8");
const profileState = JSON.parse(profileSave);
if (profileState.turn !== 300 || profileState.bands?.length !== 256) throw new Error("maximum-render save fixture must be turn 300 with 256 bands");
for (const name of ["index.html", "main.wasm", "wasm_exec.js"]) {
  const info = await stat(resolve(webRoot, name));
  if (!info.isFile() || info.size === 0) throw new Error(`staged web artifact ${name} is missing or empty`);
}

const mime = new Map([[".html", "text/html; charset=utf-8"], [".js", "text/javascript; charset=utf-8"], [".wasm", "application/wasm"]]);
const server = createServer(async (request, response) => {
  const url = new URL(request.url, "http://127.0.0.1");
  const name = url.pathname === "/" ? "index.html" : url.pathname.slice(1);
  if (!new Set(["index.html", "main.wasm", "wasm_exec.js"]).has(name)) return response.writeHead(404).end();
  const body = await readFile(resolve(webRoot, name));
  response.writeHead(200, { "Content-Type": mime.get(extname(name)), "Cache-Control": "no-store" });
  response.end(body);
});
await new Promise(resolveListen => server.listen(0, "127.0.0.1", resolveListen));
const origin = `http://127.0.0.1:${server.address().port}`;
const browser = await chromium.launch({ headless: true });
const allConfigurations = [
  { dpr: 1, minimumFPS: 30 },
  { dpr: 2, minimumFPS: 20 },
];
const selectedConfiguration = process.env.A2I_PROFILE_CONFIGURATION;
const configurations = selectedConfiguration
  ? allConfigurations.filter(configuration => `dpr${configuration.dpr}` === selectedConfiguration)
  : allConfigurations;
if (configurations.length === 0) throw new Error(`unknown A2I_PROFILE_CONFIGURATION ${selectedConfiguration}`);

const percentile = (values, fraction) => {
  const sorted = [...values].sort((a, b) => a - b);
  return sorted[Math.min(sorted.length - 1, Math.floor(sorted.length * fraction))];
};

const profiles = [];
try {
  for (const configuration of configurations) {
    const context = await browser.newContext({ viewport: { width: 1280, height: 720 }, deviceScaleFactor: configuration.dpr });
    const page = await context.newPage();
    const failures = [];
    page.on("pageerror", error => failures.push(`pageerror: ${error.message}`));
    page.on("console", message => { if (message.type() === "error") failures.push(`console.error: ${message.text()}`); });
    await page.addInitScript(save => {
      window.africa2iceE2EObserver = () => {};
      window.africa2iceRenderProfileSave = save;
    }, profileSave);
    await page.goto(`${origin}/?e2e=1&profile=1`, { waitUntil: "load" });
    await page.waitForFunction(() => document.documentElement.dataset.africa2iceReady === "true", null, { timeout: 10_000 });
    await page.waitForTimeout(5_000);

    const gapsPromise = page.evaluate(async () => {
      const gaps = [];
      const started = performance.now();
      let previous = started;
      await new Promise(resolveSample => {
        const frame = now => {
          gaps.push(now - previous);
          previous = now;
          if (now - started >= 30_000) resolveSample(); else requestAnimationFrame(frame);
        };
        requestAnimationFrame(frame);
      });
      return gaps.slice(1);
    });
    const turnLatencies = [];
    for (let index = 0; index < 6; index++) {
      await page.waitForTimeout(5_000);
      const before = JSON.parse(await page.locator("html").getAttribute("data-africa2ice-summary")).turn;
      const started = performance.now();
      await page.keyboard.press("Space");
      await page.waitForFunction(turn => JSON.parse(document.documentElement.dataset.africa2iceSummary).turn === turn + 1, before, { timeout: 2_000 });
      turnLatencies.push(performance.now() - started);
    }
    const gaps = await gapsPromise;
    const medianGap = percentile(gaps, 0.5);
    const result = {
      view: "top-down",
      dpr: configuration.dpr,
      median_fps: 1000 / medianGap,
      p95_frame_gap_ms: percentile(gaps, 0.95),
      maximum_turn_latency_ms: Math.max(...turnLatencies),
      js_heap_bytes: await page.evaluate(() => performance.memory?.usedJSHeapSize ?? null),
    };
    process.stdout.write(`${JSON.stringify(result)}\n`);
    if (failures.length > 0) throw new Error(failures.join("\n"));
    if (result.median_fps < configuration.minimumFPS) throw new Error(`top-down DPR${configuration.dpr} median FPS ${result.median_fps} is below ${configuration.minimumFPS}`);
    if (result.p95_frame_gap_ms > 150) throw new Error(`top-down DPR${configuration.dpr} p95 gap ${result.p95_frame_gap_ms}ms exceeds 150ms`);
    if (result.maximum_turn_latency_ms > 2_000) throw new Error(`top-down DPR${configuration.dpr} turn latency exceeds 2s`);
    profiles.push(result);
    await context.close();
  }
  const report = {
    browser: browser.version(),
    machine: { arch: arch(), cpu: cpus()[0]?.model ?? "unknown", logical_cpus: cpus().length, free_memory_bytes: freemem(), os: `${platform()} ${release()}`, total_memory_bytes: totalmem() },
    profiles,
    sample_seconds: 30,
    warmup_seconds: 5,
  };
  await writeFile(resolve(root, "performance-profile.json"), `${JSON.stringify(report, null, 2)}\n`);
  process.stdout.write(`${JSON.stringify(report, null, 2)}\n`);
} finally {
  await browser.close();
  await new Promise(resolveClose => server.close(resolveClose));
}
