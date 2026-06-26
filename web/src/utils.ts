/**
 * Decode a base64-encoded UTF-8 string into a proper JavaScript string.
 *
 * Using atob() alone produces a "binary string" where each character's code
 * point equals a single byte value (0–255).  That breaks multi-byte UTF-8
 * sequences — Chinese / CJK characters render as Latin-1 mojibake.
 * TextDecoder correctly interprets the bytes as UTF-8.
 */
export function decodeBase64(b64: string): string {
  const binary = atob(b64);
  const bytes = Uint8Array.from(binary, (c) => c.charCodeAt(0));
  return new TextDecoder().decode(bytes);
}
