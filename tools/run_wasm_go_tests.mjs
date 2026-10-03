import { createServer } from "node:http";
import { copyFile, mkdtemp, readdir, readFile, rm, writeFile } from "node:fs/promises";
import { tmpdir } from "node:os";
import { join, relative, resolve } from "node:path";
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
// Browser-only tests are the Test functions declared in test files that build
// only for js. Suites are discovered rather than listed: a hand-kept filter let
// a new test that was never added to it skip silently while the run still
// passed. Each package with such files becomes one suite running exactly those
// tests, and every declared test must then report "=== RUN" or the run fails.
async function discoverSuites() {
  const suites = [];
  const walk = async directory => {
    const entries = await readdir(directory, { withFileTypes: true });
    const tests = [];
    for (const entry of entries) {
      if (entry.isDirectory()) {
        if (entry.name !== "testdata" && entry.name !== "node_modules") await walk(join(directory, entry.name));
        continue;
      }
      if (!entry.name.endsWith("_test.go")) continue;
      const source = await readFile(join(directory, entry.name), "utf8");
      if (!/^\/\/go:build js(\s*$|\s+&&)/m.test(source)) continue;
      for (const match of source.matchAll(/^func (Test\w+)\(t \*testing\.T\)/gm)) tests.push(match[1]);
    }
    if (tests.length === 0) return;
    const packagePath = `./${relative(root, directory)}`;
    suites.push({ packagePath, file: `${relative(root, directory).replaceAll("/", "_")}.test.wasm`, tests: tests.sort() });
  };
  for (const tree of ["internal", "pkg"]) await walk(join(root, tree));
  if (suites.length === 0) throw new Error("no browser-only test files found; the discovery pattern is broken");
  return suites;
}
const suites = await discoverSuites();

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
    const ran = new Set();
    page.on("console", message => {
      process.stdout.write(`${message.text()}\n`);
      for (const match of message.text().matchAll(/^=== RUN\s+(Test\w+)$/gm)) ran.add(match[1]);
      if (message.type() === "error") failures.push(`console.error: ${message.text()}`);
    });
    // Chromium can deliver a queued IndexedDB event after the Go test binary
    // has already called os.Exit. wasm_exec.js answers that late callback by
    // throwing "Go program has already exited", which is a property of the
    // browser's event ordering rather than of the code under test: the suite's
    // own verdict is the exit code and the console output collected above, and
    // both are complete by then. Treating this one message as a failure made
    // the gate flaky, so it is ignored and every other page error still fails.
    page.on("pageerror", error => {
      if (/Go program has already exited/.test(error.message)) return;
      failures.push(`pageerror: ${error.message}`);
    });
    await page.goto(`http://127.0.0.1:${server.address().port}/`, { waitUntil: "load" });
    await page.evaluate(async ({ file, tests }) => {
      const go = new Go();
      go.argv = [file, "-test.run", `^(${tests.join("|")})$`, "-test.v=true"];
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
    for (const test of suite.tests) {
      if (!ran.has(test)) failures.push(`${suite.packagePath}: declared browser test ${test} never ran`);
    }
    if (failures.length > 0) throw new Error(failures.join("\n"));
    await page.close();
  }
} finally {
  if (browser) await browser.close();
  if (server) await new Promise(resolveClose => server.close(resolveClose));
  await rm(working, { recursive: true, force: true });
}
