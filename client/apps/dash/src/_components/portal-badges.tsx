import { Badge } from "@trenova/shared/components/ui/badge";

type BadgeVariant = React.ComponentProps<typeof Badge>["variant"];

const moveStatusVariants: Record<string, { label: string; variant: BadgeVariant }> = {
  New: { label: "New", variant: "secondary" },
  Assigned: { label: "Assigned", variant: "info" },
  InTransit: { label: "In Transit", variant: "purple" },
  Completed: { label: "Completed", variant: "active" },
  Canceled: { label: "Canceled", variant: "inactive" },
};

export function LoadStatusBadge({ status }: { status: string }) {
  const entry = moveStatusVariants[status] ?? { label: status, variant: "secondary" };
  return <Badge variant={entry.variant}>{entry.label}</Badge>;
}

const disputeStatusVariants: Record<string, { label: string; variant: BadgeVariant }> = {
  Open: { label: "Open", variant: "info" },
  InReview: { label: "In Review", variant: "warning" },
  Resolved: { label: "Resolved", variant: "active" },
  Denied: { label: "Denied", variant: "inactive" },
  Withdrawn: { label: "Withdrawn", variant: "secondary" },
};

export function DisputeStatusBadge({ status }: { status: string }) {
  const entry = disputeStatusVariants[status] ?? { label: status, variant: "secondary" };
  return <Badge variant={entry.variant}>{entry.label}</Badge>;
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
  Approved: { label: "Approved", variant: "active" },
  Rejected: { label: "Denied", variant: "inactive" },
  Cancelled: { label: "Cancelled", variant: "secondary" },
};

export function PtoStatusBadge({ status }: { status: string }) {
  const entry = ptoStatusVariants[status] ?? { label: status, variant: "secondary" };
  return <Badge variant={entry.variant}>{entry.label}</Badge>;
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
  Approved: { label: "Approved", variant: "active" },
  Rejected: { label: "Rejected", variant: "inactive" },
  Reimbursed: { label: "Reimbursed", variant: "active" },
  Cancelled: { label: "Cancelled", variant: "secondary" },
};

export function ExpenseStatusBadge({ status }: { status: string }) {
  const entry = expenseStatusVariants[status] ?? { label: status, variant: "secondary" };
  return <Badge variant={entry.variant}>{entry.label}</Badge>;
}
