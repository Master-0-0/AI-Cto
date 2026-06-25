/**
 * Base64-decode a JWT payload (without validation).
 * Returns the parsed claims object, or null if decoding fails.
 */
export function decodeJwtPayload(token: string): Record<string, unknown> | null {
  try {
    const parts = token.split('.');
    if (parts.length !== 3) return null;
    const payload = parts[1];
    // Add padding and replace URL-safe chars
    const base64 = payload.replace(/-/g, '+').replace(/_/g, '/');
    const json = atob(base64);
    return JSON.parse(json);
  } catch {
    return null;
  }
}

/**
 * Create a tampered JWT by modifying the last character of the signature.
 */
export function tamperSignature(token: string): string {
  const parts = token.split('.');
  if (parts.length !== 3) return token;
  const sig = parts[2];
  // Flip the last character
  const lastChar = sig[sig.length - 1];
  const flipped = lastChar === 'A' ? 'B' : 'A';
  parts[2] = sig.slice(0, -1) + flipped;
  return parts.join('.');
}

/**
 * Create a JWT with a tampered payload claim.
 * Replaces a claim value, re-encodes, but does NOT re-sign.
 * The resulting token will have an invalid signature.
 */
export function tamperClaim(token: string, claim: string, newValue: string): string {
  const parts = token.split('.');
  if (parts.length !== 3) return token;
  const payload = decodeJwtPayload(token);
  if (!payload) return token;
  payload[claim] = newValue;
  const newPayload = btoa(JSON.stringify(payload))
    .replace(/=/g, '')
    .replace(/\+/g, '-')
    .replace(/\//g, '_');
  parts[1] = newPayload;
  return parts.join('.');
}

/**
 * A JWT that is structurally valid but expired in 2020.
 * Used for testing expired token behavior.
 */
export const EXPIRED_JWT =
  'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.' +
  'eyJvaWQiOiJ0ZXN0IiwidW5hbWUiOiJhZG1pbiIsInJvbGUiOiJhZG1pbiIsImV4cCI6MTU3NzgzNjgwMH0.' +
  'tampered-signature-placeholder';
