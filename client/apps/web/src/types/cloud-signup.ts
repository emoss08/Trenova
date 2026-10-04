import { z } from "zod";

export const SIGNUP_MIN_PASSWORD_LENGTH = 12;
export const SIGNUP_MAX_PASSWORD_BYTES = 72;

const utf8 = new TextEncoder();

export type PasswordRequirement = {
  id: "length" | "not-email";
  label: string;
  met: boolean;
};

function emailLocalPart(emailAddress: string): string {
  return emailAddress.trim().toLowerCase().split("@")[0] ?? "";
}

/**
 * Mirrors the server's signup password policy, minus the breached-password list that
 * only the server holds: 12 or more characters, and neither the address itself nor
 * its local part. The server stays the authority; this exists so the person sees
 * the rule before a round trip burns a Turnstile token.
 */
export function passwordMatchesEmail(password: string, emailAddress: string): boolean {
  const candidate = password.trim().toLowerCase();
  if (candidate === "") {
    return false;
  }
  const email = emailAddress.trim().toLowerCase();
  if (email === "") {
    return false;
  }
  const local = emailLocalPart(email);
  return candidate === email || (local.length > 0 && candidate === local);
}

export function passwordRequirements(
  password: string,
  emailAddress: string,
): PasswordRequirement[] {
  return [
    {
      id: "length",
      label: `At least ${SIGNUP_MIN_PASSWORD_LENGTH} characters`,
      met: password.length >= SIGNUP_MIN_PASSWORD_LENGTH,
    },
    {
      id: "not-email",
      label: "Not your email address",
      met: password.length > 0 && !passwordMatchesEmail(password, emailAddress),
    },
  ];
}

export type PasswordStrength = 0 | 1 | 2 | 3 | 4;

/**
 * A coarse strength estimate for the meter under the password field: length carries
 * most of the weight, character variety the rest. It never blocks submission — the
 * requirements above do — it only tells the person whether they could do better.
 */
export function passwordStrength(password: string, emailAddress: string): PasswordStrength {
  if (password.length === 0 || passwordMatchesEmail(password, emailAddress)) {
    return 0;
  }

  const classes = [/[a-z]/, /[A-Z]/, /\d/, /[^A-Za-z0-9]/].filter((pattern) =>
    pattern.test(password),
  ).length;
  const distinct = new Set(password).size;
  const uniqueRatio = distinct / password.length;

  let score = 0;
  if (password.length >= SIGNUP_MIN_PASSWORD_LENGTH) score += 1;
  if (password.length >= 16) score += 1;
  if (classes >= 3) score += 1;
  if (password.length >= 20 || (classes === 4 && password.length >= 14)) score += 1;
  if (uniqueRatio < 0.4) score -= 1;
  if (distinct < 5) score = Math.min(score, 1);

  return Math.max(0, Math.min(4, score)) as PasswordStrength;
}

export const signupFormSchema = z
  .object({
    name: z
      .string()
      .trim()
      .min(1, { error: "Enter your name" })
      .max(100, { error: "Name must be 100 characters or fewer" }),
    emailAddress: z
      .string()
      .trim()
      .min(1, { error: "Enter your work email" })
      .pipe(z.email({ error: "Enter a valid email address" })),
    companyName: z
      .string()
      .trim()
      .min(1, { error: "Enter your company name" })
      .max(100, { error: "Company name must be 100 characters or fewer" }),
    password: z
      .string()
      .min(SIGNUP_MIN_PASSWORD_LENGTH, {
        error: `Password must be at least ${SIGNUP_MIN_PASSWORD_LENGTH} characters`,
      })
      .refine((value) => utf8.encode(value).length <= SIGNUP_MAX_PASSWORD_BYTES, {
        error: `Password must be ${SIGNUP_MAX_PASSWORD_BYTES} bytes or fewer`,
      }),
    acceptTerms: z.boolean().refine((value) => value, {
      error: "Accept the terms of service and privacy policy to continue",
    }),
    website: z.string(),
  })
  .superRefine((values, ctx) => {
    if (passwordMatchesEmail(values.password, values.emailAddress)) {
      ctx.addIssue({
        code: "custom",
        path: ["password"],
        message: "Password must not be your email address",
      });
    }
  });

export type SignupFormValues = z.input<typeof signupFormSchema>;

export const SIGNUP_FORM_DEFAULTS: SignupFormValues = {
  name: "",
  emailAddress: "",
  companyName: "",
  password: "",
  acceptTerms: false,
  website: "",
};

export type SignupRequest = {
  name: string;
  emailAddress: string;
  password: string;
  companyName: string;
  acceptTerms: boolean;
  turnstileToken: string;
  website: string;
};

export type SignupResendRequest = {
  emailAddress: string;
  turnstileToken: string;
};

export const signupAcceptedSchema = z.object({ status: z.string().optional() }).loose().nullish();
