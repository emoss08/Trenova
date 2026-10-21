import type { MFAChallenge } from "@trenova/shared/types/mfa";

const TOKEN_KEY = "mfa_challenge";
const EXPIRES_KEY = "mfa_expires";

/**
 * The second-factor challenge an SSO callback hands to the sign-in page when the identity
 * provider's own MFA is not trusted. It travels in the URL fragment so the token never
 * reaches a server log or a Referer header.
 */
export function ssoChallengeFromHash(hash: string): MFAChallenge | null {
  const params = new URLSearchParams(hash.startsWith("#") ? hash.slice(1) : hash);
  const token = params.get(TOKEN_KEY)?.trim();
  const expiresAt = Number(params.get(EXPIRES_KEY));
  if (!token || !Number.isFinite(expiresAt) || expiresAt <= 0) {
    return null;
  }

  return {
    mfaRequired: true,
    mfaChallengeToken: token,
    mfaMethods: ["totp", "recovery_code"],
    expiresAt,
  };
}
