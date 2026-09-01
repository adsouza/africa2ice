import { createServer } from "node:http";
import { readFile, writeFile } from "node:fs/promises";
import { resolve } from "node:path";
import { createRequire } from "node:module";

const root = resolve(import.meta.dirname, "..");
const require = createRequire(resolve(root, "tools/web-e2e/package.json"));
const { chromium } = require("playwright");
const wasm = await readFile(resolve(root, "web/main.wasm"));
const wasmExec = await readFile(resolve(root, "web/wasm_exec.js"));
const html = Buffer.from(`<!doctype html>
<meta charset="utf-8"><script src="wasm_exec.js"></script><script>
globalThis.africa2iceCheckpointReady = (payload, error) => {
  globalThis.africa2iceCheckpointPayload = payload;
  if (error) {
    globalThis.africa2iceCheckpointError = error;
    document.documentElement.dataset.failed = "true";
  } else {
    document.documentElement.dataset.done = "true";
  }
};
const go = new Go();
go.exit = code => { document.documentElement.dataset.exitCode = String(code); };
WebAssembly.instantiateStreaming(fetch("main.wasm"), go.importObject)
  .then(result => go.run(result.instance))
  .catch(error => {
    globalThis.africa2iceCheckpointError = error.message;
    document.documentElement.dataset.failed = "true";
  });
</script>`);

let browser;
let server;
try {
  server = createServer((request, response) => {
    let body = html;
    let contentType = "text/html";
    if (request.url === "/main.wasm") {
      body = wasm;
      contentType = "application/wasm";
    } else if (request.url === "/wasm_exec.js") {
      body = wasmExec;
      contentType = "text/javascript";
    }
    response.writeHead(200, { "Content-Type": contentType });
    response.end(body);
  });
  await new Promise(resolveListen => server.listen(0, "127.0.0.1", resolveListen));
  const address = server.address();
  browser = await chromium.launch({ headless: true });
  const page = await browser.newPage();
  const failures = [];
  const pageErrors = [];
  page.on("console", message => {
    if (message.type() === "error") failures.push(`console.error: ${message.text()}`);
  });
  page.on("pageerror", error => pageErrors.push(error.message));
  await page.goto(`http://127.0.0.1:${address.port}/?checkpoint=1`, { waitUntil: "load" });
  await page.waitForFunction(() => {
    const state = document.documentElement.dataset;
    return state.failed === "true" || (state.done === "true" && state.exitCode !== undefined);
  }, null, { timeout: 30_000 });
  const result = await page.evaluate(() => ({
    payload: globalThis.africa2iceCheckpointPayload,
    error: globalThis.africa2iceCheckpointError,
    exitCode: document.documentElement.dataset.exitCode,
  }));
  if (result.error) failures.push(`checkpoint: ${result.error}`);
  if (result.exitCode !== "0") failures.push(`Go checkpoint exit code: ${result.exitCode ?? "missing"}`);
  let validPayload = false;
  if (typeof result.payload !== "string" || result.payload.length === 0) {
    failures.push("optimized module produced no checkpoint payload");
  } else {
    try {
      JSON.parse(result.payload);
      validPayload = true;
    } catch (error) {
      failures.push(`invalid checkpoint payload: ${error.message}`);
    }
  }

  // Go may leave a scheduler callback queued while a short-lived WASM main
  // returns. Newer browser/Node timing can dispatch it after the clean exit,
  // at which point wasm_exec.js reports this exact message. It is benign only
  // after the checkpoint and zero exit code have both been validated.
  await page.waitForTimeout(100);
  for (const message of pageErrors) {
    const expectedAfterCleanExit = validPayload && result.exitCode === "0" && message === "Go program has already exited";
    if (!expectedAfterCleanExit) failures.push(`pageerror: ${message}`);
  }
  if (failures.length > 0) throw new Error(failures.join("\n"));
  await writeFile(resolve(root, "reference-checkpoints.json"), `${result.payload}\n`);
  process.stdout.write(`wrote reference-checkpoints.json from optimized web/main.wasm (${Buffer.byteLength(result.payload)} bytes)\n`);
} finally {
  if (browser) await browser.close();
  if (server) await new Promise(resolveClose => server.close(resolveClose));
}
