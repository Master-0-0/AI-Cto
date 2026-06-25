import { test, expect } from '@playwright/test';
import { login } from '../fixtures/auth.fixture';
import { apiRequest } from '../helpers/api';

test.describe('M10: Build Panel', () => {

  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  test.describe('TC-C2-M10-001: Build Panel Renders and Shows Stub Message', () => {
    test('should display OS/Arch selectors and stub message on build click', async ({ page }) => {
      await page.goto('/build');
      await page.waitForSelector('.page-title');

      await expect(page.locator('.page-title')).toContainText('Build Implant');

      // Verify select dropdowns exist
      const selects = page.locator('.form-select');
      await expect(selects).toHaveCount(2);

      // Select an OS and ARCH
      await selects.first().selectOption('linux');
      await selects.last().selectOption('amd64');

      // Verify Build button exists
      const buildBtn = page.locator('button:has-text("Build Implant")');
      await expect(buildBtn).toBeVisible();

      // Click Build
      await buildBtn.click();

      // Stub message should appear
      await expect(page.locator('.text-muted.mt-16')).toContainText('not yet available', { timeout: 5000 });
    });
  });

  test.describe('TC-C2-M10-002: Unauthorized Build Access', () => {
    test('should return 401 without token', async ({ page }) => {
      await page.goto('/login'); // establish origin

      const resp = await apiRequest(page, 'POST', '/build', {
        body: { os: 'linux', arch: 'amd64' },
      });
      expect(resp.status()).toBe(401);
    });
  });
});
