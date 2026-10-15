import type {
  FactoringAccountRole,
  FactoringAccountingTreatment,
  FactoringAgreementStatus,
  FactoringFixAction,
  FactoringInvoiceEffect,
  FactoringNoaStatus,
  FactoringRecourse,
  FactoringSftpAuthMethod,
  FactoringSubmissionItemStatus,
  FactoringSubmissionStatus,
  FactoringTransportMethod,
  FactoringVerdict,
  InvoiceFactoringOverride,
} from "@trenova/graphql/generated/graphql";
import { defineLabels } from "@trenova/shared/i18n/labels";
import type { StatusPhase } from "@trenova/shared/lib/status-phase";

export const FACTORING_PATH = "/billing/factoring";

/** The most invoices one schedule of accounts carries; the server refuses more. */
export const FACTORING_SUBMISSION_MAX_INVOICES = 200;

/** Basis points in a whole: 10,000 bp is 100%. */
const BP_PER_WHOLE = 10_000;
const MINUTES_PER_HOUR = 60;
const MINUTES_PER_DAY = 24 * MINUTES_PER_HOUR;
const SECONDS_PER_DAY = 24 * 60 * 60;

export const FACTORING_VERDICT_LABELS = defineLabels({
  Ready: "Ready",
  NotReady: "Not ready",
  Submitted: "Submitted",
  Funded: "Funded",
  Excluded: "Not factored",
});

const VERDICT_PHASES: Record<FactoringVerdict, StatusPhase> = {
  Ready: "complete",
  NotReady: "attention",
  Submitted: "active",
  Funded: "awaiting",
  Excluded: "closed",
};

export function factoringVerdictPhase(verdict: FactoringVerdict): StatusPhase {
  return VERDICT_PHASES[verdict];
}

export const FACTORING_NOA_STATUS_LABELS = defineLabels({
  Draft: "Draft",
  Sent: "Sent",
  Acknowledged: "Acknowledged",
  Released: "Notice released",
  Expired: "Expired",
});

const NOA_PHASES: Record<FactoringNoaStatus | "Expired", StatusPhase> = {
  Draft: "draft",
  Sent: "awaiting",
  Acknowledged: "complete",
  Released: "closed",
  Expired: "failed",
};

/**
 * Where a notice stands as the customer sees it: an acknowledged notice whose
 * signed copy has lapsed no longer directs payment, so it reads as expired
 * even though nothing has changed its stored status.
 */
export function factoringNoaDisplayStatus(
  noa: { status: FactoringNoaStatus; expiresAt?: number | null },
  nowUnix: number,
): FactoringNoaStatus | "Expired" {
  if (noa.status === "Acknowledged" && noa.expiresAt != null && noa.expiresAt <= nowUnix) {
    return "Expired";
  }
  return noa.status;
}

export function factoringNoaPhase(status: FactoringNoaStatus | "Expired"): StatusPhase {
  return NOA_PHASES[status];
}

export const FACTORING_SUBMISSION_STATUS_LABELS = defineLabels({
  Sending: "Sending",
  Sent: "Awaiting funding",
  PartiallyFunded: "Partly funded",
  Funded: "Funded",
  Rejected: "Rejected",
  Failed: "Failed to send",
  Cancelled: "Cancelled",
});

const SUBMISSION_PHASES: Record<FactoringSubmissionStatus, StatusPhase> = {
  Sending: "queued",
  Sent: "awaiting",
  PartiallyFunded: "awaiting",
  Funded: "complete",
  Rejected: "failed",
  Failed: "failed",
  Cancelled: "closed",
};

/** A schedule the factor has received and not decided in full, so funding can be recorded on it. */
export function factoringSubmissionAwaitsFunding(status: FactoringSubmissionStatus): boolean {
  return status === "Sent" || status === "PartiallyFunded";
}

export function factoringSubmissionPhase(status: FactoringSubmissionStatus): StatusPhase {
  return SUBMISSION_PHASES[status];
}

export const FACTORING_SUBMISSION_ITEM_STATUS_LABELS = defineLabels({
  Pending: "On schedule",
  Released: "Released",
  Funded: "Funded",
  Rejected: "Rejected",
  Settled: "Settled",
  ChargedBack: "Charged back",
});

