import { chromium } from '@playwright/test';

/**
 * Global setup: verify backend and frontend are reachable before running tests.
 */
export default async function globalSetup() {
  console.log('[Global Setup] Checking backend health...');
  const browser = await chromium.launch();
  const context = await browser.newContext({ ignoreHTTPSErrors: true });
  const page = await context.newPage();

  try {
    // Check backend health endpoint
    const healthResp = await page.request.get(
      'http://localhost:5173/api/v1/health'
    );
    if (!healthResp.ok()) {
      console.warn(
        `[Global Setup] WARNING: Backend health check returned ${healthResp.status()}. ` +
        'Ensure teamserver is running on https://localhost:8443'
      );
    } else {
      const body = await healthResp.json();
      console.log(`[Global Setup] Backend health: ${JSON.stringify(body)}`);
    }
  } catch (err) {
    console.warn(
      '[Global Setup] WARNING: Could not reach backend. ' +
      'Ensure teamserver is running (`make dev` or `./bin/teamserver`).'
    );
  }

  // Check frontend
  try {
    await page.goto('http://localhost:5173', { waitUntil: 'domcontentloaded' });
    console.log('[Global Setup] Frontend is reachable at http://localhost:5173');
  } catch (err) {
    console.warn(
      '[Global Setup] WARNING: Could not reach frontend. ' +
      'Ensure `npm run dev` is running in web/ directory.'
    );
  }

  await context.close();
  await browser.close();
  console.log('[Global Setup] Complete.\n');
}
