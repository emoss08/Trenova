import { Badge, type BadgeVariant } from "@/components/ui/badge";
import type { BankReceiptStatus } from "@/types/bank-receipt";
import type { BankReceiptBatchStatus } from "@/types/bank-receipt-batch";
import type { WorkItemStatus } from "@/types/bank-receipt-work-item";
import type { JournalReversalStatus } from "@/types/journal-reversal";
import type { ManualJournalStatus } from "@/types/manual-journal";

type AccountingStatus =
  | ManualJournalStatus
  | JournalReversalStatus
  | BankReceiptStatus
  | BankReceiptBatchStatus
  | WorkItemStatus;

const STATUS_VARIANT_MAP: Record<string, BadgeVariant> = {
  Draft: "secondary",
  Requested: "secondary",
  PendingApproval: "orange",
  Approved: "active",
  Rejected: "inactive",
  Cancelled: "secondary",
  Posted: "info",
  Processing: "orange",
  Completed: "active",
  Imported: "secondary",
  Matched: "active",
  Exception: "inactive",
  Open: "secondary",
  Assigned: "purple",
  InReview: "orange",
  Resolved: "active",
  Dismissed: "secondary",
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
