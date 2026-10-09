const CUSTOM_PREFIX = "custom:";
const KEY_ALPHABET = "abcdefghijklmnopqrstuvwxyz0123456789";
const KEY_LENGTH = 12;

/** Whether a checklist step is one the organization added. */
export function isCustomStep(key: string): boolean {
  return key.startsWith(CUSTOM_PREFIX);
}

/**
 * A key for a step the organization adds: random, lower case, and in the
 * shape the server accepts (`custom:` and 6–32 letters or digits), so it
 * stays the step's identity, and its ticks', across saves and renames.
 */
export function newCustomStepKey(): string {
  const bytes = new Uint8Array(KEY_LENGTH);
  crypto.getRandomValues(bytes);
  let key = CUSTOM_PREFIX;
  for (const byte of bytes) {
    key += KEY_ALPHABET[byte % KEY_ALPHABET.length];
  }

  return key;
}
