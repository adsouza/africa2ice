import { createServer } from "node:http";
import { copyFile, mkdtemp, readFile, rm, writeFile } from "node:fs/promises";
import { tmpdir } from "node:os";
import { join, resolve } from "node:path";
import { spawnSync } from "node:child_process";
import { createRequire } from "node:module";

const root = resolve(import.meta.dirname, "..");
const require = createRequire(resolve(root, "tools/web-e2e/package.json"));
const { chromium } = require("playwright");
const working = await mkdtemp(join(tmpdir(), "africa2ice-go-tests-"));
const run = (command, args, options = {}) => {
  const result = spawnSync(command, args, { cwd: root, encoding: "utf8", ...options });
  if (result.status !== 0) throw new Error(`${command} ${args.join(" ")} failed\n${result.stdout ?? ""}${result.stderr ?? ""}`);
  return (result.stdout ?? "").trim();
};
const suites = [
  { packagePath: "./internal/adapters/storage", file: "storage.test.wasm", test: "TestIndexedDBRepositoryBrowserContract" },
  { packagePath: "./pkg/ui", file: "ui.test.wasm", test: "TestIndexedDBUISettingsStoreBrowserContract" },
];

let browser;
let server;
try {
  for (const suite of suites) {
    run("go", ["test", "-c", "-o", join(working, suite.file), suite.packagePath], { env: { ...process.env, GOOS: "js", GOARCH: "wasm" } });
  }
  const goRoot = run("go", ["env", "GOROOT"]);
  await copyFile(join(goRoot, "lib/wasm/wasm_exec.js"), join(working, "wasm_exec.js"));
  await writeFile(join(working, "index.html"), "<!doctype html><meta charset=\"utf-8\"><script src=\"wasm_exec.js\"></script>");
  server = createServer(async (request, response) => {
    const requested = request.url.slice(1);
    const name = requested === "wasm_exec.js" || suites.some(suite => suite.file === requested) ? requested : "index.html";
    const body = await readFile(join(working, name));
    response.writeHead(200, { "Content-Type": name.endsWith(".wasm") ? "application/wasm" : name.endsWith(".js") ? "text/javascript" : "text/html" });
    response.end(body);
  });
  await new Promise(resolveListen => server.listen(0, "127.0.0.1", resolveListen));
  browser = await chromium.launch({ headless: true });

  for (const suite of suites) {
    const page = await browser.newPage();
    const failures = [];
    page.on("console", message => {
      process.stdout.write(`${message.text()}\n`);
      if (message.type() === "error") failures.push(`console.error: ${message.text()}`);
    });
    page.on("pageerror", error => failures.push(`pageerror: ${error.message}`));
    await page.goto(`http://127.0.0.1:${server.address().port}/`, { waitUntil: "load" });
    await page.evaluate(async ({ file, test }) => {
      const go = new Go();
      go.argv = [file, "-test.run", `^${test}$`, "-test.v=true"];
      go.exit = code => { document.documentElement.dataset.exitCode = String(code); };
      try {
        const result = await WebAssembly.instantiateStreaming(fetch(file), go.importObject);
        await go.run(result.instance);
        document.documentElement.dataset.done = "true";
      } catch (error) {
        console.error(error);
        document.documentElement.dataset.failed = "true";
      }
    }, suite);
    await page.waitForFunction(() => document.documentElement.dataset.done === "true" || document.documentElement.dataset.failed === "true", null, { timeout: 30_000 });
    const exitCode = await page.locator("html").getAttribute("data-exit-code");
    if (exitCode !== "0") failures.push(`Go test exit code for ${suite.packagePath}: ${exitCode || "missing"}`);
    if (failures.length > 0) throw new Error(failures.join("\n"));
    await page.close();
  }
} finally {
  if (browser) await browser.close();
  if (server) await new Promise(resolveClose => server.close(resolveClose));
  await rm(working, { recursive: true, force: true });
}