const SUBMISSION_ITEM_PHASES: Record<FactoringSubmissionItemStatus, StatusPhase> = {
  Pending: "active",
  Released: "closed",
  Funded: "awaiting",
  Rejected: "failed",
  Settled: "complete",
  ChargedBack: "attention",
};

export function factoringSubmissionItemPhase(status: FactoringSubmissionItemStatus): StatusPhase {
  return SUBMISSION_ITEM_PHASES[status];
}

export const FACTORING_RESERVE_ENTRY_KIND_LABELS = defineLabels({
  Funding: "Funding",
  ReserveRelease: "Reserve release",
  Chargeback: "Chargeback",
  ChargebackReversal: "Chargeback reversed",
});

export const FACTORING_ACCOUNT_ROLE_LABELS = defineLabels({
  AccountsReceivable: "Accounts receivable",
  Cash: "Cash",
  DueFromFactor: "Due from factor",
  FeeExpense: "Factoring fees",
  Liability: "Owed to factor",
});

export const FACTORING_INVOICE_EFFECT_LABELS = defineLabels({
  None: "Leaves the invoice's balance as it is.",
  Settle: "Settles the invoice's open balance.",
  Reopen: "Puts the invoice's balance back open to collect.",
});

export function factoringInvoiceEffectLabel(effect: FactoringInvoiceEffect): string {
  return FACTORING_INVOICE_EFFECT_LABELS[effect];
}

export function factoringAccountRoleLabel(role: FactoringAccountRole): string {
  return FACTORING_ACCOUNT_ROLE_LABELS[role];
}

/**
 * The movements a person can book on one invoice of a schedule, from where
 * the invoice stands: a funded invoice's reserve is released or charged back
 * (a chargeback only under recourse), and a charged-back invoice's last
 * chargeback can be reversed.
 */
export function factoringItemActions(
  status: FactoringSubmissionItemStatus,
  recourse: boolean,
): ("release" | "chargeback" | "reverse")[] {
  switch (status) {
    case "Funded":
      return recourse ? ["release", "chargeback"] : ["release"];
    case "ChargedBack":
      return ["reverse"];
    case "Pending":
    case "Released":
    case "Rejected":
    case "Settled":
      return [];
  }
}

export const FACTORING_TRANSPORT_METHOD_LABELS = defineLabels({
  Email: "Email",
  Sftp: "SFTP",
});

export const FACTORING_TRANSPORT_METHOD_DESCRIPTIONS = defineLabels({
  Email: "The schedule goes to the factor's inbox, with each invoice's packet attached.",
  Sftp: "The schedule and every packet are uploaded to a folder on the factor's server.",
});

export const FACTORING_TRANSPORT_METHOD_VALUES: readonly FactoringTransportMethod[] = [
  "Email",
  "Sftp",
];

export const FACTORING_SFTP_AUTH_METHOD_LABELS = defineLabels({
  Password: "Password",
  PrivateKey: "Private key",
});

export const FACTORING_SFTP_AUTH_METHOD_VALUES: readonly FactoringSftpAuthMethod[] = [
  "Password",
  "PrivateKey",
];

/** The most addresses one factor's schedules are emailed to; the server refuses more. */
export const FACTORING_EMAIL_RECIPIENT_LIMIT = 10;

export const DEFAULT_SFTP_PORT = 22;

/**
 * The factors a selection of queue rows spans, largest first, so a schedule
 * (which goes to one factor) starts from the factor most of the selection is with.
 */
export function groupInvoicesByFactor<
  T extends {
    invoiceId: string;
    factoringCompanyId?: string | null;
    factoringCompany?: { id: string; name: string } | null;
  },
>(rows: readonly T[]): { factoringCompanyId: string; name: string; invoiceIds: string[] }[] {
  const groups = new Map<
    string,
    { factoringCompanyId: string; name: string; invoiceIds: string[] }
  >();
  for (const row of rows) {
    const factorId = row.factoringCompanyId;
    if (!factorId) {
      continue;
    }
    const group = groups.get(factorId);
    if (group) {
      group.invoiceIds.push(row.invoiceId);
    } else {
      groups.set(factorId, {
        factoringCompanyId: factorId,
        name: row.factoringCompany?.name ?? factorId,
        invoiceIds: [row.invoiceId],
      });
    }
  }
  return [...groups.values()].sort(
    (a, b) => b.invoiceIds.length - a.invoiceIds.length || a.name.localeCompare(b.name),
  );
}

