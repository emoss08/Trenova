import type { AttentionSummaryQuery } from "@trenova/graphql/generated/graphql";

export type AttentionSummary = AttentionSummaryQuery["attentionSummary"];
export type AttentionTone = "default" | "warning" | "destructive";

export interface AttentionRowConfig {
  key: keyof AttentionSummary;
  label: string;
  module: string;
  path: string;
  tone: AttentionTone;
}

/**
 * Where each attention count lives and how urgently it reads. The sidebar and
 * the home screen both render these, so the label, destination, and tone are
 * defined once — two copies would let a route move in one place and not the
 * other.
 */
export const ATTENTION_ROWS: AttentionRowConfig[] = [
  {
    key: "billingQueue",
    label: "Billing Queue",
    module: "Billing",
    path: "/billing/queue",
    tone: "default",
  },
  {
    key: "pendingApprovals",
    label: "Pending Approvals",
    module: "Billing",
    path: "/billing/pending-approvals",
    tone: "default",
  },
  {
    key: "reconciliationExceptions",
    label: "Reconciliation Exceptions",
    module: "Billing",
    path: "/billing/reconciliation-exceptions",
    tone: "destructive",
  },
  {
    key: "serviceFailures",
    label: "Service Failures",
    module: "Shipments",
    path: "/shipment-management/service-failures",
    tone: "warning",
  },
  {
    key: "ediAttention",
    label: "EDI Needs Attention",
    module: "EDI",
    path: "/edi/overview",
    tone: "warning",
  },
];

export const ATTENTION_ROWS_BY_KEY = new Map(ATTENTION_ROWS.map((row) => [row.key as string, row]));

export const ATTENTION_TONE_DOT_CLASSES: Record<AttentionTone, string> = {
  default: "bg-info",
  warning: "bg-warning",
  destructive: "bg-destructive",
};
