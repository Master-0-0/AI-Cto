import { test, expect } from '@playwright/test';
import { login } from '../fixtures/auth.fixture';

test.describe('M11: Navigation', () => {

  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  test.describe('TC-C2-M11-001: Sidebar Navigation to All Pages', () => {
    test('should navigate to all 5 pages via sidebar', async ({ page }) => {
      const navItems = [
        { label: 'Dashboard', url: '/' },
        { label: 'Listeners', url: '/listeners' },
        { label: 'Agents', url: '/agents' },
        { label: 'Build', url: '/build' },
        { label: 'Audit Log', url: '/audit' },
      ];

      for (const item of navItems) {
        // Click sidebar link
        await page.locator('.sidebar-link').filter({ hasText: item.label }).click();
        await expect(page).toHaveURL(item.url);

        // Verify page title matches
        if (item.label === 'Dashboard') {
          await expect(page.locator('.page-title')).toContainText('Dashboard');
        } else if (item.label === 'Audit Log') {
          await expect(page.locator('.page-title')).toContainText('Audit Log');
        } else if (item.label === 'Build') {
          await expect(page.locator('.page-title')).toContainText('Build Implant');
        } else {
          await expect(page.locator('.page-title')).toContainText(item.label);
        }
      }
    });
  });

  test.describe('TC-C2-M11-002: Active Link Highlighting', () => {
    test('should highlight only the current page link', async ({ page }) => {
      // Go to Listeners
      await page.goto('/listeners');
      await page.waitForSelector('.page-title');

      // Only one link should be active
      const activeLinks = page.locator('.sidebar-link.active');
      await expect(activeLinks).toHaveCount(1);
      await expect(activeLinks).toContainText('Listeners');

      // Navigate to Agents
      await page.goto('/agents');
      await page.waitForSelector('.page-title');

      // Only one link should be active, and it should be Agents
      await expect(page.locator('.sidebar-link.active')).toHaveCount(1);
      await expect(page.locator('.sidebar-link.active')).toContainText('Agents');

      // Navigate to Dashboard
      await page.goto('/');
      await page.waitForSelector('.page-title');

      await expect(page.locator('.sidebar-link.active')).toHaveCount(1);
      await expect(page.locator('.sidebar-link.active')).toContainText('Dashboard');
    });
  });

  test.describe('TC-C2-M11-003: Header Shows Username and Role', () => {
    test('should display username (role) in header', async ({ page }) => {
      await page.goto('/');
      await page.waitForSelector('.header-user');

      const headerUser = page.locator('.header-user');
      await expect(headerUser).toBeVisible();
      await expect(headerUser).toContainText('admin');
      await expect(headerUser).toContainText('(admin)');
    });
  });
});
