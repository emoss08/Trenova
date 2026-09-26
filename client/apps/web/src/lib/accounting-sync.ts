import { recordPath, type RecordEntityType } from "@/config/record-links";
import { API_BASE_URL } from "@trenova/shared/lib/constants";
import type {
  AccountingAppliedObjectType,
  AccountingConnectionStatus,
  AccountingDriftKind,
  AccountingDriftStatus,
  AccountingInboundChangeReason,
  AccountingInboundChangeStatus,
  AccountingInboundPaymentPolicy,
  AccountingSyncObjectType,
  AccountingSyncRecordStatus,
  AccountingMappingFilterInput,
  AccountingMappingState,
  AccountingMappingTargetType,
  AccountingReferenceKind,
  AccountingSetupStep,
  AccountingSystem,
} from "@trenova/graphql/generated/graphql";
import type { StatusPhase } from "@trenova/shared/lib/status-phase";
import { formatCurrency } from "@trenova/shared/lib/utils";

export const ACCOUNTING_MAPPINGS_PATH = "/accounting/sync/mappings";

export const ACCOUNTING_SYNC_PATH = "/accounting/sync";

export const ACCOUNTING_INBOUND_PATH = "/accounting/sync/inbound";

export const ACCOUNTING_DRIFT_PATH = "/accounting/sync/drift";

const DRIFT_PHASES: Record<AccountingDriftStatus, StatusPhase> = {
  Open: "attention",
  Resolved: "complete",
  Dismissed: "closed",
};

export function accountingDriftPhase(status: AccountingDriftStatus): StatusPhase {
  return DRIFT_PHASES[status];
}

export const ACCOUNTING_DRIFT_STATUSES: readonly AccountingDriftStatus[] = [
  "Open",
  "Resolved",
  "Dismissed",
];

export const ACCOUNTING_DRIFT_KINDS: readonly AccountingDriftKind[] = [
  "AmountMismatch",
  "StatusMismatch",
  "DeletedInProvider",
  "VoidedInProvider",
  "CustomerBalanceMismatch",
];

const MONEY_DRIFT_KINDS = new Set<AccountingDriftKind>([
  "AmountMismatch",
  "CustomerBalanceMismatch",
]);

export function accountingDriftWithinTolerance(
  finding: { kind: AccountingDriftKind; differenceMinor?: number | null },
  toleranceMinor: number,
): boolean {
  if (!MONEY_DRIFT_KINDS.has(finding.kind) || finding.differenceMinor == null) {
    return false;
  }
  return Math.abs(finding.differenceMinor) <= toleranceMinor;
}

const INBOUND_PHASES: Record<AccountingInboundChangeStatus, StatusPhase> = {
  Detected: "queued",
  Proposed: "awaiting",
  Applied: "complete",
  Ignored: "closed",
  Superseded: "closed",
};

export function accountingInboundPhase(status: AccountingInboundChangeStatus): StatusPhase {
  return INBOUND_PHASES[status];
}

export const ACCOUNTING_INBOUND_STATUSES: readonly AccountingInboundChangeStatus[] = [
  "Detected",
  "Proposed",
  "Applied",
  "Ignored",
  "Superseded",
];

export const ACCOUNTING_INBOUND_REASONS: readonly AccountingInboundChangeReason[] = [
  "PolicyPropose",
  "PeriodNotOpen",
  "UnknownDocument",
  "PartyMismatch",
  "Overpayment",
  "PartialBillPayment",
  "AlreadyPaid",
  "CurrencyMismatch",
  "Voided",
  "NotTrenovaDocument",
  "SentFromTrenova",
  "ApplyFailed",
];

const APPLICABLE_REASONS = new Set<AccountingInboundChangeReason>([
  "PolicyPropose",
  "PeriodNotOpen",
  "ApplyFailed",
]);

export function accountingInboundCanApply(
  status: AccountingInboundChangeStatus,
  reason: AccountingInboundChangeReason | null | undefined,
): boolean {
  return status === "Proposed" && reason != null && APPLICABLE_REASONS.has(reason);
}

export function accountingInboundIsOpen(status: AccountingInboundChangeStatus): boolean {
  return status === "Detected" || status === "Proposed";
}

export function accountingSyncObjectPath(
  objectType: AccountingSyncObjectType,
  objectId: string,
): string | null {
  switch (objectType) {
    case "Invoice":
    case "CreditMemo":
    case "DebitMemo":
      return recordPath("invoice", objectId);
    case "Customer":
      return recordPath("customer", objectId);
    case "CarrierBill":
    case "CarrierBillPayment":
      return recordPath("carrier_settlement", objectId);
    case "DriverBill":
    case "DriverBillPayment":
      return recordPath("driver_settlement", objectId);
    case "CarrierVendor":
      return recordPath("carrier", objectId);
    case "DriverVendor":
      return recordPath("worker", objectId);
    case "CustomerPayment":
    case "CreditApplication":
      return null;
  }
}

export function accountingAppliedObjectPath(
  type: AccountingAppliedObjectType,
  id: string,
): string | null {
  switch (type) {
    case "CarrierSettlement":
      return recordPath("carrier_settlement", id);
    case "DriverSettlement":
      return recordPath("driver_settlement", id);
    case "CustomerPayment":
    case "CreditMemoApplication":
      return null;
  }
}

