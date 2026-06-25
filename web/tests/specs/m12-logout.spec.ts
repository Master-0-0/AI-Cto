import { test, expect } from '@playwright/test';
import { login, logout } from '../fixtures/auth.fixture';
import { TEST_USER } from '../fixtures/auth.fixture';
import { apiLogin } from '../helpers/api';

test.describe('M12: Logout', () => {

  test.describe('TC-C2-M12-001: Logout Button Click', () => {
    test('should clear tokens and redirect to login', async ({ page }) => {
      await login(page);

      // Click logout button in header
      await page.locator('.header .btn-secondary').click();
      await page.waitForURL(/\/login/);

      // Verify redirected to login
      await expect(page.locator('.login-card')).toBeVisible();

      // Verify localStorage is cleared
      const accessToken = await page.evaluate(() => localStorage.getItem('access_token'));
      const refreshToken = await page.evaluate(() => localStorage.getItem('refresh_token'));
      expect(accessToken).toBeNull();
      expect(refreshToken).toBeNull();

      // Sidebar should not be visible
      await expect(page.locator('.sidebar')).not.toBeVisible();
    });
  });

  test.describe('TC-C2-M12-002: Access Protected Pages After Logout', () => {
    test('should redirect to login for all protected routes after logout', async ({ page }) => {
      await login(page);
      await logout(page);

      // Verify we are on /login
      await expect(page).toHaveURL(/\/login/);

      const protectedRoutes = ['/', '/listeners', '/agents', '/audit', '/build'];

      for (const route of protectedRoutes) {
        await page.goto(route);
        await page.waitForURL(/\/login/, { timeout: 10000 });
        await expect(page.locator('.login-card')).toBeVisible();
      }
    });
  });

  test.describe('TC-C2-M12-003: Browser Back After Logout Does Not Reveal Content', () => {
    test('should stay on login page after pressing back', async ({ page }) => {
      await login(page);

      // Navigate to a few pages to build history
      await page.goto('/agents');
      await page.waitForSelector('.page-title');
      await page.goto('/listeners');
      await page.waitForSelector('.page-title');

      // Logout
      await logout(page);
      await expect(page).toHaveURL(/\/login/);

      // Press back twice
      await page.goBack();
      await page.waitForTimeout(1000);

      // After back, we should either be redirected to /login or the URL may have changed
      // ProtectedRoute should prevent rendering of protected content
      const url = page.url();
      if (!url.includes('/login')) {
        // If we landed somewhere else, the page should quickly redirect
        await page.waitForURL(/\/login/, { timeout: 5000 });
      }
      await expect(page.locator('.login-card')).toBeVisible();

      // Try one more time
      await page.goBack();
      await page.waitForTimeout(1000);
      const url2 = page.url();
      if (!url2.includes('/login')) {
        await page.waitForURL(/\/login/, { timeout: 5000 });
      }
      await expect(page.locator('.login-card')).toBeVisible();
    });
  });
});
