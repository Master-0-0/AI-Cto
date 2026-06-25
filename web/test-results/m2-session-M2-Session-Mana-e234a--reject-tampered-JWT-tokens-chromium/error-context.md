# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: m2-session.spec.ts >> M2: Session Management >> TC-C2-M2-005: Tampered JWT Token >> should reject tampered JWT tokens
- Location: tests\specs\m2-session.spec.ts:113:5

# Error details

```
Error: expect(received).toBe(expected) // Object.is equality

Expected: 401
Received: 200
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
  25  |       const newToken = await page.evaluate(() => localStorage.getItem('access_token'));
  26  |       expect(newToken).toBeTruthy();
  27  |       expect(newToken).not.toBe(EXPIRED_JWT);
  28  | 
  29  |       console.log(`[TC-C2-M2-001] Token refresh successful. New token differs from expired.`);
  30  |     });
  31  |   });
  32  | 
  33  |   test.describe('TC-C2-M2-002: Expired Refresh Token — Forced Re-login', () => {
  34  |     test('should redirect to login when both tokens are invalid', async ({ page }) => {
  35  |       // Inject invalid tokens
  36  |       await setTokens(page, 'invalid_access_token_xyz', 'invalid_refresh_token_xyz');
  37  | 
  38  |       // Navigate to protected page
  39  |       await page.goto('/');
  40  | 
  41  |       // Should be redirected to /login after auth check fails
  42  |       await page.waitForURL(/\/login/, { timeout: 10000 });
  43  |       await expect(page.locator('.login-card')).toBeVisible();
  44  | 
  45  |       const accessToken = await page.evaluate(() => localStorage.getItem('access_token'));
  46  |       const refreshToken = await page.evaluate(() => localStorage.getItem('refresh_token'));
  47  |       expect(accessToken).toBeNull();
  48  |       expect(refreshToken).toBeNull();
  49  |     });
  50  |   });
  51  | 
  52  |   test.describe('TC-C2-M2-003: Logout Invalidates Refresh Token', () => {
  53  |     test('should revoke refresh token on logout', async ({ page }) => {
  54  |       // Login via API to capture tokens directly
  55  |       const tokens = await apiLogin(page, TEST_USER.username, TEST_USER.password);
  56  | 
  57  |       // Store tokens and navigate
  58  |       await page.goto('/login');
  59  |       await page.evaluate(
  60  |         ({ at, rt }) => {
  61  |           localStorage.setItem('access_token', at);
  62  |           localStorage.setItem('refresh_token', rt);
  63  |         },
  64  |         { at: tokens.access_token, rt: tokens.refresh_token }
  65  |       );
  66  | 
  67  |       // Go to dashboard to confirm logged in
  68  |       await page.goto('/');
  69  |       await page.waitForSelector('.sidebar');
  70  | 
  71  |       // Click logout
  72  |       await page.locator('.header .btn-secondary').click();
  73  |       await page.waitForURL(/\/login/);
  74  | 
  75  |       // Try to use the old refresh token — should be revoked
  76  |       const refreshResp = await apiRefresh(page, tokens.refresh_token);
  77  |       expect(refreshResp).toBeTruthy();
  78  |       // The refresh should fail (token was revoked in DB)
  79  |       // Note: Playwright request returns parsed JSON even on error;
  80  |       // we check if the response contains an error or the status
  81  |       // Make a raw request to check status
  82  |       const rawResp = await apiRequest(page, 'POST', '/auth/refresh', {
  83  |         body: { refresh_token: tokens.refresh_token },
  84  |       });
  85  |       expect(rawResp.status()).toBe(401);
  86  |     });
  87  |   });
  88  | 
  89  |   test.describe('TC-C2-M2-004: Access Protected API without Token', () => {
  90  |     test('should return 401 for all protected endpoints without token', async ({ page }) => {
  91  |       // Must navigate to origin first for page.request to work
  92  |       await page.goto('/login');
  93  | 
  94  |       const endpoints = [
  95  |         { method: 'GET' as const, path: '/dashboard/stats' },
  96  |         { method: 'GET' as const, path: '/agents' },
  97  |         { method: 'POST' as const, path: '/listeners', body: { name: 'test', protocol: 'tcp', host: '0.0.0.0', port: 9999 } },
  98  |         { method: 'GET' as const, path: '/audit?page=1&limit=10' },
  99  |         { method: 'GET' as const, path: '/auth/me' },
  100 |       ];
  101 | 
  102 |       for (const ep of endpoints) {
  103 |         const resp = await apiRequest(page, ep.method, ep.path, { body: ep.body as Record<string, unknown> | undefined });
  104 |         expect(resp.status()).toBe(401);
  105 |         const body = await resp.json();
  106 |         expect(body.error).toBeDefined();
  107 |         expect(body.error.code).toBe('UNAUTHORIZED');
  108 |       }
  109 |     });
  110 |   });
  111 | 
  112 |   test.describe('TC-C2-M2-005: Tampered JWT Token', () => {
  113 |     test('should reject tampered JWT tokens', async ({ page }) => {
  114 |       // Login to get a valid token
  115 |       const token = await loginAndGetToken(page);
  116 | 
  117 |       // Test 1: Tampered signature
  118 |       const badSig = tamperSignature(token);
  119 |       const resp1 = await apiRequest(page, 'GET', '/dashboard/stats', { token: badSig });
  120 |       expect(resp1.status()).toBe(401);
  121 | 
  122 |       // Test 2: Tampered claim (role changed without re-signing)
  123 |       const badClaim = tamperClaim(token, 'role', 'admin');
  124 |       const resp2 = await apiRequest(page, 'GET', '/dashboard/stats', { token: badClaim });
> 125 |       expect(resp2.status()).toBe(401);
      |                              ^ Error: expect(received).toBe(expected) // Object.is equality
  126 |     });
  127 |   });
  128 | });
  129 | 
```