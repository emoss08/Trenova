import { Badge, badgeVariants } from "@/components/ui/badge";
import { cn } from "@/lib/utils";
import type { BillingQueueStatus } from "@/types/billing-queue";
import type { InvoiceStatus } from "@/types/invoice";
import { shipmentStatusSchema, type ShipmentStatus } from "@/types/shipment";
import type { PTOStatus, PTOType } from "@/types/worker";
import type { VariantProps } from "class-variance-authority";
import { CheckCheckIcon, CheckIcon, ClockIcon, LockIcon, XIcon } from "lucide-react";
import type React from "react";

export type BadgeAttrProps = {
  variant: VariantProps<typeof badgeVariants>["variant"];
  text: string;
  description?: string;
  icon?: React.ReactNode;
};

type StatusBadgeProps = {
  status: string;
  className?: string;
};

export type PlainBadgeAttrProps = {
  text: string;
  description?: string;
  className?: string;
};

const STATUS_VARIANTS: Record<
  string,
  | "default"
  | "secondary"
  | "active"
  | "inactive"
  | "info"
  | "purple"
  | "orange"
  | "indigo"
  | "pink"
  | "teal"
  | "warning"
  | "outline"
> = {
  active: "active",
  inactive: "inactive",
  draft: "secondary",
  pending: "warning",
  completed: "default",
  cancelled: "inactive",
  processing: "secondary",
  // Compliance statuses
  compliant: "active",
  noncompliant: "inactive",
};

const STATUS_ICONS: Record<string, React.ReactNode> = {
  active: <CheckCheckIcon />,
  inactive: <XIcon />,
  draft: <ClockIcon />,
  pending: <ClockIcon />,
  completed: <CheckIcon />,
  cancelled: <XIcon />,
  processing: <ClockIcon />,
  // Compliance statuses
  compliant: <CheckCheckIcon />,
  noncompliant: <XIcon />,
};

export function StatusBadge({ status, className }: StatusBadgeProps) {
  const normalizedStatus = status.toLowerCase();
  const variant = STATUS_VARIANTS[normalizedStatus] || "outline";

  return (
    <Badge variant={variant} className={cn("capitalize", className)}>
      {STATUS_ICONS[normalizedStatus]}
      {status}
    </Badge>
  );
}

export function BooleanBadge({ value }: { value: boolean }) {
  return (
    <Badge variant={value ? "active" : "inactive"} className="max-h-6">
      {value ? "Yes" : "No"}
    </Badge>
  );
}

export function PTOStatusBadge({ status }: { status: PTOStatus }) {
  const ptoStatusAttrs: Record<PTOStatus, BadgeAttrProps> = {
    Requested: {
      variant: "purple",
      text: "Requested",
    },
    Approved: {
      variant: "active",
      text: "Approved",
    },
    Cancelled: {
      variant: "inactive",
      text: "Cancelled",
    },
    Rejected: {
      variant: "inactive",
      text: "Rejected",
    },
  };

  return (
    <Badge variant={ptoStatusAttrs[status].variant} className="max-h-6">
      {ptoStatusAttrs[status].text}
    </Badge>
  );
}

export function PermissionScopeBadge({ scope }: { scope?: string }) {
  if (!scope) {
    return "-";
  }

  const valueAttrs: Record<string, BadgeAttrProps> = {
    full: {
      text: "Full Access",
      variant: "secondary",
      icon: <CheckIcon />,
    },
    restricted: {
      text: "Restricted",
      variant: "secondary",
      icon: <LockIcon />,
    },
  };

  return (
    <Badge variant={valueAttrs[scope].variant} className="max-h-6">
      {valueAttrs[scope].icon}
      {valueAttrs[scope].text}
    </Badge>
  );
}

export function PTOTypeBadge({ type }: { type: PTOType }) {
  const ptoTypeAttributes: Record<PTOType, BadgeAttrProps> = {
    ["Personal"]: {
      variant: "secondary",
      text: "Personal",
    },
    ["Vacation"]: {
      variant: "purple",
      text: "Vacation",
    },
    ["Sick"]: {
      variant: "active",
      text: "Sick",
    },
    ["Holiday"]: {
      variant: "inactive",
      text: "Holiday",
    },
    ["Bereavement"]: {
      variant: "warning",
      text: "Bereavement",
    },
    ["Maternity"]: {
      variant: "warning",
      text: "Maternity",
    },
    ["Paternity"]: {
      variant: "warning",
      text: "Paternity",
    },
  };

  return (
    <Badge variant={ptoTypeAttributes[type].variant} className="max-h-6">
      {ptoTypeAttributes[type].text}
    </Badge>
  );
}