export const FACTORING_AGREEMENT_STATUS_LABELS = defineLabels({
  Draft: "Draft",
  Active: "Active",
  Ended: "Ended",
});

const AGREEMENT_PHASES: Record<FactoringAgreementStatus, StatusPhase> = {
  Draft: "draft",
  Active: "active",
  Ended: "closed",
};

export function factoringAgreementPhase(status: FactoringAgreementStatus): StatusPhase {
  return AGREEMENT_PHASES[status];
}

export const FACTORING_RECOURSE_LABELS = defineLabels({
  Recourse: "Recourse",
  NonRecourse: "Non-recourse",
});

export const FACTORING_RECOURSE_DESCRIPTIONS = defineLabels({
  Recourse: "The factor can charge an unpaid invoice back after the chargeback window.",
  NonRecourse: "The factor buys the invoice and carries the credit risk.",
});

export const FACTORING_ACCOUNTING_TREATMENT_LABELS = defineLabels({
  Sale: "Sale of receivables",
  SecuredBorrowing: "Secured borrowing",
});

export const FACTORING_ACCOUNTING_TREATMENT_DESCRIPTIONS = defineLabels({
  Sale: "Each funded invoice leaves receivables, with the fee as an expense.",
  SecuredBorrowing: "Receivables stay on the books and the advance is a liability to the factor.",
});

export const FACTORING_RECOURSE_VALUES: readonly FactoringRecourse[] = ["Recourse", "NonRecourse"];
export const FACTORING_ACCOUNTING_TREATMENT_VALUES: readonly FactoringAccountingTreatment[] = [
  "Sale",
  "SecuredBorrowing",
];

export const INVOICE_FACTORING_OVERRIDE_LABELS = defineLabels({
  Default: "Follow the customer",
  Exclude: "Keep out of factoring",
});

export const INVOICE_FACTORING_OVERRIDE_VALUES: readonly InvoiceFactoringOverride[] = [
  "Default",
  "Exclude",
];

export const FACTORING_FIX_ACTION_LABELS = defineLabels({
  SetUpAgreement: "Set up the agreement",
  SendNoa: "Send the notice",
  RecordNoaAcknowledgement: "Record the acknowledgement",
  ResolveDispute: "Resolve the dispute",
  RequestDocument: "Request documents",
  ReviewShipment: "Review the shipment",
});

export function factoringFixActionLabel(action: FactoringFixAction): string {
  return FACTORING_FIX_ACTION_LABELS[action];
}

/** A rate in basis points as a percentage a person reads: 8500 bp is "85%", 275 bp "2.75%". */
export function formatBasisPoints(bp: number): string {
  return `${(bp / 100).toLocaleString(undefined, { maximumFractionDigits: 2 })}%`;
}

/** A percentage a person typed, as whole basis points. */
export function percentToBasisPoints(percent: number): number {
  return Math.round(percent * 100);
}

export function basisPointsToPercent(bp: number): number {
  return bp / 100;
}

export const MAX_BASIS_POINTS = BP_PER_WHOLE;

/** Minutes after midnight as a 24-hour clock time ("09:30"). */
export function formatMinutesOfDay(minutes: number): string {
  const bounded = ((minutes % MINUTES_PER_DAY) + MINUTES_PER_DAY) % MINUTES_PER_DAY;
  const hours = Math.floor(bounded / MINUTES_PER_HOUR);
  const rest = bounded % MINUTES_PER_HOUR;
  return `${String(hours).padStart(2, "0")}:${String(rest).padStart(2, "0")}`;
}

/** A 24-hour clock time ("09:30") as minutes after midnight, or null when it is not one. */
export function parseMinutesOfDay(value: string): number | null {
  const match = /^([01]?\d|2[0-3]):([0-5]\d)$/.exec(value.trim());
  if (!match) {
    return null;
  }
  return Number(match[1]) * MINUTES_PER_HOUR + Number(match[2]);
}

/** Whether an invoice has been ready for longer than a day without being submitted. */
export function readyOverADay(readySince: number | null | undefined, nowUnix: number): boolean {
  return readySince != null && nowUnix - readySince >= SECONDS_PER_DAY;
}
