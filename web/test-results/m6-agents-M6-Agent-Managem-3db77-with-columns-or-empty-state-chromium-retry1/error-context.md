# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: m6-agents.spec.ts >> M6: Agent Management >> TC-C2-M6-001: Agent List Display >> should display agent table with columns or empty state
- Location: tests\specs\m6-agents.spec.ts:11:5

# Error details

```
Error: expect(received).toBeGreaterThan(expected)

Expected: > 1
Received:   1
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
      - heading "Agents" [level=1] [ref=e28]
      - paragraph [ref=e30]: No agents connected. Deploy an implant to get started.
```

# Test source

```ts
  1   | import { test, expect } from '@playwright/test';
  2   | import { login } from '../fixtures/auth.fixture';
  3   | 
  4   | test.describe('M6: Agent Management', () => {
  5   | 
  6   |   test.beforeEach(async ({ page }) => {
  7   |     await login(page);
  8   |   });
  9   | 
  10  |   test.describe('TC-C2-M6-001: Agent List Display', () => {
  11  |     test('should display agent table with columns or empty state', async ({ page }) => {
  12  |       await page.goto('/agents');
  13  |       await page.waitForSelector('.page-title');
  14  | 
  15  |       // Either shows table with agents or empty-state message
  16  |       const emptyMsg = page.locator('.text-muted').filter({ hasText: /No agents connected/ });
  17  |       const table = page.locator('table');
  18  | 
  19  |       const hasEmpty = await emptyMsg.isVisible().catch(() => false);
  20  |       const hasTable = await table.isVisible().catch(() => false);
  21  | 
  22  |       if (hasTable) {
  23  |         // Verify column headers
  24  |         await expect(page.locator('th').filter({ hasText: 'Hostname' })).toBeVisible();
  25  |         await expect(page.locator('th').filter({ hasText: 'OS' })).toBeVisible();
  26  |         await expect(page.locator('th').filter({ hasText: 'Username' })).toBeVisible();
  27  |         await expect(page.locator('th').filter({ hasText: 'Status' })).toBeVisible();
  28  |         await expect(page.locator('th').filter({ hasText: 'Last Seen' })).toBeVisible();
  29  |         await expect(page.locator('th').filter({ hasText: 'Actions' })).toBeVisible();
  30  |       } else {
  31  |         expect(hasEmpty).toBe(true);
  32  |       }
  33  | 
  34  |       // Verify polling: 2+ requests within 11 seconds
  35  |       let requestCount = 0;
  36  |       page.on('request', (req) => {
  37  |         if (req.url().includes('/api/v1/agents') && !req.url().includes('/tasks')) {
  38  |           requestCount++;
  39  |         }
  40  |       });
  41  |       await page.waitForTimeout(11000);
> 42  |       expect(requestCount).toBeGreaterThan(1);
      |                            ^ Error: expect(received).toBeGreaterThan(expected)
  43  |       console.log(`[TC-C2-M6-001] GET /agents requests in 11s: ${requestCount}`);
  44  |     });
  45  |   });
  46  | 
  47  |   test.describe('TC-C2-M6-002: Delete Agent', () => {
  48  |     test('should delete an agent after confirmation', async ({ page }) => {
  49  |       await page.goto('/agents');
  50  |       await page.waitForSelector('.page-title');
  51  | 
  52  |       // Check if there are any agents to delete
  53  |       const deleteButtons = page.locator('.btn-danger');
  54  |       const count = await deleteButtons.count();
  55  | 
  56  |       if (count === 0) {
  57  |         console.log('[TC-C2-M6-002] No agents to delete — skipping');
  58  |         return;
  59  |       }
  60  | 
  61  |       // Get the hostname of the first agent
  62  |       const firstRow = page.locator('tbody tr').first();
  63  |       const hostname = await firstRow.locator('td').first().textContent();
  64  | 
  65  |       // Handle confirm dialog
  66  |       page.once('dialog', async (dialog) => {
  67  |         await dialog.accept();
  68  |       });
  69  | 
  70  |       // Click Delete on first agent
  71  |       await deleteButtons.first().click();
  72  |       await page.waitForTimeout(1500);
  73  | 
  74  |       // Agent should be removed
  75  |       if (hostname) {
  76  |         await expect(page.locator('td').filter({ hasText: hostname })).not.toBeVisible({ timeout: 5000 });
  77  |       }
  78  |     });
  79  |   });
  80  | 
  81  |   test.describe('TC-C2-M6-003: Agent Detail — System Information', () => {
  82  |     test('should display full agent details', async ({ page }) => {
  83  |       await page.goto('/agents');
  84  |       await page.waitForSelector('.page-title');
  85  | 
  86  |       // Check if agents exist
  87  |       const agentLink = page.locator('tbody tr a').first();
  88  |       const hasAgent = await agentLink.isVisible().catch(() => false);
  89  | 
  90  |       if (!hasAgent) {
  91  |         console.log('[TC-C2-M6-003] No agents available — skipping');
  92  |         return;
  93  |       }
  94  | 
  95  |       // Click on first agent hostname
  96  |       await agentLink.click();
  97  |       await page.waitForURL(/\/agents\/[a-f0-9-]+/);
  98  | 
  99  |       // Verify System Information card
  100 |       await expect(page.locator('.card-title').filter({ hasText: 'System Information' })).toBeVisible({ timeout: 10000 });
  101 | 
  102 |       // Verify key info rows
  103 |       await expect(page.locator('td').filter({ hasText: 'Agent ID' })).toBeVisible();
  104 |       await expect(page.locator('td').filter({ hasText: 'Hostname' })).toBeVisible();
  105 |       await expect(page.locator('td').filter({ hasText: 'OS' })).toBeVisible();
  106 |       await expect(page.locator('td').filter({ hasText: 'Username' })).toBeVisible();
  107 |       await expect(page.locator('td').filter({ hasText: 'PID' })).toBeVisible();
  108 | 
  109 |       // Breadcrumb should show "Agents / <hostname>"
  110 |       await expect(page.locator('.page-title')).toContainText('Agents /');
  111 |     });
  112 |   });
  113 | 
  114 |   test.describe('TC-C2-M6-004: Non-existent Agent ID', () => {
  115 |     test('should show loading spinner for non-existent agent', async ({ page }) => {
  116 |       await page.goto('/agents/nonexistent-id-12345');
  117 | 
  118 |       // Should show a spinner (loading state)
  119 |       const spinner = page.locator('.spinner');
  120 |       await expect(spinner).toBeVisible({ timeout: 5000 });
  121 | 
  122 |       // App should not crash — sidebar still present
  123 |       await expect(page.locator('.sidebar')).toBeVisible();
  124 |     });
  125 |   });
  126 | });
  127 | 
```