export function formatAccountingMinor(minor: number, currency: string): string {
  return formatCurrency(minor / 100, currency || "USD");
}

export const ACCOUNTING_INBOUND_POLICIES: readonly AccountingInboundPaymentPolicy[] = [
  "Propose",
  "Apply",
  "Off",
];

const SYNC_RECORD_PHASES: Record<AccountingSyncRecordStatus, StatusPhase> = {
  Queued: "queued",
  AwaitingApproval: "awaiting",
  InFlight: "active",
  Retrying: "active",
  Synced: "complete",
  Blocked: "attention",
  DeadLettered: "failed",
  Skipped: "closed",
  Superseded: "closed",
};

export function accountingSyncRecordPhase(status: AccountingSyncRecordStatus): StatusPhase {
  return SYNC_RECORD_PHASES[status];
}

export function accountingSyncNeedsAction(status: AccountingSyncRecordStatus): boolean {
  return status === "Blocked" || status === "DeadLettered" || status === "AwaitingApproval";
}

export const REFERENCE_REFRESH_STALE_SECONDS = 2 * 60 * 60;

export const ACCOUNTING_MAPPING_TARGET_TYPES: readonly AccountingMappingTargetType[] = [
  "AccountRole",
  "LineType",
  "AccessorialCharge",
  "ItemRole",
  "Customer",
  "Carrier",
  "Driver",
  "GLAccount",
  "PaymentTerm",
  "PaymentMethod",
];

const MAPPING_PHASES: Record<AccountingMappingState, StatusPhase> = {
  Unmatched: "attention",
  Proposed: "awaiting",
  Confirmed: "complete",
};

const CREATABLE_KINDS: ReadonlySet<AccountingReferenceKind> = new Set([
  "Item",
  "Customer",
  "Vendor",
]);

const MAPPING_RECORD_ENTITIES: Partial<Record<AccountingMappingTargetType, RecordEntityType>> = {
  Customer: "customer",
  Carrier: "carrier",
};

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

export function accountingMappingPhase(state: AccountingMappingState): StatusPhase {
  return MAPPING_PHASES[state];
}

export function referenceRefreshRunning(
  connection: { referenceRefreshStartedAt?: number | null } | null | undefined,
  nowSeconds = Math.floor(Date.now() / 1000),
): boolean {
  const startedAt = connection?.referenceRefreshStartedAt;
  return startedAt != null && nowSeconds - startedAt <= REFERENCE_REFRESH_STALE_SECONDS;
}

export function needsAccountingMappings(
  connection:
    | { status: AccountingConnectionStatus; setupStep: AccountingSetupStep }
    | null
    | undefined,
): boolean {
  return hasLiveAccountingConnection(connection) && connection?.setupStep === "Mappings";
}

export function needsAccountingStartDate(
  connection:
    | { status: AccountingConnectionStatus; setupStep: AccountingSetupStep }
    | null
    | undefined,
): boolean {
  return hasLiveAccountingConnection(connection) && connection?.setupStep === "StartDate";
}

export function mappingCheckKey(mapping: { id: string; externalId: string }): string {
  return `${mapping.id}:${mapping.externalId}`;
}

export function checkedMappingConfirmations(
  mappings: ReadonlyArray<{
    id: string;
    externalId: string;
    state: AccountingMappingState;
    prechecked: boolean;
  }>,
  overrides: Readonly<Record<string, boolean>>,
): { id: string; externalId: string }[] {
  return mappings
    .filter(
      (mapping) =>
        mapping.state === "Proposed" && (overrides[mappingCheckKey(mapping)] ?? mapping.prechecked),
    )
    .map((mapping) => ({ id: mapping.id, externalId: mapping.externalId }));
}

export function accountingMappingRecordPath(mapping: {
  targetType: AccountingMappingTargetType;
  trenovaObjectId?: string | null;
}): string | null {
  const entity = MAPPING_RECORD_ENTITIES[mapping.targetType];
  if (!entity || !mapping.trenovaObjectId) {
    return null;
  }
  return recordPath(entity, mapping.trenovaObjectId);
}

export function accountingMappingFilterKey(filter: AccountingMappingFilterInput): string {
  return JSON.stringify({
    targetTypes: [...(filter.targetTypes ?? [])].sort(),
    states: [...(filter.states ?? [])].sort(),
    requiredOnly: filter.requiredOnly ?? false,
    search: filter.search?.trim() ?? "",
  });
}

export function accountingMappingCreatable(mapping: {
  providerKind: AccountingReferenceKind;
  state: AccountingMappingState;
}): boolean {
  return CREATABLE_KINDS.has(mapping.providerKind) && mapping.state !== "Confirmed";
}

export function accountingReferenceDetail(ref: {
  accountType: string;
  itemType: string;
  number: string;
  companyName: string;
  city: string;
  state: string;
}): string {
  return [ref.accountType || ref.itemType, ref.number, ref.companyName, ref.city, ref.state]
    .filter(Boolean)
    .join(" · ");
}

export function accountingWebhookUrl(
  webhookPath: string,
  apiBaseUrl = API_BASE_URL,
  origin = window.location.origin,
): string {
  if (!webhookPath) {
    return "";
  }
  const apiBase = apiBaseUrl.startsWith("http") ? apiBaseUrl : `${origin}${apiBaseUrl}`;
  return `${apiBase.replace(/\/+$/, "")}${webhookPath}`;
}
