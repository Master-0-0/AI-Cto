import { test, expect } from '@playwright/test';
import { login } from '../fixtures/auth.fixture';
import { apiRequest } from '../helpers/api';

test.describe('M9: Audit Log', () => {

  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  test.describe('TC-C2-M9-001: Audit Log Paginated Display', () => {
    test('should display audit log table with correct columns', async ({ page }) => {
      await page.goto('/audit');
      await page.waitForSelector('.page-title');

      await expect(page.locator('.page-title')).toContainText('Audit Log');

      // Check if data exists
      const emptyMsg = page.locator('.text-muted').filter({ hasText: /No audit entries/ });
      const table = page.locator('table');

      const hasEmpty = await emptyMsg.isVisible().catch(() => false);

      if (!hasEmpty) {
        // Verify column headers
        await expect(page.locator('th').filter({ hasText: 'Time' })).toBeVisible();
        await expect(page.locator('th').filter({ hasText: 'Action' })).toBeVisible();
        await expect(page.locator('th').filter({ hasText: 'Resource' })).toBeVisible();
        await expect(page.locator('th').filter({ hasText: 'IP' })).toBeVisible();

        // Login action should appear (at least one audit entry from login)
        await expect(page.locator('td').filter({ hasText: 'auth.login' })).toBeVisible({ timeout: 5000 });
      }
    });
  });

  test.describe('TC-C2-M9-002: Audit Log Pagination Controls', () => {
    test('should handle previous/next pagination correctly', async ({ page }) => {
      await page.goto('/audit');
      await page.waitForSelector('.page-title');

      const emptyMsg = page.locator('.text-muted').filter({ hasText: /No audit entries/ });
      const hasEmpty = await emptyMsg.isVisible().catch(() => false);

      if (hasEmpty) {
        console.log('[TC-C2-M9-002] No audit entries — skipping pagination test');
        return;
      }

      // Previous should be disabled on page 1
      const prevBtn = page.locator('button:has-text("Previous")');
      await expect(prevBtn).toBeDisabled();

      // Should show page indicator
      await expect(page.locator('.text-muted').filter({ hasText: /Page/ })).toBeVisible();

      // Click Next
      const nextBtn = page.locator('button:has-text("Next")');
      const isNextDisabled = await nextBtn.isDisabled();

      if (isNextDisabled) {
        // Fewer than 50 entries — only one page. That's valid.
        console.log('[TC-C2-M9-002] Only one page of audit logs (less than 50 entries)');
        return;
      }

      await nextBtn.click();
      await page.waitForTimeout(1000);

      // Previous should now be enabled on page 2
      await expect(prevBtn).toBeEnabled();

      // Go back to page 1
      await prevBtn.click();
      await page.waitForTimeout(1000);

      // Previous should be disabled again
      await expect(prevBtn).toBeDisabled();
    });
  });

  test.describe('TC-C2-M9-003: Unauthorized Audit Access', () => {
    test('should return 401 without token', async ({ page }) => {
      await page.goto('/login'); // establish origin

      const resp = await apiRequest(page, 'GET', '/audit?page=1&limit=10');
      expect(resp.status()).toBe(401);
    });
  });
});
