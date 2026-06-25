import { Page } from '@playwright/test';

const API_BASE = 'http://localhost:5173/api/v1';

/**
 * Make a direct API request. If a token is provided, it's attached as Bearer.
 * The request object comes from Playwright's `page.request` (APIRequestContext).
 */
export async function apiRequest(
  page: Page,
  method: 'GET' | 'POST' | 'DELETE',
  path: string,
  options?: {
    body?: Record<string, unknown>;
    token?: string;
  }
) {
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
  };
  if (options?.token) {
    headers['Authorization'] = `Bearer ${options.token}`;
  }
  return page.request.fetch(`${API_BASE}${path}`, {
    method,
    headers,
    data: options?.body ? JSON.stringify(options.body) : undefined,
  });
}

/**
 * Login via direct API call (bypasses UI).
 * Returns { access_token, refresh_token }.
 */
export async function apiLogin(
  page: Page,
  username: string,
  password: string
): Promise<{ access_token: string; refresh_token: string }> {
  const resp = await page.request.post(`${API_BASE}/auth/login`, {
    data: { username, password },
    headers: { 'Content-Type': 'application/json' },
  });
  if (!resp.ok()) {
    throw new Error(`API login failed: ${resp.status()} ${await resp.text()}`);
  }
  const body = await resp.json();
  return { access_token: body.access_token, refresh_token: body.refresh_token };
}

/**
 * Refresh an access token via direct API call.
 */
export async function apiRefresh(
  page: Page,
  refreshToken: string
): Promise<{ access_token: string }> {
  const resp = await page.request.post(`${API_BASE}/auth/refresh`, {
    data: { refresh_token: refreshToken },
    headers: { 'Content-Type': 'application/json' },
  });
  return resp.json();
}
