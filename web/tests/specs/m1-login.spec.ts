import { test, expect } from '@playwright/test';
import { login, loginAndGetToken, clearAuth } from '../fixtures/auth.fixture';
import { TEST_USER } from '../fixtures/auth.fixture';

test.describe('M1: Login Module', () => {

  test.describe('TC-C2-M1-001: Successful Login', () => {
    test('should login with valid credentials and redirect to dashboard', async ({ page }) => {
      await page.goto('/login');
      await expect(page.locator('.login-card')).toBeVisible();
      await expect(page.locator('.login-title')).toHaveText('C2 Platform');

      await page.locator('input[type="text"]').fill(TEST_USER.username);
      await page.locator('input[type="password"]').fill(TEST_USER.password);
      await page.locator('button[type="submit"]').click();

      await page.waitForURL('**/');
      await expect(page.locator('.sidebar')).toBeVisible();
      await expect(page.locator('.header-user')).toContainText(TEST_USER.username);

      const accessToken = await page.evaluate(() => localStorage.getItem('access_token'));
      const refreshToken = await page.evaluate(() => localStorage.getItem('refresh_token'));
      expect(accessToken).toBeTruthy();
      expect(refreshToken).toBeTruthy();
    });
  });

  test.describe('TC-C2-M1-002: Login with Wrong Password', () => {
    test('should show error and stay on login page', async ({ page }) => {
      await page.goto('/login');
      await page.locator('input[type="text"]').fill(TEST_USER.username);
      await page.locator('input[type="password"]').fill('wrongpassword123');
      await page.locator('button[type="submit"]').click();

      await expect(page.locator('.alert-error')).toBeVisible({ timeout: 5000 });
      await expect(page).toHaveURL(/\/login/);

      const accessToken = await page.evaluate(() => localStorage.getItem('access_token'));
      expect(accessToken).toBeNull();
    });
  });

  test.describe('TC-C2-M1-003: Login with Empty Fields', () => {
    test('should be blocked by HTML5 required validation', async ({ page }) => {
      await page.goto('/login');

      // Verify required attribute exists on both inputs
      const usernameInput = page.locator('input[type="text"]');
      const passwordInput = page.locator('input[type="password"]');
      await expect(usernameInput).toHaveAttribute('required');
      await expect(passwordInput).toHaveAttribute('required');

      // Bypass HTML5 validation via evaluate to test backend behavior
      await page.evaluate(() => {
        const form = document.querySelector('form');
        if (form) form.noValidate = true;
      });
      await page.locator('button[type="submit"]').click();

      // Backend should return error; app stays on /login
      await expect(page).toHaveURL(/\/login/);
    });
  });

  test.describe('TC-C2-M1-004: Login with Non-existent Username', () => {
    test('should show invalid credentials error', async ({ page }) => {
      await page.goto('/login');
      await page.locator('input[type="text"]').fill('nonexistent_user_xyz');
      await page.locator('input[type="password"]').fill('anything');
      await page.locator('button[type="submit"]').click();

      await expect(page.locator('.alert-error')).toBeVisible({ timeout: 5000 });
      await expect(page.locator('.alert-error')).toContainText(/Invalid credentials/i);
    });
  });

  test.describe('TC-C2-M1-005: Already Authenticated User Redirect', () => {
    test('should redirect to dashboard when already logged in', async ({ page }) => {
      // First login to get valid tokens
      await login(page);

      // Now navigate to /login — should redirect away
      await page.goto('/login');
      await page.waitForURL('**/');
      await expect(page).not.toHaveURL(/\/login/);
      await expect(page.locator('.sidebar')).toBeVisible();
    });
  });

  test.describe('TC-C2-M1-006: GET Method Not Allowed on Login', () => {
    test('should reject GET requests to login endpoint', async ({ page }) => {
      const resp = await page.request.get(
        `http://localhost:5173/api/v1/auth/login?username=${TEST_USER.username}&password=${TEST_USER.password}`
      );
      expect(resp.status()).not.toBe(200);
    });
  });

  test.describe('TC-C2-M1-007: SQL Injection Attempt', () => {
    test('should reject SQL injection in username field', async ({ page }) => {
      await page.goto('/login');
      await page.locator('input[type="text"]').fill("admin' OR '1'='1");
      await page.locator('input[type="password"]').fill('anything');
      await page.locator('button[type="submit"]').click();

      await expect(page.locator('.alert-error')).toBeVisible({ timeout: 5000 });
    });
  });

  test.describe('TC-C2-M1-008: Rapid Consecutive Wrong Password Attempts', () => {
    test('should handle rapid failed login attempts', async ({ page }) => {
      await page.goto('/login');
      const timings: number[] = [];

      for (let i = 0; i < 5; i++) {
        const start = Date.now();
        await page.locator('input[type="text"]').fill(TEST_USER.username);
        await page.locator('input[type="password"]').fill(`wrongpass${i}`);
        await page.locator('button[type="submit"]').click();
        await expect(page.locator('.alert-error')).toBeVisible({ timeout: 10000 });
        timings.push(Date.now() - start);
        // Clear fields for next attempt
        await page.locator('input[type="text"]').fill('');
        await page.locator('input[type="password"]').fill('');
      }

      // Now attempt with correct password — should still succeed
      await page.locator('input[type="text"]').fill(TEST_USER.username);
      await page.locator('input[type="password"]').fill(TEST_USER.password);
      await page.locator('button[type="submit"]').click();
      await page.waitForURL('**/');
      await expect(page.locator('.sidebar')).toBeVisible();

      // Log timings for analysis (bcrypt cost=12 should show ~100-300ms)
      console.log(`[TC-C2-M1-008] Login attempt response times (ms): ${timings.join(', ')}`);
    });
  });
});
