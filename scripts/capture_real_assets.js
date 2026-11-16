const fs = require('fs');
const path = require('path');
const { execSync, spawn } = require('child_process');
const puppeteer = require('puppeteer-core');

const CHROME_PATH = 'C:\\Program Files\\Google\\Chrome\\Application\\chrome.exe';
const OUTPUT_DIR = path.join(__dirname, '..', 'docs', 'images');
const FRAMES_DIR = path.join(__dirname, '..', 'docs', 'images', 'frames');

async function sleep(ms) {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

async function populateTelemetry() {
  console.log('[capture] Populating SQLite database with authentic tool calls...');
  const benchPath = path.join(__dirname, '..', 'examples', 'failing_invocations.jsonl');
  const lines = fs.readFileSync(benchPath, 'utf8').split('\n').filter(Boolean);

  for (const line of lines.slice(0, 35)) {
    try {
      const item = JSON.parse(line);
      const tool = item.tool || item.name;
      const args = item.arguments;

      await fetch('http://127.0.0.1:7171/v1/tools/execute', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ tool, arguments: args }),
      });
    } catch (e) {
      // ignore
    }
  }
  console.log('[capture] Telemetry populated with real calls.');
}

async function main() {
  if (!fs.existsSync(OUTPUT_DIR)) fs.mkdirSync(OUTPUT_DIR, { recursive: true });
  if (!fs.existsSync(FRAMES_DIR)) fs.mkdirSync(FRAMES_DIR, { recursive: true });

  // 1. Populate telemetry
  await populateTelemetry();

  // 2. Launch Puppeteer with Chrome
  console.log('[capture] Launching Chrome via Puppeteer-core...');
  const browser = await puppeteer.launch({
    executablePath: CHROME_PATH,
    headless: 'new',
    args: ['--no-sandbox', '--disable-setuid-sandbox', '--window-size=1440,960'],
  });

  const page = await browser.newPage();
  await page.setViewport({ width: 1440, height: 900, deviceScaleFactor: 2 });

  console.log('[capture] Navigating to http://127.0.0.1:7171...');
  await page.goto('http://127.0.0.1:7171', { waitUntil: 'domcontentloaded' });
  await sleep(2500);

  // Screenshot 1: Desktop Dashboard Overview
  console.log('[capture] Taking screenshot: dashboard_desktop.png...');
  await page.screenshot({
    path: path.join(OUTPUT_DIR, 'dashboard_desktop.png'),
    fullPage: false,
  });

  // Screenshot 2: Invocations with an expanded card
  console.log('[capture] Expanding first invocation item...');
  const firstInv = await page.$('.space-y-2 > div:first-child');
  if (firstInv) {
    await firstInv.click();
    await sleep(500);
  }
  await page.screenshot({
    path: path.join(OUTPUT_DIR, 'invocations_stream.png'),
    fullPage: false,
  });

  // Screenshot 3: Interactive Sandbox Tab
  console.log('[capture] Switching to Interactive Sandbox tab...');
  const sandboxTabBtn = await page.evaluateHandle(() => {
    const buttons = Array.from(document.querySelectorAll('button'));
    return buttons.find((b) => b.textContent.includes('Interactive Sandbox'));
  });
  if (sandboxTabBtn) {
    await sandboxTabBtn.click();
    await sleep(800);

    // Click "Heal Parameters"
    const healBtn = await page.evaluateHandle(() => {
      const buttons = Array.from(document.querySelectorAll('button'));
      return buttons.find((b) => b.textContent.includes('Heal Parameters'));
    });
    if (healBtn) {
      await healBtn.click();
      await sleep(1200);
    }

    await page.screenshot({
      path: path.join(OUTPUT_DIR, 'sandbox_healed.png'),
      fullPage: false,
    });
  }

  // Screenshot 4: Schema Drift Analytics Tab
  console.log('[capture] Switching to Schema Drift Analytics tab...');
  const driftsTabBtn = await page.evaluateHandle(() => {
    const buttons = Array.from(document.querySelectorAll('button'));
    return buttons.find((b) => b.textContent.includes('Schema Drift'));
  });
  if (driftsTabBtn) {
    await driftsTabBtn.click();
    await sleep(800);
    await page.screenshot({
      path: path.join(OUTPUT_DIR, 'schema_drifts.png'),
      fullPage: false,
    });
  }

  // Screenshot 5: Tool Schema Catalog Tab
  console.log('[capture] Switching to Tool Catalog tab...');
  const catalogTabBtn = await page.evaluateHandle(() => {
    const buttons = Array.from(document.querySelectorAll('button'));
    return buttons.find((b) => b.textContent.includes('Tool Schema Catalog'));
  });
  if (catalogTabBtn) {
    await catalogTabBtn.click();
    await sleep(800);
    await page.screenshot({
      path: path.join(OUTPUT_DIR, 'tool_catalog.png'),
      fullPage: false,
    });
  }

  // Screenshot 6: Responsive Mobile Viewport (iPhone 14/15 size: 390x844)
  console.log('[capture] Taking mobile responsive screenshot (390x844)...');
  await page.setViewport({ width: 390, height: 844, deviceScaleFactor: 2, isMobile: true, hasTouch: true });
  // Switch back to live stream on mobile
  const liveTabMobile = await page.evaluateHandle(() => {
    const buttons = Array.from(document.querySelectorAll('button'));
    return buttons.find((b) => b.textContent.includes('Live Telemetry'));
  });
  if (liveTabMobile) {
    await liveTabMobile.click();
    await sleep(800);
  }
  await page.screenshot({
    path: path.join(OUTPUT_DIR, 'mobile_responsive.png'),
    fullPage: false,
  });

  // 3. Capture animated frames for GIF demo
  console.log('[capture] Capturing frames for toolheal_demo.gif...');
  await page.setViewport({ width: 1280, height: 760, deviceScaleFactor: 1 });
  await page.goto('http://127.0.0.1:7171', { waitUntil: 'domcontentloaded' });
  await sleep(1500);

  let frameIdx = 0;
  async function grabFrame() {
    const num = String(frameIdx).padStart(4, '0');
    await page.screenshot({ path: path.join(FRAMES_DIR, `frame_${num}.png`) });
    frameIdx++;
  }

  // Phase A: Dashboard Overview (15 frames)
  for (let i = 0; i < 15; i++) {
    await grabFrame();
    await sleep(100);
  }

  // Phase B: Switch to Sandbox
  const sandboxTab = await page.evaluateHandle(() => {
    const buttons = Array.from(document.querySelectorAll('button'));
    return buttons.find((b) => b.textContent.includes('Interactive Sandbox'));
  });
  if (sandboxTab) {
    await sandboxTab.click();
    for (let i = 0; i < 10; i++) {
      await grabFrame();
      await sleep(100);
    }

    // Click "Heal Parameters"
    const healBtn = await page.evaluateHandle(() => {
      const buttons = Array.from(document.querySelectorAll('button'));
      return buttons.find((b) => b.textContent.includes('Heal Parameters'));
    });
    if (healBtn) {
      await healBtn.click();
      for (let i = 0; i < 18; i++) {
        await grabFrame();
        await sleep(100);
      }
    }
  }

  // Phase C: Switch to Schema Drifts Tab
  const driftsTab = await page.evaluateHandle(() => {
    const buttons = Array.from(document.querySelectorAll('button'));
    return buttons.find((b) => b.textContent.includes('Schema Drift'));
  });
  if (driftsTab) {
    await driftsTab.click();
    for (let i = 0; i < 15; i++) {
      await grabFrame();
      await sleep(100);
    }
  }

  // Phase D: Open Benchmark Runner Modal
  const benchBtn = await page.evaluateHandle(() => {
    const buttons = Array.from(document.querySelectorAll('button'));
    return buttons.find((b) => b.textContent.includes('Run Benchmark'));
  });
  if (benchBtn) {
    await benchBtn.click();
    await sleep(300);
    for (let i = 0; i < 8; i++) {
      await grabFrame();
      await sleep(100);
    }

    // Click "Start Benchmark Run"
    const startBenchBtn = await page.evaluateHandle(() => {
      const buttons = Array.from(document.querySelectorAll('button'));
      return buttons.find((b) => b.textContent.includes('Start Benchmark Run'));
    });
    if (startBenchBtn) {
      await startBenchBtn.click();
      for (let i = 0; i < 20; i++) {
        await grabFrame();
        await sleep(100);
      }
    }
  }

  await browser.close();
  console.log(`[capture] Captured ${frameIdx} frames. Compiling with ffmpeg...`);

  // Compile frames to GIF using FFmpeg palettegen/paletteuse
  const gifPath = path.join(OUTPUT_DIR, 'toolheal_demo.gif');
  const palettePath = path.join(OUTPUT_DIR, 'palette.png');
  const framesPattern = path.join(FRAMES_DIR, 'frame_%04d.png');

  try {
    // Generate palette
    execSync(
      `ffmpeg -y -framerate 10 -i "${framesPattern}" -vf "fps=10,scale=1000:-1:flags=lanczos,palettegen=stats_mode=diff" "${palettePath}"`,
      { stdio: 'inherit' }
    );
    // Render GIF using palette
    execSync(
      `ffmpeg -y -framerate 10 -i "${framesPattern}" -i "${palettePath}" -lavfi "fps=10,scale=1000:-1:flags=lanczos [x]; [x][1:v] paletteuse=dither=bayer:bayer_scale=3" "${gifPath}"`,
      { stdio: 'inherit' }
    );
    // Cleanup palette and frame images
    fs.unlinkSync(palettePath);
    const frameFiles = fs.readdirSync(FRAMES_DIR);
    for (const f of frameFiles) {
      fs.unlinkSync(path.join(FRAMES_DIR, f));
    }
    fs.rmdirSync(FRAMES_DIR);
    console.log(`✓ Generated animated demo GIF at: ${gifPath}`);
  } catch (err) {
    console.error('Error compiling GIF with ffmpeg:', err);
  }
}

main().catch(console.error);
