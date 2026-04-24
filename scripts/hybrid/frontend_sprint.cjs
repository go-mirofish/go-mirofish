/* Browser-based checks against the Vite dev server (or any FRONTEND_URL). */
const fs = require("fs");
const path = require("path");

const artifactPath = process.env.ARTIFACT || path.join(__dirname, "frontend-sprint-out.json");
const startUrl = process.env.FRONTEND_URL || "http://127.0.0.1:3000/";

let chromium;
try {
  ({ chromium } = require("playwright"));
} catch (e) {
  console.error(
    JSON.stringify({
      mode: "frontend_e2e",
      ok: false,
      fatal: "playwright package not installed; run: npm install playwright && npx playwright install chromium",
    })
  );
  process.exit(1);
}

(async () => {
  const out = {
    mode: "frontend_e2e",
    ok: true,
    start_url: startUrl,
    page_load_ms: null,
    title: null,
    network_errors: [],
    page_errors: [],
    docs_reachable: false,
  };
  const browser = await chromium.launch({ headless: true });
  const page = await browser.newPage();
  const t0 = Date.now();
  page.on("response", (res) => {
    const u = res.url();
    if (res.status() >= 400 && (u.includes("/api/") || u.endsWith(".html") || u.endsWith("/") )) {
      out.network_errors.push({ url: u, status: res.status() });
    }
  });
  page.on("pageerror", (err) => {
    out.page_errors.push(String(err));
  });
  try {
    await page.goto(startUrl, { waitUntil: "domcontentloaded", timeout: 120000 });
    out.page_load_ms = Date.now() - t0;
    out.title = await page.title();
    const docLink = page.locator('a[href*="/docs"]');
    const n = await docLink.count();
    if (n > 0) {
      await docLink.first().click({ timeout: 20000 });
      await page.waitForURL(/\/docs/, { timeout: 30000 });
      out.docs_reachable = true;
    }
  } catch (e) {
    out.ok = false;
    out.fatal = String(e);
  }
  if (out.network_errors.length) {
    out.ok = false;
  }
  await browser.close();
  const text = JSON.stringify(out, null, 2);
  fs.writeFileSync(artifactPath, text, "utf8");
  console.log(text);
  process.exit(out.ok && !out.fatal ? 0 : 1);
})();
