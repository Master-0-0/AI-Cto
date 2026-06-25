import { test, expect } from '@playwright/test';

test.describe('M3: Auth Guard / Protected Routes', () => {

  // All tests in this suite require a clean context with NO tokens
  test.use({
    storageState: undefined,
  });

  const protectedRoutes = [
    { path: '/', name: 'Dashboard' },
    { path: '/listeners', name: 'Listeners' },
    { path: '/agents', name: 'Agents' },
    { path: '/agents/test-uuid-12345', name: 'Agent Detail' },
    { path: '/audit', name: 'Audit Log' },
    { path: '/build', name: 'Build Panel' },
  ];

  for (const route of protectedRoutes) {
    test(`TC-C2-M3: Direct access to ${route.name} (${route.path}) without login → redirect to /login`, async ({ page }) => {
      // Clear any stored tokens before navigation
      await page.goto('/login');
      await page.evaluate(() => {
        localStorage.removeItem('access_token');
        localStorage.removeItem('refresh_token');
      });

      // Navigate to protected route
      await page.goto(route.path);

      // Should redirect to /login
      await page.waitForURL(/\/login/, { timeout: 10000 });
      await expect(page.locator('.login-card')).toBeVisible();
      await expect(page.locator('.sidebar')).not.toBeVisible();
    });
  }

  test('TC-C2-M3: Browser back after clearing tokens should not reveal content', async ({ page }) => {
    // This is tested more fully in M12-003; basic guard check here
    await page.goto('/login');
    await page.evaluate(() => {
      localStorage.removeItem('access_token');
      localStorage.removeItem('refresh_token');
    });

    // Direct navigation to /agents should redirect
    await page.goto('/agents');
    await page.waitForURL(/\/login/, { timeout: 10000 });
    await expect(page.locator('.login-card')).toBeVisible();
  });
});
