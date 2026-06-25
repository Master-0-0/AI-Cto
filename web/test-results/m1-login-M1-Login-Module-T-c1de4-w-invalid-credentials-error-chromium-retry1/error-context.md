# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: m1-login.spec.ts >> M1: Login Module >> TC-C2-M1-004: Login with Non-existent Username >> should show invalid credentials error
- Location: tests\specs\m1-login.spec.ts:66:5

# Error details

```
Error: expect(locator).toBeVisible() failed

Locator: locator('.alert-error')
Expected: visible
Timeout: 5000ms
Error: element(s) not found

Call log:
  - Expect "toBeVisible" with timeout 5000ms
  - waiting for locator('.alert-error')
    - waiting for navigation to finish...
    - navigated to "http://localhost:5173/login"

```

```yaml
- heading "C2 Platform" [level=1]
- text: Username
- textbox
- text: Password
- textbox
- button "Login"
```

# Test source

```ts
  1   | import { test, expect } from '@playwright/test';
  2   | import { login, loginAndGetToken, clearAuth } from '../fixtures/auth.fixture';
  3   | import { TEST_USER } from '../fixtures/auth.fixture';
  4   | 
  5   | test.describe('M1: Login Module', () => {
  6   | 
  7   |   test.describe('TC-C2-M1-001: Successful Login', () => {
  8   |     test('should login with valid credentials and redirect to dashboard', async ({ page }) => {
  9   |       await page.goto('/login');
  10  |       await expect(page.locator('.login-card')).toBeVisible();
  11  |       await expect(page.locator('.login-title')).toHaveText('C2 Platform');
  12  | 
  13  |       await page.locator('input[type="text"]').fill(TEST_USER.username);
  14  |       await page.locator('input[type="password"]').fill(TEST_USER.password);
  15  |       await page.locator('button[type="submit"]').click();
  16  | 
  17  |       await page.waitForURL('**/');
  18  |       await expect(page.locator('.sidebar')).toBeVisible();
  19  |       await expect(page.locator('.header-user')).toContainText(TEST_USER.username);
  20  | 
  21  |       const accessToken = await page.evaluate(() => localStorage.getItem('access_token'));
  22  |       const refreshToken = await page.evaluate(() => localStorage.getItem('refresh_token'));
  23  |       expect(accessToken).toBeTruthy();
  24  |       expect(refreshToken).toBeTruthy();
  25  |     });
  26  |   });
  27  | 
  28  |   test.describe('TC-C2-M1-002: Login with Wrong Password', () => {
  29  |     test('should show error and stay on login page', async ({ page }) => {
  30  |       await page.goto('/login');
  31  |       await page.locator('input[type="text"]').fill(TEST_USER.username);
  32  |       await page.locator('input[type="password"]').fill('wrongpassword123');
  33  |       await page.locator('button[type="submit"]').click();
  34  | 
  35  |       await expect(page.locator('.alert-error')).toBeVisible({ timeout: 5000 });
  36  |       await expect(page).toHaveURL(/\/login/);
  37  | 
  38  |       const accessToken = await page.evaluate(() => localStorage.getItem('access_token'));
  39  |       expect(accessToken).toBeNull();
  40  |     });
  41  |   });
  42  | 
  43  |   test.describe('TC-C2-M1-003: Login with Empty Fields', () => {
  44  |     test('should be blocked by HTML5 required validation', async ({ page }) => {
  45  |       await page.goto('/login');
  46  | 
  47  |       // Verify required attribute exists on both inputs
  48  |       const usernameInput = page.locator('input[type="text"]');
  49  |       const passwordInput = page.locator('input[type="password"]');
  50  |       await expect(usernameInput).toHaveAttribute('required');
  51  |       await expect(passwordInput).toHaveAttribute('required');
  52  | 
  53  |       // Bypass HTML5 validation via evaluate to test backend behavior
  54  |       await page.evaluate(() => {
  55  |         const form = document.querySelector('form');
  56  |         if (form) form.noValidate = true;
  57  |       });
  58  |       await page.locator('button[type="submit"]').click();
  59  | 
  60  |       // Backend should return error; app stays on /login
  61  |       await expect(page).toHaveURL(/\/login/);
  62  |     });
  63  |   });
  64  | 
  65  |   test.describe('TC-C2-M1-004: Login with Non-existent Username', () => {
  66  |     test('should show invalid credentials error', async ({ page }) => {
  67  |       await page.goto('/login');
  68  |       await page.locator('input[type="text"]').fill('nonexistent_user_xyz');
  69  |       await page.locator('input[type="password"]').fill('anything');
  70  |       await page.locator('button[type="submit"]').click();
  71  | 
> 72  |       await expect(page.locator('.alert-error')).toBeVisible({ timeout: 5000 });
      |                                                  ^ Error: expect(locator).toBeVisible() failed
  73  |       await expect(page.locator('.alert-error')).toContainText(/Invalid credentials/i);
  74  |     });
  75  |   });
  76  | 
  77  |   test.describe('TC-C2-M1-005: Already Authenticated User Redirect', () => {
  78  |     test('should redirect to dashboard when already logged in', async ({ page }) => {
  79  |       // First login to get valid tokens
  80  |       await login(page);
  81  | 
  82  |       // Now navigate to /login — should redirect away
  83  |       await page.goto('/login');
  84  |       await page.waitForURL('**/');
  85  |       await expect(page).not.toHaveURL(/\/login/);
  86  |       await expect(page.locator('.sidebar')).toBeVisible();
  87  |     });
  88  |   });
  89  | 
  90  |   test.describe('TC-C2-M1-006: GET Method Not Allowed on Login', () => {
  91  |     test('should reject GET requests to login endpoint', async ({ page }) => {
  92  |       const resp = await page.request.get(
  93  |         `http://localhost:5173/api/v1/auth/login?username=${TEST_USER.username}&password=${TEST_USER.password}`
  94  |       );
  95  |       expect(resp.status()).not.toBe(200);
  96  |     });
  97  |   });
  98  | 
  99  |   test.describe('TC-C2-M1-007: SQL Injection Attempt', () => {
  100 |     test('should reject SQL injection in username field', async ({ page }) => {
  101 |       await page.goto('/login');
  102 |       await page.locator('input[type="text"]').fill("admin' OR '1'='1");
  103 |       await page.locator('input[type="password"]').fill('anything');
  104 |       await page.locator('button[type="submit"]').click();
  105 | 
  106 |       await expect(page.locator('.alert-error')).toBeVisible({ timeout: 5000 });
  107 |     });
  108 |   });
  109 | 
  110 |   test.describe('TC-C2-M1-008: Rapid Consecutive Wrong Password Attempts', () => {
  111 |     test('should handle rapid failed login attempts', async ({ page }) => {
  112 |       await page.goto('/login');
  113 |       const timings: number[] = [];
  114 | 
  115 |       for (let i = 0; i < 5; i++) {
  116 |         const start = Date.now();
  117 |         await page.locator('input[type="text"]').fill(TEST_USER.username);
  118 |         await page.locator('input[type="password"]').fill(`wrongpass${i}`);
  119 |         await page.locator('button[type="submit"]').click();
  120 |         await expect(page.locator('.alert-error')).toBeVisible({ timeout: 10000 });
  121 |         timings.push(Date.now() - start);
  122 |         // Clear fields for next attempt
  123 |         await page.locator('input[type="text"]').fill('');
  124 |         await page.locator('input[type="password"]').fill('');
  125 |       }
  126 | 
  127 |       // Now attempt with correct password — should still succeed
  128 |       await page.locator('input[type="text"]').fill(TEST_USER.username);
  129 |       await page.locator('input[type="password"]').fill(TEST_USER.password);
  130 |       await page.locator('button[type="submit"]').click();
  131 |       await page.waitForURL('**/');
  132 |       await expect(page.locator('.sidebar')).toBeVisible();
  133 | 
  134 |       // Log timings for analysis (bcrypt cost=12 should show ~100-300ms)
  135 |       console.log(`[TC-C2-M1-008] Login attempt response times (ms): ${timings.join(', ')}`);
  136 |     });
  137 |   });
  138 | });
  139 | 
```