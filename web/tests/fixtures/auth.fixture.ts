import { Page, BrowserContext } from '@playwright/test';

/**
 * Test credentials — configured for localhost development environment.
 */
export const TEST_USER = {
  username: 'admin',
  password: '259a2daf10da',
};

/**
 * Login via UI: navigate to /login, fill credentials, submit, wait for dashboard.
 */
export async function login(page: Page): Promise<void> {
  await page.goto('/login');
  await page.waitForSelector('.login-card');
  await page.locator('input[type="text"]').fill(TEST_USER.username);
  await page.locator('input[type="password"]').fill(TEST_USER.password);
  await page.locator('button[type="submit"]').click();
  await page.waitForURL('**/');
  await page.waitForSelector('.sidebar');
}

/**
 * Login and return the access_token for use in direct API tests.
 */
export async function loginAndGetToken(page: Page): Promise<string> {
  await login(page);
  const token = await page.evaluate(() =>
    localStorage.getItem('access_token')
  );
  if (!token) {
    throw new Error('Login succeeded but no access_token found in localStorage');
  }
  return token;
}

/**
 * Logout via the header button.
 */
export async function logout(page: Page): Promise<void> {
  await page.locator('.header .btn-secondary').click();
  await page.waitForURL('**/login');
}

/**
 * Clear all auth state from localStorage (for a fresh context).
 */
export async function clearAuth(context: BrowserContext): Promise<void> {
  await context.addInitScript(() => {
    localStorage.removeItem('access_token');
    localStorage.removeItem('refresh_token');
  });
}

/**
 * Inject tokens directly into localStorage before page navigation.
 */
export async function setTokens(page: Page, accessToken: string, refreshToken: string): Promise<void> {
  await page.goto('/login'); // need to be on origin before setting localStorage
  await page.evaluate(
    ({ at, rt }) => {
      localStorage.setItem('access_token', at);
      localStorage.setItem('refresh_token', rt);
    },
    { at: accessToken, rt: refreshToken }
  );
}
