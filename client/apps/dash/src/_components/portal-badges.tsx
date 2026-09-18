import { useT } from "@trenova/shared/i18n/use-t";
import { Badge } from "@trenova/shared/components/ui/badge";

type BadgeVariant = React.ComponentProps<typeof Badge>["variant"];

const moveStatusVariants: Record<string, { label: string; variant: BadgeVariant }> = {
  New: { label: "New", variant: "neutral" },
  Assigned: { label: "Assigned", variant: "info" },
  InTransit: { label: "In Transit", variant: "info" },
  Completed: { label: "Completed", variant: "success" },
  Canceled: { label: "Canceled", variant: "danger" },
};

export function LoadStatusBadge({ status }: { status: string }) {
  const t = useT();

  const entry = moveStatusVariants[status] ?? { label: status, variant: "secondary" };
  return <Badge variant={entry.variant}>{t(entry.label)}</Badge>;
}

const disputeStatusVariants: Record<string, { label: string; variant: BadgeVariant }> = {
  Open: { label: "Open", variant: "info" },
  InReview: { label: "In Review", variant: "warning" },
  Resolved: { label: "Resolved", variant: "success" },
  Denied: { label: "Denied", variant: "danger" },
  Withdrawn: { label: "Withdrawn", variant: "neutral" },
};

export function DisputeStatusBadge({ status }: { status: string }) {
  const t = useT();

  const entry = disputeStatusVariants[status] ?? { label: status, variant: "secondary" };
  return <Badge variant={entry.variant}>{t(entry.label)}</Badge>;
}

export const disputeCategoryLabels: Record<string, string> = {
  MissingPay: "Missing pay",
  IncorrectRate: "Incorrect rate",
  IncorrectDeduction: "Incorrect deduction",
  MissingReimbursement: "Missing reimbursement",
  Other: "Something else",
};

const ptoStatusVariants: Record<string, { label: string; variant: BadgeVariant }> = {
  Requested: { label: "Requested", variant: "info" },
  Approved: { label: "Approved", variant: "success" },
  Rejected: { label: "Denied", variant: "danger" },
  Cancelled: { label: "Cancelled", variant: "neutral" },
};

export function PtoStatusBadge({ status }: { status: string }) {
  const t = useT();

  const entry = ptoStatusVariants[status] ?? { label: status, variant: "secondary" };
  return <Badge variant={entry.variant}>{t(entry.label)}</Badge>;
}

export const ptoTypeLabels: Record<string, string> = {
  Personal: "Personal",
  Vacation: "Vacation",
  Sick: "Sick",
  Holiday: "Holiday",
  Bereavement: "Bereavement",
  Maternity: "Maternity",
  Paternity: "Paternity",
};

const expenseStatusVariants: Record<string, { label: string; variant: BadgeVariant }> = {
  Pending: { label: "Pending", variant: "info" },
  Approved: { label: "Approved", variant: "success" },
  Rejected: { label: "Rejected", variant: "danger" },
  Reimbursed: { label: "Reimbursed", variant: "success" },
  Cancelled: { label: "Cancelled", variant: "neutral" },
};

export function ExpenseStatusBadge({ status }: { status: string }) {
  const t = useT();

  const entry = expenseStatusVariants[status] ?? { label: status, variant: "secondary" };
  return <Badge variant={entry.variant}>{t(entry.label)}</Badge>;
}
