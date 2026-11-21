import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { formatIdentityProviderName } from "@trenova/shared/lib/utils";
import type { AuthEvent, RiskDecision } from "@trenova/shared/types/iam";

export type ActivityTone = "success" | "warning" | "danger" | "neutral";

export type ActivityLine = {
  id: string;
  label: string;
  /** Where it came from: an address, an error code or a risk signal. */
  detail: string;
  tone: ActivityTone;
  occurredAt: number;
};

const OUTCOME_TONE: Record<AuthEvent["outcome"], ActivityTone> = {
  success: "success",
  challenge: "warning",
  denied: "danger",
  failed: "danger",
};

const RISK_TONE: Record<RiskDecision["outcome"], ActivityTone> = {
  allow: "success",
  challenge: "warning",
  deny: "danger",
};

/** The parts of an auth event its label is read from, whether it came from REST or GraphQL. */
export type AuthEventSummary = { provider: string; outcome: string };

function passwordLabel(t: TranslateFn, outcome: string): string {
  switch (outcome) {
    case "success":
      return t("Signed in with a password");
    case "challenge":
      return t("Asked for a second factor");
    case "denied":
      return t("Password sign-in refused");
    default:
      return t("Password sign-in failed");
  }
}

function providerLabel(t: TranslateFn, provider: string, outcome: string): string {
  const name = formatIdentityProviderName(provider);
  switch (outcome) {
    case "success":
      return t("Signed in with {0}", name);
    case "challenge":
      return t("{0} asked for a second factor", name);
    case "denied":
      return t("{0} sign-in refused", name);
    default:
      return t("{0} sign-in failed", name);
  }
}

/**
 * What an authentication event was, in words. The server records sign-outs, revoked sessions,
 * password resets, invitations and sign-ups as auth events too, each under a provider of its
 * own, so the provider says what happened before the outcome says how it went.
 */
export function authEventLabel(t: TranslateFn, event: AuthEventSummary): string {
  switch (event.provider) {
    case "session.logout":
      return t("Signed out");
    case "session.revoked":
      return t("Session revoked");
    case "password_reset.request":
      return t("Password reset requested");
    case "password_reset.confirm":
      return t("Password reset");
    case "invite.accept":
      return t("Invitation accepted");
    case "signup_requested":
      return t("Sign-up requested");
    case "signup_rejected":
      return t("Sign-up rejected");
    case "signup_verified":
      return t("Sign-up verified");
    case "signup_provisioned":
      return t("Account created");
    case "login_throttled":
      return t("Sign-in throttled");
    case "mfa":
      return event.outcome === "success"
        ? t("Two-factor check passed")
        : t("Two-factor check failed");
    case "password":
      return passwordLabel(t, event.outcome);
    default:
      return providerLabel(t, event.provider, event.outcome);
  }
}

function eventTone(event: AuthEvent): ActivityTone {
  if (event.riskOutcome === "deny") {
    return "danger";
  }
  if (event.provider === "login_throttled") {
    return "warning";
  }
  if (event.provider.startsWith("session.") || event.provider.startsWith("signup_")) {
    return "neutral";
  }
  return OUTCOME_TONE[event.outcome];
}

function riskLabel(t: TranslateFn, outcome: RiskDecision["outcome"]): string {
  switch (outcome) {
    case "allow":
      return t("Risk check allowed a sign-in");
    case "challenge":
      return t("Risk check asked for a second factor");
    case "deny":
      return t("Risk check blocked a sign-in");
  }
}

/**
 * The latest sign-in and risk signals as one list, newest first. A risk check that allowed the
 * sign-in it looked at says nothing the sign-in itself does not, so only one that challenged or
 * blocked is listed beside it.
 */
export function recentActivity(
  t: TranslateFn,
  authEvents: readonly AuthEvent[],
  riskDecisions: readonly RiskDecision[],
  limit: number,
): ActivityLine[] {
  return [
    ...authEvents.map((event) => ({
      id: event.id,
      label: authEventLabel(t, event),
      detail: event.errorCode || event.ipAddress || "",
      tone: eventTone(event),
      occurredAt: event.occurredAt,
    })),
    ...riskDecisions
      .filter((decision) => decision.outcome !== "allow")
      .map((decision) => ({
        id: decision.id,
        label: riskLabel(t, decision.outcome),
        detail: decision.reason || decision.signals.join(", "),
        tone: RISK_TONE[decision.outcome],
        occurredAt: decision.createdAt,
      })),
  ]
    .sort((left, right) => right.occurredAt - left.occurredAt)
    .slice(0, Math.max(0, limit));
}
