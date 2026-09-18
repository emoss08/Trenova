import { Badge, type BadgeVariant } from "@trenova/shared/components/ui/badge";
import type { BankReceiptStatus } from "@/types/bank-receipt";
import type { BankReceiptBatchStatus } from "@/types/bank-receipt-batch";
import type { WorkItemStatus } from "@/types/bank-receipt-work-item";
import type { JournalReversalStatus } from "@/types/journal-reversal";
import type { ManualJournalStatus } from "@/types/manual-journal";

export type AccountingStatus =
  | ManualJournalStatus
  | JournalReversalStatus
  | BankReceiptStatus
  | BankReceiptBatchStatus
  | WorkItemStatus
  | (string & {});

const STATUS_VARIANT_MAP: Record<string, BadgeVariant> = {
  Draft: "neutral",
  Requested: "neutral",
  Pending: "warning",
  PendingApproval: "warning",
  Approved: "success",
  Rejected: "danger",
  Cancelled: "neutral",
  Posted: "info",
  Processing: "warning",
  Completed: "success",
  Imported: "neutral",
  Matched: "success",
  Exception: "danger",
  Open: "neutral",
  Assigned: "info",
  InReview: "warning",
  Resolved: "success",
  Dismissed: "neutral",
};

const STATUS_LABEL_MAP: Record<string, string> = {
  PendingApproval: "Pending Approval",
  InReview: "In Review",
};

export function AccountingStatusBadge({ status }: { status: AccountingStatus }) {
  const variant = STATUS_VARIANT_MAP[status] ?? "secondary";
  const label = STATUS_LABEL_MAP[status] ?? status;

  return <Badge variant={variant}>{label}</Badge>;
}
