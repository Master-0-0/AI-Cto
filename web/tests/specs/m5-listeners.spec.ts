import { test, expect } from '@playwright/test';
import { login } from '../fixtures/auth.fixture';

test.describe('M5: Listener Management', () => {

  test.beforeEach(async ({ page }) => {
    await login(page);
    await page.goto('/listeners');
    await page.waitForSelector('.page-title');
  });

  test.describe('TC-C2-M5-005: Create Listener with Invalid Port', () => {
    test('should show error for port out of range', async ({ page }) => {
      // Open create modal
      await page.locator('button:has-text("+ Create")').click();
      await page.waitForSelector('.modal-overlay');

      // Fill with invalid port (>65535)
      await page.locator('.modal .form-input').first().fill('BadPort');
      await page.locator('.modal input[type="number"]').fill('70000');
      await page.locator('.modal button[type="submit"]').click();

      // Should show error in modal
      await expect(page.locator('.modal .alert-error')).toBeVisible({ timeout: 5000 });

      // Close modal via Cancel
      await page.locator('.modal button:has-text("Cancel")').click();
      await page.waitForSelector('.modal-overlay', { state: 'detached' });
    });
  });

  test.describe('TC-C2-M5-001: Create and List TCP Listener', () => {
    test('should create a listener and display it in the table', async ({ page }) => {
      const listenerName = `Test-TCP-${Date.now()}`;

      // Open modal
      await page.locator('button:has-text("+ Create")').click();
      await page.waitForSelector('.modal-overlay');

      // Fill form
      await page.locator('.modal .form-input').first().fill(listenerName);
      await page.locator('.modal input[type="number"]').fill('19999');
      await page.locator('.modal button[type="submit"]').click();

      // Modal should close
      await page.waitForSelector('.modal-overlay', { state: 'detached', timeout: 5000 });

      // Listener should appear in the table
      await expect(page.locator('td').filter({ hasText: listenerName })).toBeVisible({ timeout: 5000 });

      // Verify status is "stopped"
      const row = page.locator('tr', { hasText: listenerName });
      await expect(row.locator('.badge-stopped')).toBeVisible();
    });
  });

  test.describe('TC-C2-M5-002: Start and Stop Listener', () => {
    test('should toggle between running and stopped', async ({ page }) => {
      // First create a listener
      const listenerName = `Toggle-${Date.now()}`;
      await page.locator('button:has-text("+ Create")').click();
      await page.waitForSelector('.modal-overlay');
      await page.locator('.modal .form-input').first().fill(listenerName);
      await page.locator('.modal input[type="number"]').fill('19998');
      await page.locator('.modal button[type="submit"]').click();
      await page.waitForSelector('.modal-overlay', { state: 'detached', timeout: 5000 });
      await page.waitForTimeout(500); // let table update

      // Find the row
      const row = page.locator('tr', { hasText: listenerName });

      // Should be stopped initially
      await expect(row.locator('.badge-stopped')).toBeVisible({ timeout: 5000 });

      // Click Start
      await row.locator('button:has-text("Start")').click();
      await page.waitForTimeout(1000); // wait for API + table update

      // Should be running now
      await expect(row.locator('.badge-running')).toBeVisible({ timeout: 5000 });

      // Click Stop
      await row.locator('button:has-text("Stop")').click();
      await page.waitForTimeout(1000);

      // Should be stopped again
      await expect(row.locator('.badge-stopped')).toBeVisible({ timeout: 5000 });
    });
  });

  test.describe('TC-C2-M5-003: Cannot Delete Running Listener', () => {
    test('should hide delete button when listener is running', async ({ page }) => {
      // Create a listener
      const listenerName = `NoDel-${Date.now()}`;
      await page.locator('button:has-text("+ Create")').click();
      await page.waitForSelector('.modal-overlay');
      await page.locator('.modal .form-input').first().fill(listenerName);
      await page.locator('.modal input[type="number"]').fill('19997');
      await page.locator('.modal button[type="submit"]').click();
      await page.waitForSelector('.modal-overlay', { state: 'detached', timeout: 5000 });
      await page.waitForTimeout(500);

      const row = page.locator('tr', { hasText: listenerName });

      // Start the listener
      await row.locator('button:has-text("Start")').click();
      await page.waitForTimeout(1000);
      await expect(row.locator('.badge-running')).toBeVisible({ timeout: 5000 });

      // Delete button should NOT be visible while running
      await expect(row.locator('.btn-danger')).not.toBeVisible();

      // Stop the listener
      await row.locator('button:has-text("Stop")').click();
      await page.waitForTimeout(1000);
      await expect(row.locator('.badge-stopped')).toBeVisible({ timeout: 5000 });

      // Delete button should now be visible
      await expect(row.locator('.btn-danger')).toBeVisible();
    });
  });

  test.describe('TC-C2-M5-004: Delete Stopped Listener', () => {
    test('should remove listener from table on delete', async ({ page }) => {
      // Create a listener
      const listenerName = `DelMe-${Date.now()}`;
      await page.locator('button:has-text("+ Create")').click();
      await page.waitForSelector('.modal-overlay');
      await page.locator('.modal .form-input').first().fill(listenerName);
      await page.locator('.modal input[type="number"]').fill('19996');
      await page.locator('.modal button[type="submit"]').click();
      await page.waitForSelector('.modal-overlay', { state: 'detached', timeout: 5000 });
      await page.waitForTimeout(500);

      // Confirm listener exists
      await expect(page.locator('td').filter({ hasText: listenerName })).toBeVisible({ timeout: 5000 });

      // Handle confirm dialog
      page.once('dialog', async (dialog) => {
        expect(dialog.type()).toBe('confirm');
        await dialog.accept();
      });

      // Click Delete
      const row = page.locator('tr', { hasText: listenerName });
      await row.locator('.btn-danger').click();

      // Wait for table update
      await page.waitForTimeout(1000);

      // Listener should be gone
      await expect(page.locator('td').filter({ hasText: listenerName })).not.toBeVisible({ timeout: 5000 });
    });
  });
});
