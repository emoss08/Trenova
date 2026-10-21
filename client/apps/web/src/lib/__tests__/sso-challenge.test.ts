import { describe, expect, it } from "vitest";
import { ssoChallengeFromHash } from "../sso-challenge";

describe("ssoChallengeFromHash", () => {
  it("reads the challenge the SSO callback put in the fragment", () => {
    expect(ssoChallengeFromHash("#mfa_challenge=challenge%2Ftoken%2B1&mfa_expires=1800000300")).toEqual(
      {
        mfaRequired: true,
        mfaChallengeToken: "challenge/token+1",
        mfaMethods: ["totp", "recovery_code"],
        expiresAt: 1_800_000_300,
      },
    );
  });

  it("ignores a fragment without a token or an expiry", () => {
    expect(ssoChallengeFromHash("")).toBeNull();
    expect(ssoChallengeFromHash("#mfa_expires=1800000300")).toBeNull();
    expect(ssoChallengeFromHash("#mfa_challenge=abc")).toBeNull();
    expect(ssoChallengeFromHash("#mfa_challenge=abc&mfa_expires=soon")).toBeNull();
    expect(ssoChallengeFromHash("#section-2")).toBeNull();
  });
});
