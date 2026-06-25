import { test, expect } from '@playwright/test';
import { login, loginAndGetToken, setTokens } from '../fixtures/auth.fixture';
import { TEST_USER } from '../fixtures/auth.fixture';
import { apiRequest, apiLogin, apiRefresh } from '../helpers/api';
import { EXPIRED_JWT, tamperSignature, tamperClaim } from '../helpers/jwt';

test.describe('M2: Session Management', () => {

  test.describe('TC-C2-M2-001: Token Auto-Refresh on 401', () => {
    test('should transparently refresh token and retry request', async ({ page }) => {
      // Login normally to get valid tokens
      await login(page);
      const originalToken = await page.evaluate(() => localStorage.getItem('access_token'));

      // Replace with an expired JWT (structurally valid but expired in 2020)
      await page.evaluate((expired) => {
        localStorage.setItem('access_token', expired);
      }, EXPIRED_JWT);

      // Navigate to dashboard — triggers getMe() which will 401, then refresh
      await page.goto('/');
      await page.waitForSelector('.sidebar', { timeout: 15000 });

      // A new access_token should be stored (different from the expired one)
      const newToken = await page.evaluate(() => localStorage.getItem('access_token'));
      expect(newToken).toBeTruthy();
      expect(newToken).not.toBe(EXPIRED_JWT);

      console.log(`[TC-C2-M2-001] Token refresh successful. New token differs from expired.`);
    });
  });

  test.describe('TC-C2-M2-002: Expired Refresh Token — Forced Re-login', () => {
    test('should redirect to login when both tokens are invalid', async ({ page }) => {
      // Inject invalid tokens
      await setTokens(page, 'invalid_access_token_xyz', 'invalid_refresh_token_xyz');

      // Navigate to protected page
      await page.goto('/');

      // Should be redirected to /login after auth check fails
      await page.waitForURL(/\/login/, { timeout: 10000 });
      await expect(page.locator('.login-card')).toBeVisible();

      const accessToken = await page.evaluate(() => localStorage.getItem('access_token'));
      const refreshToken = await page.evaluate(() => localStorage.getItem('refresh_token'));
      expect(accessToken).toBeNull();
      expect(refreshToken).toBeNull();
    });
  });

  test.describe('TC-C2-M2-003: Logout Invalidates Refresh Token', () => {
    test('should revoke refresh token on logout', async ({ page }) => {
      // Login via API to capture tokens directly
      const tokens = await apiLogin(page, TEST_USER.username, TEST_USER.password);

      // Store tokens and navigate
      await page.goto('/login');
      await page.evaluate(
        ({ at, rt }) => {
          localStorage.setItem('access_token', at);
          localStorage.setItem('refresh_token', rt);
        },
        { at: tokens.access_token, rt: tokens.refresh_token }
      );

      // Go to dashboard to confirm logged in
      await page.goto('/');
      await page.waitForSelector('.sidebar');

      // Click logout
      await page.locator('.header .btn-secondary').click();
      await page.waitForURL(/\/login/);

      // Try to use the old refresh token — should be revoked
      const refreshResp = await apiRefresh(page, tokens.refresh_token);
      expect(refreshResp).toBeTruthy();
      // The refresh should fail (token was revoked in DB)
      // Note: Playwright request returns parsed JSON even on error;
      // we check if the response contains an error or the status
      // Make a raw request to check status
      const rawResp = await apiRequest(page, 'POST', '/auth/refresh', {
        body: { refresh_token: tokens.refresh_token },
      });
      expect(rawResp.status()).toBe(401);
    });
  });

  test.describe('TC-C2-M2-004: Access Protected API without Token', () => {
    test('should return 401 for all protected endpoints without token', async ({ page }) => {
      // Must navigate to origin first for page.request to work
      await page.goto('/login');

      const endpoints = [
        { method: 'GET' as const, path: '/dashboard/stats' },
        { method: 'GET' as const, path: '/agents' },
        { method: 'POST' as const, path: '/listeners', body: { name: 'test', protocol: 'tcp', host: '0.0.0.0', port: 9999 } },
        { method: 'GET' as const, path: '/audit?page=1&limit=10' },
        { method: 'GET' as const, path: '/auth/me' },
      ];

      for (const ep of endpoints) {
        const resp = await apiRequest(page, ep.method, ep.path, { body: ep.body as Record<string, unknown> | undefined });
        expect(resp.status()).toBe(401);
        const body = await resp.json();
        expect(body.error).toBeDefined();
        expect(body.error.code).toBe('UNAUTHORIZED');
      }
    });
  });

  test.describe('TC-C2-M2-005: Tampered JWT Token', () => {
    test('should reject tampered JWT tokens', async ({ page }) => {
      // Login to get a valid token
      const token = await loginAndGetToken(page);

      // Test 1: Tampered signature
      const badSig = tamperSignature(token);
      const resp1 = await apiRequest(page, 'GET', '/dashboard/stats', { token: badSig });
      expect(resp1.status()).toBe(401);

      // Test 2: Tampered claim (role changed without re-signing)
      const badClaim = tamperClaim(token, 'role', 'admin');
      const resp2 = await apiRequest(page, 'GET', '/dashboard/stats', { token: badClaim });
      expect(resp2.status()).toBe(401);
    });
  });
});
