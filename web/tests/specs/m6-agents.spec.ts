import { test, expect } from '@playwright/test';
import { login } from '../fixtures/auth.fixture';

test.describe('M6: Agent Management', () => {

  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  test.describe('TC-C2-M6-001: Agent List Display', () => {
    test('should display agent table with columns or empty state', async ({ page }) => {
      await page.goto('/agents');
      await page.waitForSelector('.page-title');

      // Either shows table with agents or empty-state message
      const emptyMsg = page.locator('.text-muted').filter({ hasText: /No agents connected/ });
      const table = page.locator('table');

      const hasEmpty = await emptyMsg.isVisible().catch(() => false);
      const hasTable = await table.isVisible().catch(() => false);

      if (hasTable) {
        // Verify column headers
        await expect(page.locator('th').filter({ hasText: 'Hostname' })).toBeVisible();
        await expect(page.locator('th').filter({ hasText: 'OS' })).toBeVisible();
        await expect(page.locator('th').filter({ hasText: 'Username' })).toBeVisible();
        await expect(page.locator('th').filter({ hasText: 'Status' })).toBeVisible();
        await expect(page.locator('th').filter({ hasText: 'Last Seen' })).toBeVisible();
        await expect(page.locator('th').filter({ hasText: 'Actions' })).toBeVisible();
      } else {
        expect(hasEmpty).toBe(true);
      }

      // Verify polling: 2+ requests within 11 seconds
      let requestCount = 0;
      page.on('request', (req) => {
        if (req.url().includes('/api/v1/agents') && !req.url().includes('/tasks')) {
          requestCount++;
        }
      });
      await page.waitForTimeout(11000);
      expect(requestCount).toBeGreaterThan(1);
      console.log(`[TC-C2-M6-001] GET /agents requests in 11s: ${requestCount}`);
    });
  });

  test.describe('TC-C2-M6-002: Delete Agent', () => {
    test('should delete an agent after confirmation', async ({ page }) => {
      await page.goto('/agents');
      await page.waitForSelector('.page-title');

      // Check if there are any agents to delete
      const deleteButtons = page.locator('.btn-danger');
      const count = await deleteButtons.count();

      if (count === 0) {
        console.log('[TC-C2-M6-002] No agents to delete — skipping');
        return;
      }

      // Get the hostname of the first agent
      const firstRow = page.locator('tbody tr').first();
      const hostname = await firstRow.locator('td').first().textContent();

      // Handle confirm dialog
      page.once('dialog', async (dialog) => {
        await dialog.accept();
      });

      // Click Delete on first agent
      await deleteButtons.first().click();
      await page.waitForTimeout(1500);

      // Agent should be removed
      if (hostname) {
        await expect(page.locator('td').filter({ hasText: hostname })).not.toBeVisible({ timeout: 5000 });
      }
    });
  });

  test.describe('TC-C2-M6-003: Agent Detail — System Information', () => {
    test('should display full agent details', async ({ page }) => {
      await page.goto('/agents');
      await page.waitForSelector('.page-title');

      // Check if agents exist
      const agentLink = page.locator('tbody tr a').first();
      const hasAgent = await agentLink.isVisible().catch(() => false);

      if (!hasAgent) {
        console.log('[TC-C2-M6-003] No agents available — skipping');
        return;
      }

      // Click on first agent hostname
      await agentLink.click();
      await page.waitForURL(/\/agents\/[a-f0-9-]+/);

      // Verify System Information card
      await expect(page.locator('.card-title').filter({ hasText: 'System Information' })).toBeVisible({ timeout: 10000 });

      // Verify key info rows
      await expect(page.locator('td').filter({ hasText: 'Agent ID' })).toBeVisible();
      await expect(page.locator('td').filter({ hasText: 'Hostname' })).toBeVisible();
      await expect(page.locator('td').filter({ hasText: 'OS' })).toBeVisible();
      await expect(page.locator('td').filter({ hasText: 'Username' })).toBeVisible();
      await expect(page.locator('td').filter({ hasText: 'PID' })).toBeVisible();

      // Breadcrumb should show "Agents / <hostname>"
      await expect(page.locator('.page-title')).toContainText('Agents /');
    });
  });

  test.describe('TC-C2-M6-004: Non-existent Agent ID', () => {
    test('should show loading spinner for non-existent agent', async ({ page }) => {
      await page.goto('/agents/nonexistent-id-12345');

      // Should show a spinner (loading state)
      const spinner = page.locator('.spinner');
      await expect(spinner).toBeVisible({ timeout: 5000 });

      // App should not crash — sidebar still present
      await expect(page.locator('.sidebar')).toBeVisible();
    });
  });
});
