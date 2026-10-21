export const PASSWORD_MIN_LENGTH = 12;

export const PASSWORD_MAX_BYTES = 72;

const encoder = new TextEncoder();

export function passwordLength(password: string): number {
  return [...password].length;
}

export function passwordByteLength(password: string): number {
  return encoder.encode(password).length;
}

export function meetsPasswordLength(password: string): boolean {
  return (
    passwordLength(password) >= PASSWORD_MIN_LENGTH &&
    passwordByteLength(password) <= PASSWORD_MAX_BYTES
  );
}
