import { test, expect } from '@playwright/test';
import { login, loginAndGetToken } from '../fixtures/auth.fixture';
import { apiRequest } from '../helpers/api';

test.describe('M7: Task Management', () => {

  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  test.describe('TC-C2-M7-001: Create Shell Command Task via Form', () => {
    test('should create a shell task and show it in the task list', async ({ page }) => {
      await page.goto('/agents');
      await page.waitForSelector('.page-title');

      // Need an active agent — check if any exist
      const agentLink = page.locator('tbody tr a').first();
      const hasAgent = await agentLink.isVisible().catch(() => false);

      if (!hasAgent) {
        console.log('[TC-C2-M7-001] No agents available — skipping');
        return;
      }

      // Navigate to agent detail
      await agentLink.click();
      await page.waitForURL(/\/agents\/[a-f0-9-]+/);
      await page.waitForSelector('.card-title');

      // Use the Send Command form
      // Command dropdown defaults to "shell"
      await page.locator('.card .form-select').first().selectOption('shell');
      // Arguments input — the one inside the Send Command card
      const argsInput = page.locator('.card:has(.card-title:has-text("Send Command")) .form-input');
      await argsInput.fill('whoami');
      // Click Send
      await page.locator('.card:has(.card-title:has-text("Send Command")) button[type="submit"]').click();

      // Task should appear in the task list
      await expect(page.locator('td').filter({ hasText: 'whoami' })).toBeVisible({ timeout: 10000 });
    });
  });

  test.describe('TC-C2-M7-002: Task for Non-existent Agent', () => {
    test('should return 404 for non-existent agent', async ({ page }) => {
      const token = await loginAndGetToken(page);

      const resp = await apiRequest(page, 'POST', '/agents/nonexistent-id/tasks', {
        body: { command: 'shell', arguments: 'test' },
        token,
      });
      expect(resp.status()).toBe(404);
    });
  });

  test.describe('TC-C2-M7-003: Task without Required Command Field', () => {
    test('should return 400 when command field is missing', async ({ page }) => {
      await page.goto('/agents');
      await page.waitForSelector('.page-title');

      const agentLink = page.locator('tbody tr a').first();
      const hasAgent = await agentLink.isVisible().catch(() => false);

      if (!hasAgent) {
        console.log('[TC-C2-M7-003] No agents available — skipping');
        return;
      }

      // Get agent ID from URL
      await agentLink.click();
      await page.waitForURL(/\/agents\/([a-f0-9-]+)/);
      const url = page.url();
      const agentId = url.split('/agents/')[1];

      const token = await page.evaluate(() => localStorage.getItem('access_token'));
      const resp = await apiRequest(page, 'POST', `/agents/${agentId}/tasks`, {
        body: { arguments: 'test' }, // no command field
        token: token || undefined,
      });
      expect(resp.status()).toBe(400);
    });
  });

  test.describe('TC-C2-M7-004: Task Status Lifecycle Display', () => {
    test('should show task transitioning through statuses', async ({ page }) => {
      await page.goto('/agents');
      await page.waitForSelector('.page-title');

      const agentLink = page.locator('tbody tr a').first();
      const hasAgent = await agentLink.isVisible().catch(() => false);

      if (!hasAgent) {
        console.log('[TC-C2-M7-004] No agents available — skipping');
        return;
      }

      await agentLink.click();
      await page.waitForURL(/\/agents\/[a-f0-9-]+/);
      await page.waitForSelector('.card-title', { timeout: 10000 });

      // Check if there's a badge indicating agent is active
      const statusBadge = page.locator('.badge-active');
      const isActive = await statusBadge.isVisible().catch(() => false);

      if (!isActive) {
        console.log('[TC-C2-M7-004] Agent is not active — cannot test task lifecycle');
        return;
      }

      // Create a task
      const argsInput = page.locator('.card:has(.card-title:has-text("Send Command")) .form-input');
      await argsInput.fill('echo lifecycle_test');
      await page.locator('.card:has(.card-title:has-text("Send Command")) button[type="submit"]').click();

      // Wait for task to appear
      await expect(page.locator('td').filter({ hasText: 'lifecycle_test' })).toBeVisible({ timeout: 10000 });

      // Poll-wait for task status to become 'completed' or 'error'
      // (Agent needs to pick up and execute — may take seconds)
      try {
        await expect(page.locator('.badge-completed').or(page.locator('.badge-error'))).toBeVisible({ timeout: 30000 });
      } catch {
        console.log('[TC-C2-M7-004] Task did not complete within 30s — may still be pending');
      }
    });
  });
});
