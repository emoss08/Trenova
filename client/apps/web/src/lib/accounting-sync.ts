import type {
  AccountingConnectionStatus,
  AccountingSystem,
} from "@trenova/graphql/generated/graphql";
import type { StatusPhase } from "@trenova/shared/lib/status-phase";

export const ACCOUNTING_RECONNECT_WARNING_SECONDS = 14 * 24 * 60 * 60;

const CONNECTION_PHASES: Record<AccountingConnectionStatus, StatusPhase> = {
  Connected: "complete",
  Degraded: "attention",
  Failing: "failed",
  Revoked: "failed",
  Disconnected: "closed",
};

const AUTHORIZE_HOSTS: Record<AccountingSystem, readonly string[]> = {
  QuickBooksOnline: ["appcenter.intuit.com"],
};

export type AccountingCallback =
  | { kind: "authorized"; code: string; state: string; realmId: string }
  | { kind: "denied" }
  | { kind: "provider-error"; error: string }
  | { kind: "incomplete" };

export function accountingConnectionPhase(status: AccountingConnectionStatus): StatusPhase {
  return CONNECTION_PHASES[status];
}

export function hasLiveAccountingConnection(
  connection: { status: AccountingConnectionStatus } | null | undefined,
): boolean {
  return connection != null && connection.status !== "Disconnected";
}

export function isTrustedAuthorizeUrl(system: AccountingSystem, value: string): boolean {
  let url: URL;
  try {
    url = new URL(value);
  } catch {
    return false;
  }

  return (
    url.protocol === "https:" &&
    url.username === "" &&
    url.password === "" &&
    url.port === "" &&
    AUTHORIZE_HOSTS[system].includes(url.hostname)
  );
}

export function readAccountingCallback(params: URLSearchParams): AccountingCallback {
  const error = params.get("error")?.trim() ?? "";
  if (error === "access_denied") {
    return { kind: "denied" };
  }
  if (error !== "") {
    return { kind: "provider-error", error };
  }

  const code = params.get("code")?.trim() ?? "";
  const state = params.get("state")?.trim() ?? "";
  const realmId = params.get("realmId")?.trim() ?? "";
  if (code === "" || state === "" || realmId === "") {
    return { kind: "incomplete" };
  }

  return { kind: "authorized", code, state, realmId };
}

export function reconnectDeadlineNear(
  absoluteExpiresAt: number,
  nowSeconds = Math.floor(Date.now() / 1000),
): boolean {
  return absoluteExpiresAt - nowSeconds <= ACCOUNTING_RECONNECT_WARNING_SECONDS;
}

export function accountingSetupPath(system: AccountingSystem): string {
  return `/admin/integrations?type=${system}`;
}
