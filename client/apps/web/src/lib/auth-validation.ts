import { z } from "zod";

/** Something, an @, something, a dot, something; the server judges deliverability. */
const EMAIL_PATTERN = /^\S+@\S+\.\S+$/;

/**
 * An email address field, trimmed. Each screen words the refusal for its own reader —
 * sign-in asks for the invited address, signup for a work one — so the message is the
 * caller's, read when the form validates rather than when the module loads.
 */
export function emailSchema(message: () => string) {
  return z
    .string()
    .trim()
    .regex(EMAIL_PATTERN, { error: message });
}

export type PasswordScore = 0 | 1 | 2 | 3 | 4;

export const DEFAULT_PASSWORD_MIN_LENGTH = 8;

/** A password this long earns the last point without a symbol. */
const LONG_PASSWORD_LENGTH = 14;

/**
 * An advisory strength score for the meter under a new password: one point each for
 * reaching `minLength`, mixing upper and lower case, a digit, and a symbol (or a
 * length of 14 or more). It never blocks a submit; the schema does that.
 */
export function scorePassword(
  password: string,
  minLength: number = DEFAULT_PASSWORD_MIN_LENGTH,
): PasswordScore {
  const points = [
    password.length >= minLength,
    /[a-z]/.test(password) && /[A-Z]/.test(password),
    /\d/.test(password),
    /[^A-Za-z0-9]/.test(password) || password.length >= LONG_PASSWORD_LENGTH,
  ].filter(Boolean).length;
  return points as PasswordScore;
}
