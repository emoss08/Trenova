import { translate } from "@trenova/shared/i18n/runtime";
import type { AuthEvent, RiskDecision } from "@trenova/shared/types/iam";
import { describe, expect, it } from "vitest";
import {
  authEventLabel,
  recentActivity,
} from "../_components/security-access/activity-copy";

/** An auth event as `GET /organizations/:id/iam/auth-events` returns it. */
function authEvent(overrides: Partial<AuthEvent>): AuthEvent {
  return {
    id: "ae_01",
    userId: "usr_01",
    organizationId: "org_01",
    businessUnitId: "bu_01",
    identityProviderId: "",
    provider: "password",
    outcome: "success",
    ipAddress: "203.0.113.7",
    userAgent: "",
    authenticatorAal: 1,
    federationFal: 1,
    mfaState: "",
    riskOutcome: "allow",
    riskSignals: [],
    errorCode: "",
    occurredAt: 1_760_000_000,
    createdAt: 1_760_000_000,
    ...overrides,
  } as AuthEvent;
}

function riskDecision(overrides: Partial<RiskDecision>): RiskDecision {
  return {
    id: "rd_01",
    userId: "usr_01",
    organizationId: "org_01",
    businessUnitId: "bu_01",
    outcome: "allow",
    signals: [],
    reason: "",
    createdAt: 1_760_000_000,
    ...overrides,
  } as RiskDecision;
}

describe("authEventLabel", () => {
  it("names a sign-out as one, not as a successful sign-in", () => {
    expect(authEventLabel(translate, authEvent({ provider: "session.logout" }))).toBe(
      "Signed out",
    );
  });

  it("says how a password sign-in went", () => {
    expect(authEventLabel(translate, authEvent({ outcome: "success" }))).toBe(
      "Signed in with a password",
    );
    expect(authEventLabel(translate, authEvent({ outcome: "failed" }))).toBe(
      "Password sign-in failed",
    );
  });

  it("names the identity provider a federated sign-in came through", () => {
    expect(authEventLabel(translate, authEvent({ provider: "Okta", outcome: "success" }))).toBe(
      "Signed in with Okta",
    );
  });

  it("tells a two-factor check that passed from one that did not", () => {
    expect(authEventLabel(translate, authEvent({ provider: "mfa", outcome: "success" }))).toBe(
      "Two-factor check passed",
    );
    expect(authEventLabel(translate, authEvent({ provider: "mfa", outcome: "failed" }))).toBe(
      "Two-factor check failed",
    );
  });
});

describe("recentActivity", () => {
  it("lists sign-ins and risk checks together, newest first, up to the limit", () => {
    const lines = recentActivity(
      translate,
      [
        authEvent({ id: "ae_old", occurredAt: 100 }),
        authEvent({ id: "ae_new", occurredAt: 300, outcome: "failed", errorCode: "bad_password" }),
      ],
      [riskDecision({ id: "rd_mid", outcome: "deny", reason: "Impossible travel", createdAt: 200 })],
      2,
    );

    expect(lines).toEqual([
      {
        id: "ae_new",
        label: "Password sign-in failed",
        detail: "bad_password",
        tone: "danger",
        occurredAt: 300,
      },
      {
        id: "rd_mid",
        label: "Risk check blocked a sign-in",
        detail: "Impossible travel",
        tone: "danger",
        occurredAt: 200,
      },
    ]);
  });

  it("leaves out a risk check that only allowed the sign-in beside it", () => {
    const lines = recentActivity(
      translate,
      [authEvent({ id: "ae_01" })],
      [riskDecision({ id: "rd_allow", outcome: "allow" })],
      5,
    );

    expect(lines.map((line) => line.id)).toEqual(["ae_01"]);
  });

  it("marks a sign-out as neutral rather than as a success", () => {
    const [line] = recentActivity(
      translate,
      [authEvent({ provider: "session.logout", outcome: "success" })],
      [],
      5,
    );

    expect(line?.tone).toBe("neutral");
  });
});
