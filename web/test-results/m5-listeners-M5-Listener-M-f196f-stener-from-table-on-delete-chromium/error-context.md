# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: m5-listeners.spec.ts >> M5: Listener Management >> TC-C2-M5-004: Delete Stopped Listener >> should remove listener from table on delete
- Location: tests\specs\m5-listeners.spec.ts:124:5

# Error details

```
Error: expect(locator).toBeVisible() failed

Locator: locator('td').filter({ hasText: 'DelMe-1782383949372' })
Expected: visible
Timeout: 5000ms
Error: element(s) not found

Call log:
  - Expect "toBeVisible" with timeout 5000ms
  - waiting for locator('td').filter({ hasText: 'DelMe-1782383949372' })

```

```yaml
- navigation:
  - text: ◈ C2 Platform
  - link "◉ Dashboard":
    - /url: /
  - link "◈ Listeners":
    - /url: /listeners
  - link "⬡ Agents":
    - /url: /agents
  - link "⚒ Build":
    - /url: /build
  - link "☰ Audit Log":
    - /url: /audit
- banner:
  - text: admin (admin)
  - button "Logout"
- heading "Listeners" [level=1]
- button "+ Create"
- table:
  - rowgroup:
    - row "Name Protocol Address Status Actions":
      - columnheader "Name"
      - columnheader "Protocol"
      - columnheader "Address"
      - columnheader "Status"
      - columnheader "Actions"
  - rowgroup:
    - row "e505199a TCP 0.0.0.0:4444 running Stop":
      - cell "e505199a"
      - cell "TCP"
      - cell "0.0.0.0:4444"
      - cell "running"
      - cell "Stop":
        - button "Stop"
    - row "86b36d83 TCP 0.0.0.0:19999 stopped Start Delete":
      - cell "86b36d83"
      - cell "TCP"
      - cell "0.0.0.0:19999"
      - cell "stopped"
      - cell "Start Delete":
        - button "Start"
        - button "Delete"
    - row "bfa96c7f TCP 0.0.0.0:19998 stopped Start Delete":
      - cell "bfa96c7f"
      - cell "TCP"
      - cell "0.0.0.0:19998"
      - cell "stopped"
      - cell "Start Delete":
        - button "Start"
        - button "Delete"
    - row "959fd8d7 TCP 0.0.0.0:19997 stopped Start Delete":
      - cell "959fd8d7"
      - cell "TCP"
      - cell "0.0.0.0:19997"
      - cell "stopped"
      - cell "Start Delete":
        - button "Start"
        - button "Delete"
    - row "184bbb4d TCP 0.0.0.0:19996 stopped Start Delete":
      - cell "184bbb4d"
      - cell "TCP"
      - cell "0.0.0.0:19996"
      - cell "stopped"
      - cell "Start Delete":
        - button "Start"
        - button "Delete"
```

# Test source

