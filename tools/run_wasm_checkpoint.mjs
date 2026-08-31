import { createServer } from "node:http";
import { copyFile, mkdtemp, readFile, rm, writeFile } from "node:fs/promises";
import { tmpdir } from "node:os";
import { join, resolve } from "node:path";
import { spawnSync } from "node:child_process";
import { createRequire } from "node:module";

const root = resolve(import.meta.dirname, "..");
const require = createRequire(resolve(root, "tools/web-e2e/package.json"));
const { chromium } = require("playwright");
const working = await mkdtemp(join(tmpdir(), "africa2ice-checkpoint-"));
const wasmPath = join(working, "verification.test.wasm");

const run = (command, args, options = {}) => {
  const result = spawnSync(command, args, { cwd: root, encoding: "utf8", ...options });
  if (result.status !== 0) {
    throw new Error(`${command} ${args.join(" ")} failed\n${result.stdout ?? ""}${result.stderr ?? ""}`);
  }
  return (result.stdout ?? "").trim();
};

let browser;
let server;
try {
  const goEnv = { ...process.env, GOOS: "js", GOARCH: "wasm" };
  run("go", ["test", "-c", "-o", wasmPath, "./internal/verification"], { env: goEnv });
  const goRoot = run("go", ["env", "GOROOT"]);
  await copyFile(join(goRoot, "lib/wasm/wasm_exec.js"), join(working, "wasm_exec.js"));
  await writeFile(join(working, "index.html"), `<!doctype html>
<meta charset="utf-8"><script src="wasm_exec.js"></script><script>
const go = new Go();
go.argv = ["verification.test.wasm", "-test.run", "^TestEmitCheckpointForCrossTarget$", "-test.v=false"];
go.exit = code => { document.documentElement.dataset.exitCode = String(code); };
WebAssembly.instantiateStreaming(fetch("verification.test.wasm"), go.importObject)
  .then(result => go.run(result.instance))
  .then(() => { document.documentElement.dataset.done = "true"; })
  .catch(error => { console.error(error); document.documentElement.dataset.failed = "true"; });
</script>`);

  server = createServer(async (request, response) => {
    const name = request.url === "/verification.test.wasm" ? "verification.test.wasm" : request.url === "/wasm_exec.js" ? "wasm_exec.js" : "index.html";
    const body = await readFile(join(working, name));
    response.writeHead(200, { "Content-Type": name.endsWith(".wasm") ? "application/wasm" : name.endsWith(".js") ? "text/javascript" : "text/html" });
    response.end(body);
  });
  await new Promise(resolveListen => server.listen(0, "127.0.0.1", resolveListen));
  const address = server.address();
  browser = await chromium.launch({ headless: true });
  const page = await browser.newPage();
  const lines = [];
  const failures = [];
  page.on("console", message => {
    if (message.type() === "error") failures.push(`console.error: ${message.text()}`);
    lines.push(message.text());
  });
  page.on("pageerror", error => failures.push(`pageerror: ${error.message}`));
  await page.goto(`http://127.0.0.1:${address.port}/`, { waitUntil: "load" });
  await page.waitForFunction(() => document.documentElement.dataset.done === "true" || document.documentElement.dataset.failed === "true", null, { timeout: 30_000 });
  const exitCode = await page.locator("html").getAttribute("data-exit-code");
  if (exitCode !== "0") failures.push(`Go test exit code: ${exitCode ?? "missing"}`);
  if (failures.length > 0) throw new Error(failures.join("\n"));

  const begin = lines.filter(line => line === "AFRICA2ICE-CHECKPOINT-BEGIN");
  const end = lines.filter(line => line === "AFRICA2ICE-CHECKPOINT-END");
  if (begin.length !== 1 || end.length !== 1) throw new Error(`expected one checkpoint block, got begin=${begin.length} end=${end.length}`);
  const beginIndex = lines.indexOf(begin[0]);
  const endIndex = lines.indexOf(end[0]);
  if (endIndex !== beginIndex + 2) throw new Error("checkpoint block must contain exactly one canonical JSON line");
  const payload = lines[beginIndex + 1];
  JSON.parse(payload);
  await writeFile(resolve(root, "reference-checkpoints.json"), `${payload}\n`);
  process.stdout.write(`wrote reference-checkpoints.json (${Buffer.byteLength(payload)} bytes)\n`);
} finally {
  if (browser) await browser.close();
  if (server) await new Promise(resolveClose => server.close(resolveClose));
  await rm(working, { recursive: true, force: true });
}
