import assert from "node:assert/strict";
import { createServer } from "node:http";
import { readFile } from "node:fs/promises";
import { resolve } from "node:path";
import { chromium } from "playwright";

const webRoot = resolve(import.meta.dirname, "../../web");
const server = createServer(async (request, response) => {
  const name = request.url === "/" ? "index.html" : request.url.slice(1);
  if (!["index.html", "wasm_exec.js", "main.wasm"].includes(name)) {
    response.writeHead(404).end();
    return;
  }
  response.writeHead(200, { "Content-Type": name.endsWith("wasm") ? "application/wasm" : name.endsWith("js") ? "text/javascript" : "text/html" });
  response.end(await readFile(resolve(webRoot, name)));
});
await new Promise(ready => server.listen(0, "127.0.0.1", ready));
const browser = await chromium.launch({ headless: true });
const context = await browser.newContext({ viewport: { width: 1280, height: 720 } });
const page = await context.newPage();
const errors = [];
page.on("pageerror", error => errors.push(String(error)));

// Observe real mixed PCM sent to Oto's worklet, before transferring its buffer.
// The observer does not replace the device, audio context, or game controls.
await page.addInitScript(() => {
  window.audioProbe = { contexts: 0, blocks: [], since: 0, focusEvents: [] };
  for (const event of ["blur", "focus", "visibilitychange"]) {
    (event === "visibilitychange" ? document : window).addEventListener(event, () => {
      window.audioProbe.focusEvents.push({ event, hidden: document.hidden, focused: document.hasFocus() });
    });
  }
  const Context = window.AudioContext;
  window.AudioContext = class extends Context {
    constructor(...args) { super(...args); window.audioProbe.contexts++; }
  };
  const Worklet = window.AudioWorkletNode;
  window.AudioWorkletNode = class extends Worklet {
    constructor(...args) {
      super(...args);
      const post = this.port.postMessage.bind(this.port);
      this.port.postMessage = (...message) => {
        const samples = message[0];
        if (samples instanceof Float32Array) {
          let peak = 0, energy = 0;
          for (const value of samples) { peak = Math.max(peak, Math.abs(value)); energy += value*value; }
          window.audioProbe.blocks.push({ time: performance.now(), peak, rms: Math.sqrt(energy/samples.length) });
          if (window.audioProbe.blocks.length > 200) window.audioProbe.blocks.shift();
        }
        return post(...message);
      };
    }
  };
});

const ready = async () => {
  await page.waitForFunction(() => document.documentElement.dataset.africa2iceReady === "true", null, { timeout: 15_000 });
};
const press = async key => {
  await page.keyboard.press(key, { delay: 150 });
  await page.waitForTimeout(100); // Give Ebiten a frame to observe key-up.
};
const mark = () => page.evaluate(() => { window.audioProbe.since = performance.now(); });
const silence = async () => {
  await page.waitForFunction(() => {
    const blocks = window.audioProbe.blocks.filter(block => block.time > window.audioProbe.since+300).slice(-6);
    return blocks.length === 6 && blocks.every(block => block.peak === 0);
  }, null, { timeout: 15_000 });
};
const audible = async (delay = 300) => {
  await page.waitForFunction(delay => window.audioProbe.blocks.some(block => block.time > window.audioProbe.since+delay && block.rms > 0.004), delay, { timeout: 15_000 });
};

try {
  await page.goto(`http://127.0.0.1:${server.address().port}/`);
  await ready();
  assert.equal(await page.evaluate(() => window.audioProbe.contexts), 0, "startup opened an audio device before a gesture");
  await mark();
  await press("Enter");
  await audible(3500); // Listen past the click effect and the music introduction.
  assert.equal(await page.evaluate(() => window.audioProbe.contexts), 1);

  await press("m");
  await mark();
  await silence();
  // Reload while muted to exercise the persisted preference on first unlock.
  await page.reload();
  await ready();
  assert.equal(await page.evaluate(() => window.audioProbe.contexts), 0);
  await press("Enter");
  await mark();
  await silence();
  await press("m");
  await mark();
  await audible();

  // Existing keyboard volume controls in Settings must also govern music.
  await press("Escape");
  await press("o");
  for (let i = 0; i < 6; i++) await press("-"); // Clamp past floating-point zero.
  await mark();
  await silence();
  for (let i = 0; i < 5; i++) await press("=");
  await mark();
  await audible();

  // Chromium automation reports every tab focused. Drive the real listener
  // with a visibility fixture; observe the resulting PCM through the worklet.
  await page.evaluate(() => {
    Object.defineProperty(document, "hidden", { value: true, configurable: true });
    document.dispatchEvent(new Event("visibilitychange"));
  });
  await mark();
  await silence();
  await page.evaluate(() => {
    Reflect.deleteProperty(document, "hidden");
    document.dispatchEvent(new Event("visibilitychange"));
  });
  await mark();
  await audible();
  assert.equal(await page.evaluate(() => window.audioProbe.contexts), 1, "controls or focus created a second device");
  assert.deepEqual(errors, [], "browser raised an audio or runtime error");
  console.log("audio browser checks passed: gesture startup, persisted mute, volume, visibility-event pause/resume, one context");
} catch (error) {
  console.error(await page.evaluate(() => ({ hidden: document.hidden, focused: document.hasFocus(), since: window.audioProbe.since, blocks: window.audioProbe.blocks.slice(-8), focusEvents: window.audioProbe.focusEvents })));
  throw error;
} finally {
  await browser.close();
  await new Promise(done => server.close(done));
}
