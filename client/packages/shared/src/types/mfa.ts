import { z } from "zod";

export const mfaChallengeSchema = z.object({
  mfaRequired: z.literal(true),
  mfaChallengeToken: z.string().min(1),
  mfaMethods: z.array(z.string()).optional().default([]),
  expiresAt: z.number(),
});

export type MFAChallenge = z.infer<typeof mfaChallengeSchema>;

export const verifyMFAChallengeRequestSchema = z
  .object({
    challengeToken: z.string().min(1),
    code: z.string().optional().default(""),
    recoveryCode: z.string().optional().default(""),
  })
  .refine((value) => value.code.trim() !== "" || value.recoveryCode.trim() !== "", {
    error: "Enter the code from your authenticator app",
    path: ["code"],
  });

export type VerifyMFAChallengeRequest = z.input<typeof verifyMFAChallengeRequestSchema>;

export const mfaStatusSchema = z.object({
  totpEnabled: z.boolean(),
  enrollmentPending: z.boolean(),
  enabledAt: z.number().optional(),
  lastUsedAt: z.number().optional(),
  recoveryCodesRemaining: z.number(),
  sessionVerified: z.boolean(),
});

export type MFAStatus = z.infer<typeof mfaStatusSchema>;

export const totpEnrollmentSchema = z.object({
  authenticatorId: z.string(),
  secret: z.string(),
  uri: z.string(),
  qrCode: z.string(),
});

export type TOTPEnrollment = z.infer<typeof totpEnrollmentSchema>;

export const recoveryCodesSchema = z.object({
  recoveryCodes: z.array(z.string()),
});

export type RecoveryCodes = z.infer<typeof recoveryCodesSchema>;

export const totpCodeSchema = z
  .string()
  .trim()
  .regex(/^\d{3}\s?\d{3}$/, { error: "Enter the six-digit code from your authenticator app" });

export type DisableTOTPRequest = {
  password: string;
  code?: string;
  recoveryCode?: string;
};

/** True when a sign-in answered with a second-factor challenge instead of a session. */
export function isMFAChallenge(value: unknown): value is MFAChallenge {
  return (
    typeof value === "object" &&
    value !== null &&
    (value as { mfaRequired?: unknown }).mfaRequired === true
  );
}

/** Recovery codes as a plain-text file a person can keep somewhere safe. */
export function recoveryCodesFileContents(codes: readonly string[], issuer: string): string {
  return [`${issuer} recovery codes`, "", ...codes, ""].join("\n");
}