export function ShipmentStatusBadge({
  status,
  className,
}: {
  status?: ShipmentStatus;
  className?: string;
}) {
  if (!status) return null;

  const statusAttributes: Record<ShipmentStatus, BadgeAttrProps> = {
    ["New"]: {
      variant: "purple",
      text: "New",
      description: "Shipment has been created and is pending initial assignment.",
    },
    [shipmentStatusSchema.enum.PartiallyAssigned]: {
      variant: "indigo",
      text: "Partially Assigned",
      description:
        "Equipment or worker assignments are pending for one or more moves within this shipment.",
    },
    [shipmentStatusSchema.enum.PartiallyCompleted]: {
      variant: "indigo",
      text: "Partially Completed",
      description: "Some moves within this shipment have been completed, but not all.",
    },
    [shipmentStatusSchema.enum.Assigned]: {
      variant: "warning",
      text: "Assigned",
      description:
        "All required equipment and workers have been assigned to this shipment's moves.",
    },
    [shipmentStatusSchema.enum.InTransit]: {
      variant: "info",
      text: "In Transit",
      description:
        "Active shipment with cargo currently in transport between designated locations.",
    },
    [shipmentStatusSchema.enum.Delayed]: {
      variant: "orange",
      text: "Delayed",
      description:
        "Shipment has exceeded scheduled arrival or delivery timeframes at one or more stops.",
    },
    [shipmentStatusSchema.enum.Completed]: {
      variant: "active",
      text: "Completed",
      description:
        "All transportation activities for this shipment have been successfully completed.",
    },
    [shipmentStatusSchema.enum.Invoiced]: {
      variant: "teal",
      text: "Invoiced",
      description: "Invoice has been generated and posted for completed transportation services.",
    },
    [shipmentStatusSchema.enum.ReadyToInvoice]: {
      variant: "pink",
      text: "Ready to Invoice",
      description:
        "All moves within this shipment have been completed, and the shipment is ready to be invoiced.",
    },
    [shipmentStatusSchema.enum.Canceled]: {
      variant: "inactive",
      text: "Canceled",
      description: "Shipment has been terminated and will not be completed as originally planned.",
    },
  };

  return (
    <Badge variant={statusAttributes[status].variant} className={cn(className, "max-h-6")}>
      {statusAttributes[status].text}
    </Badge>
  );
}

export function BillingQueueStatusBadge({
  status,
  className,
}: {
  status?: BillingQueueStatus;
  className?: string;
}) {
  if (!status) return null;

  const statusAttributes: Record<BillingQueueStatus, BadgeAttrProps> = {
    ReadyForReview: {
      variant: "info",
      text: "Ready for Review",
    },
    InReview: {
      variant: "purple",
      text: "In Review",
    },
    Approved: {
      variant: "active",
      text: "Approved",
    },
    Posted: {
      variant: "teal",
      text: "Posted",
    },
    OnHold: {
      variant: "warning",
      text: "On Hold",
    },
    SentBackToOps: {
      variant: "orange",
      text: "Sent Back to Ops",
    },
    Exception: {
      variant: "inactive",
      text: "Exception",
    },
    Canceled: {
      variant: "inactive",
      text: "Canceled",
    },
  };

  return (
    <Badge variant={statusAttributes[status].variant} className={cn(className, "max-h-6")}>
      {statusAttributes[status].text}
    </Badge>
  );
}
export function PlainBillingQueueStatusBadge({ status }: { status: BillingQueueStatus }) {
  const statusAttributes: Record<BillingQueueStatus, PlainBadgeAttrProps> = {
    ReadyForReview: {
      className:
        "bg-blue-50 text-blue-700 dark:bg-blue-950 dark:text-blue-300",
      text: "Ready for Review",
    },
    InReview: {
      className:
        "bg-indigo-50 text-indigo-700 dark:bg-indigo-950 dark:text-indigo-300",
      text: "In Review",
    },
    Approved: {
      className:
        "bg-green-50 text-green-700 dark:bg-green-950 dark:text-green-300",
      text: "Approved",
    },
    Posted: {
      className:
        "bg-green-50 text-green-700 dark:bg-green-950 dark:text-green-300",
      text: "Posted",
    },
    OnHold: {
      className:
        "bg-amber-50 text-amber-700 dark:bg-amber-950 dark:text-amber-300",
      text: "On Hold",
    },
    SentBackToOps: {
      className:
        "bg-orange-50 text-orange-700 dark:bg-orange-950 dark:text-orange-300",
      text: "Sent Back to Ops",
    },
    Exception: {
      className:
        "bg-red-50 text-red-700 dark:bg-red-950 dark:text-red-300",
      text: "Exception",
    },
    Canceled: {
      className:
        "bg-gray-100 text-gray-500 dark:bg-gray-800 dark:text-gray-400",
      text: "Canceled",
    },
  };

  return (
    <span
      className={cn(
        "inline-flex items-center rounded-full px-2 py-0.5 text-[11px] font-medium",
        statusAttributes[status].className,
      )}
    >
      {statusAttributes[status].text}
    </span>
  );
}

export function InvoiceStatusBadge({
  status,
  className,
}: {
  status?: InvoiceStatus;
  className?: string;
}) {
  if (!status) return null;

  const statusAttributes: Record<InvoiceStatus, BadgeAttrProps> = {
    Draft: {
      variant: "secondary",
      text: "Draft",
    },
    Posted: {
      variant: "active",
      text: "Posted",
    },
  };

  return (
    <Badge variant={statusAttributes[status].variant} className={cn(className, "max-h-6")}>
      {statusAttributes[status].text}
    </Badge>
  );
}

export function PlainInvoiceStatusBadge({ status }: { status: InvoiceStatus }) {
  const statusAttributes: Record<InvoiceStatus, PlainBadgeAttrProps> = {
    Draft: {
      className:
        "bg-gray-100 text-gray-700 dark:bg-gray-800 dark:text-gray-300",
      text: "Draft",
    },
    Posted: {
      className:
        "bg-green-50 text-green-700 dark:bg-green-950 dark:text-green-300",
      text: "Posted",
    },
  };

  return (
    <span
      className={cn(
        "inline-flex items-center rounded-full px-2 py-0.5 text-[11px] font-medium",
        statusAttributes[status].className,
      )}
    >
      {statusAttributes[status].text}
    </span>
  );
}