```ts
  36  |       // Open modal
  37  |       await page.locator('button:has-text("+ Create")').click();
  38  |       await page.waitForSelector('.modal-overlay');
  39  | 
  40  |       // Fill form
  41  |       await page.locator('.modal .form-input').first().fill(listenerName);
  42  |       await page.locator('.modal input[type="number"]').fill('19999');
  43  |       await page.locator('.modal button[type="submit"]').click();
  44  | 
  45  |       // Modal should close
  46  |       await page.waitForSelector('.modal-overlay', { state: 'detached', timeout: 5000 });
  47  | 
  48  |       // Listener should appear in the table
  49  |       await expect(page.locator('td').filter({ hasText: listenerName })).toBeVisible({ timeout: 5000 });
  50  | 
  51  |       // Verify status is "stopped"
  52  |       const row = page.locator('tr', { hasText: listenerName });
  53  |       await expect(row.locator('.badge-stopped')).toBeVisible();
  54  |     });
  55  |   });
  56  | 
  57  |   test.describe('TC-C2-M5-002: Start and Stop Listener', () => {
  58  |     test('should toggle between running and stopped', async ({ page }) => {
  59  |       // First create a listener
  60  |       const listenerName = `Toggle-${Date.now()}`;
  61  |       await page.locator('button:has-text("+ Create")').click();
  62  |       await page.waitForSelector('.modal-overlay');
  63  |       await page.locator('.modal .form-input').first().fill(listenerName);
  64  |       await page.locator('.modal input[type="number"]').fill('19998');
  65  |       await page.locator('.modal button[type="submit"]').click();
  66  |       await page.waitForSelector('.modal-overlay', { state: 'detached', timeout: 5000 });
  67  |       await page.waitForTimeout(500); // let table update
  68  | 
  69  |       // Find the row
  70  |       const row = page.locator('tr', { hasText: listenerName });
  71  | 
  72  |       // Should be stopped initially
  73  |       await expect(row.locator('.badge-stopped')).toBeVisible({ timeout: 5000 });
  74  | 
  75  |       // Click Start
  76  |       await row.locator('button:has-text("Start")').click();
  77  |       await page.waitForTimeout(1000); // wait for API + table update
  78  | 
  79  |       // Should be running now
  80  |       await expect(row.locator('.badge-running')).toBeVisible({ timeout: 5000 });
  81  | 
  82  |       // Click Stop
  83  |       await row.locator('button:has-text("Stop")').click();
  84  |       await page.waitForTimeout(1000);
  85  | 
  86  |       // Should be stopped again
  87  |       await expect(row.locator('.badge-stopped')).toBeVisible({ timeout: 5000 });
  88  |     });
  89  |   });
  90  | 
  91  |   test.describe('TC-C2-M5-003: Cannot Delete Running Listener', () => {
  92  |     test('should hide delete button when listener is running', async ({ page }) => {
  93  |       // Create a listener
  94  |       const listenerName = `NoDel-${Date.now()}`;
  95  |       await page.locator('button:has-text("+ Create")').click();
  96  |       await page.waitForSelector('.modal-overlay');
  97  |       await page.locator('.modal .form-input').first().fill(listenerName);
  98  |       await page.locator('.modal input[type="number"]').fill('19997');
  99  |       await page.locator('.modal button[type="submit"]').click();
  100 |       await page.waitForSelector('.modal-overlay', { state: 'detached', timeout: 5000 });
  101 |       await page.waitForTimeout(500);
  102 | 
  103 |       const row = page.locator('tr', { hasText: listenerName });
  104 | 
  105 |       // Start the listener
  106 |       await row.locator('button:has-text("Start")').click();
  107 |       await page.waitForTimeout(1000);
  108 |       await expect(row.locator('.badge-running')).toBeVisible({ timeout: 5000 });
  109 | 
  110 |       // Delete button should NOT be visible while running
  111 |       await expect(row.locator('.btn-danger')).not.toBeVisible();
  112 | 
  113 |       // Stop the listener
  114 |       await row.locator('button:has-text("Stop")').click();
  115 |       await page.waitForTimeout(1000);
  116 |       await expect(row.locator('.badge-stopped')).toBeVisible({ timeout: 5000 });
  117 | 
  118 |       // Delete button should now be visible
  119 |       await expect(row.locator('.btn-danger')).toBeVisible();
  120 |     });
  121 |   });
  122 | 
  123 |   test.describe('TC-C2-M5-004: Delete Stopped Listener', () => {
  124 |     test('should remove listener from table on delete', async ({ page }) => {
  125 |       // Create a listener
  126 |       const listenerName = `DelMe-${Date.now()}`;
  127 |       await page.locator('button:has-text("+ Create")').click();
  128 |       await page.waitForSelector('.modal-overlay');
  129 |       await page.locator('.modal .form-input').first().fill(listenerName);
  130 |       await page.locator('.modal input[type="number"]').fill('19996');
  131 |       await page.locator('.modal button[type="submit"]').click();
  132 |       await page.waitForSelector('.modal-overlay', { state: 'detached', timeout: 5000 });
  133 |       await page.waitForTimeout(500);
  134 | 
  135 |       // Confirm listener exists
> 136 |       await expect(page.locator('td').filter({ hasText: listenerName })).toBeVisible({ timeout: 5000 });
      |                                                                          ^ Error: expect(locator).toBeVisible() failed
  137 | 
  138 |       // Handle confirm dialog
  139 |       page.once('dialog', async (dialog) => {
  140 |         expect(dialog.type()).toBe('confirm');
  141 |         await dialog.accept();
  142 |       });
  143 | 
  144 |       // Click Delete
  145 |       const row = page.locator('tr', { hasText: listenerName });
  146 |       await row.locator('.btn-danger').click();
  147 | 
  148 |       // Wait for table update
  149 |       await page.waitForTimeout(1000);
  150 | 
  151 |       // Listener should be gone
  152 |       await expect(page.locator('td').filter({ hasText: listenerName })).not.toBeVisible({ timeout: 5000 });
  153 |     });
  154 |   });
  155 | });
  156 | 
```