import { test, expect } from '@playwright/test';
import { login } from '../fixtures/auth.fixture';

test.describe('M4: Dashboard', () => {

  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  test.describe('TC-C2-M4-001: Stats Cards and Recent Agents Load', () => {
    test('should display 4 stat cards and recent agents table', async ({ page }) => {
      await page.goto('/');
      await page.waitForSelector('.stat-card', { timeout: 15000 });

      // Should have 4 stat cards
      const statCards = page.locator('.stat-card');
      await expect(statCards).toHaveCount(4);

      // Verify stat card titles
      await expect(page.locator('.stat-card').nth(0)).toContainText('Total Agents');
      await expect(page.locator('.stat-card').nth(1)).toContainText('Active Agents');
      await expect(page.locator('.stat-card').nth(2)).toContainText('Listeners Running');
      await expect(page.locator('.stat-card').nth(3)).toContainText('Tasks Today');

      // Verify "Recent Agents" section exists
      await expect(page.locator('.card-title').filter({ hasText: 'Recent Agents' })).toBeVisible();

      // Should show either agents table or empty state message
      const hasAgents = await page.locator('table').count();
      const hasEmptyMsg = await page.locator('.text-muted').filter({ hasText: /No agents connected/ }).count();
      expect(hasAgents + hasEmptyMsg).toBeGreaterThan(0);
    });
  });

  test.describe('TC-C2-M4-002: Dashboard Polling Updates', () => {
    test('should poll dashboard stats every ~15 seconds', async ({ page }) => {
      let requestCount = 0;
      page.on('request', (req) => {
        if (req.url().includes('/dashboard/stats')) {
          requestCount++;
        }
      });

      await page.goto('/');
      await page.waitForSelector('.stat-card', { timeout: 15000 });

      // Wait for 16 seconds (15s poll + 1s buffer)
      await page.waitForTimeout(16000);

      // Should have at least 2 requests (initial + one poll)
      expect(requestCount).toBeGreaterThan(1);
      console.log(`[TC-C2-M4-002] /dashboard/stats requests in 16s: ${requestCount}`);
    });
  });

  test.describe('TC-C2-M4-003: Dashboard Handles API Error Gracefully', () => {
    test('should not crash when dashboard API fails', async ({ page }) => {
      // Intercept the stats endpoint and return 500
      await page.route('**/dashboard/stats', (route) => {
        route.fulfill({
          status: 500,
          contentType: 'application/json',
          body: JSON.stringify({ error: { code: 'INTERNAL', message: 'Server error' } }),
        });
      });

      await page.goto('/');

      // App should not crash — sidebar should still be visible
      await expect(page.locator('.sidebar')).toBeVisible({ timeout: 10000 });

      // Spinner may keep showing or empty state — either is acceptable
      // The key assertion: no uncaught error, app structure intact
      await expect(page.locator('.header')).toBeVisible();
    });
  });
});
