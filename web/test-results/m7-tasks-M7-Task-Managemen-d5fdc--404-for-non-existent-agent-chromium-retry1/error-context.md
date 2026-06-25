# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: m7-tasks.spec.ts >> M7: Task Management >> TC-C2-M7-002: Task for Non-existent Agent >> should return 404 for non-existent agent
- Location: tests\specs\m7-tasks.spec.ts:45:5

# Error details

```
Test timeout of 120000ms exceeded.
```

```
Error: page.waitForSelector: Test timeout of 120000ms exceeded.
Call log:
  - waiting for locator('.login-card') to be visible

```

# Page snapshot

```yaml
- generic [ref=e3]:
  - navigation [ref=e4]:
    - generic [ref=e5]: ◈ C2 Platform
    - generic [ref=e6]:
      - link "◉ Dashboard" [ref=e7] [cursor=pointer]:
        - /url: /
        - generic [ref=e8]: ◉
        - generic [ref=e9]: Dashboard
      - link "◈ Listeners" [ref=e10] [cursor=pointer]:
        - /url: /listeners
        - generic [ref=e11]: ◈
        - generic [ref=e12]: Listeners
      - link "⬡ Agents" [ref=e13] [cursor=pointer]:
        - /url: /agents
        - generic [ref=e14]: ⬡
        - generic [ref=e15]: Agents
      - link "⚒ Build" [ref=e16] [cursor=pointer]:
        - /url: /build
        - generic [ref=e17]: ⚒
        - generic [ref=e18]: Build
      - link "☰ Audit Log" [ref=e19] [cursor=pointer]:
        - /url: /audit
        - generic [ref=e20]: ☰
        - generic [ref=e21]: Audit Log
  - generic [ref=e22]:
    - banner [ref=e23]:
      - generic [ref=e24]: admin (admin)
      - button "Logout" [ref=e25] [cursor=pointer]
    - generic [ref=e27]:
      - heading "Dashboard" [level=1] [ref=e28]
      - generic [ref=e29]:
        - generic [ref=e30]:
          - generic [ref=e31]: "0"
          - generic [ref=e32]: Total Agents
        - generic [ref=e33]:
          - generic [ref=e34]: "0"
          - generic [ref=e35]: Active Agents
        - generic [ref=e36]:
          - generic [ref=e37]: "1"
          - generic [ref=e38]: Listeners Running
        - generic [ref=e39]:
          - generic [ref=e40]: "0"
          - generic [ref=e41]: Tasks Today
      - generic [ref=e42]:
        - generic [ref=e43]: Recent Agents
        - paragraph [ref=e44]: No agents connected yet.
```

# Test source

```ts
  1  | import { Page, BrowserContext } from '@playwright/test';
  2  | 
  3  | /**
  4  |  * Test credentials — configured for localhost development environment.
  5  |  */
  6  | export const TEST_USER = {
  7  |   username: 'admin',
  8  |   password: '259a2daf10da',
  9  | };
  10 | 
  11 | /**
  12 |  * Login via UI: navigate to /login, fill credentials, submit, wait for dashboard.
  13 |  */
  14 | export async function login(page: Page): Promise<void> {
  15 |   await page.goto('/login');
> 16 |   await page.waitForSelector('.login-card');
     |              ^ Error: page.waitForSelector: Test timeout of 120000ms exceeded.
  17 |   await page.locator('input[type="text"]').fill(TEST_USER.username);
  18 |   await page.locator('input[type="password"]').fill(TEST_USER.password);
  19 |   await page.locator('button[type="submit"]').click();
  20 |   await page.waitForURL('**/');
  21 |   await page.waitForSelector('.sidebar');
  22 | }
  23 | 
  24 | /**
  25 |  * Login and return the access_token for use in direct API tests.
  26 |  */
  27 | export async function loginAndGetToken(page: Page): Promise<string> {
  28 |   await login(page);
  29 |   const token = await page.evaluate(() =>
  30 |     localStorage.getItem('access_token')
  31 |   );
  32 |   if (!token) {
  33 |     throw new Error('Login succeeded but no access_token found in localStorage');
  34 |   }
  35 |   return token;
  36 | }
  37 | 
  38 | /**
  39 |  * Logout via the header button.
  40 |  */
  41 | export async function logout(page: Page): Promise<void> {
  42 |   await page.locator('.header .btn-secondary').click();
  43 |   await page.waitForURL('**/login');
  44 | }
  45 | 
  46 | /**
  47 |  * Clear all auth state from localStorage (for a fresh context).
  48 |  */
  49 | export async function clearAuth(context: BrowserContext): Promise<void> {
  50 |   await context.addInitScript(() => {
  51 |     localStorage.removeItem('access_token');
  52 |     localStorage.removeItem('refresh_token');
  53 |   });
  54 | }
  55 | 
  56 | /**
  57 |  * Inject tokens directly into localStorage before page navigation.
  58 |  */
  59 | export async function setTokens(page: Page, accessToken: string, refreshToken: string): Promise<void> {
  60 |   await page.goto('/login'); // need to be on origin before setting localStorage
  61 |   await page.evaluate(
  62 |     ({ at, rt }) => {
  63 |       localStorage.setItem('access_token', at);
  64 |       localStorage.setItem('refresh_token', rt);
  65 |     },
  66 |     { at: accessToken, rt: refreshToken }
  67 |   );
  68 | }
  69 | 
```