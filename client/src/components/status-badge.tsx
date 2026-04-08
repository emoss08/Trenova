import { Badge, badgeVariants } from "@/components/ui/badge";
import { cn } from "@/lib/utils";
import type { BillingQueueStatus } from "@/types/billing-queue";
import type { InvoiceStatus } from "@/types/invoice";
import { shipmentStatusSchema, type ShipmentStatus } from "@/types/shipment";
import type { PTOStatus, PTOType } from "@/types/worker";
import type { VariantProps } from "class-variance-authority";
import {
  CheckCheckIcon,
  CheckIcon,
  ClockIcon,
  LockIcon,
  XIcon,
} from "lucide-react";
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
      description:
        "Shipment has been created and is pending initial assignment.",
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
      description:
        "Some moves within this shipment have been completed, but not all.",
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
      description:
        "Invoice has been generated and posted for completed transportation services.",
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
      description:
        "Shipment has been terminated and will not be completed as originally planned.",
    },
  };

  return (
    <Badge
      variant={statusAttributes[status].variant}
      className={cn(className, "max-h-6")}
    >
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
    <Badge
      variant={statusAttributes[status].variant}
      className={cn(className, "max-h-6")}
    >
      {statusAttributes[status].text}
    </Badge>
  );
}
export function PlainBillingQueueStatusBadge({
  status,
}: {
  status: BillingQueueStatus;
}) {
  const statusAttributes: Record<BillingQueueStatus, PlainBadgeAttrProps> = {
    ReadyForReview: {
      className: "bg-purple-600",
      text: "Ready for Review",
      description:
        "Invoice has been generated and is ready for review by the assigned biller.",
    },
    InReview: {
      className: "bg-indigo-600",
      text: "In Review",
      description: "Invoice is being reviewed by the assigned biller.",
    },
    Approved: {
      className: "bg-green-600",
      text: "Approved",
      description: "Invoice has been approved by the assigned biller.",
    },
    Posted: {
      className: "bg-teal-600",
      text: "Posted",
      description:
        "Invoice has been posted and this billing queue item is now historical.",
    },
    OnHold: {
      className: "bg-yellow-600",
      text: "On Hold",
      description:
        "Invoice is on hold and will be reviewed by the assigned biller when ready.",
    },
    SentBackToOps: {
      className: "bg-orange-600",
      text: "Sent Back to Ops",
      description:
        "Invoice has been sent back to operations for additional review.",
    },
    Exception: {
      className: "bg-amber-600",
      text: "Exception",
      description:
        "Invoice has an exception and will be reviewed by the assigned biller when ready.",
    },
    Canceled: {
      className: "bg-red-600",
      text: "Canceled",
      description:
        "Invoice has been canceled and will not be reviewed by the assigned biller.",
    },
  };

  return (
    <div className="flex items-center gap-x-1">
      <div
        className={cn(
          "size-2 rounded-full",
          statusAttributes[status].className,
        )}
      />
      <p className="text-xs font-medium">{statusAttributes[status].text}</p>
    </div>
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
    <Badge
      variant={statusAttributes[status].variant}
      className={cn(className, "max-h-6")}
    >
      {statusAttributes[status].text}
    </Badge>
  );
}

export function PlainInvoiceStatusBadge({ status }: { status: InvoiceStatus }) {
  const statusAttributes: Record<InvoiceStatus, PlainBadgeAttrProps> = {
    Draft: {
      className: "bg-slate-500",
      text: "Draft",
      description: "Invoice exists but has not been posted yet.",
    },
    Posted: {
      className: "bg-green-600",
      text: "Posted",
      description: "Invoice has been posted and the shipment has been billed.",
    },
  };

  return (
    <div className="flex items-center gap-x-1">
      <div
        className={cn(
          "size-2 rounded-full",
          statusAttributes[status].className,
        )}
      />
      <p className="text-xs font-medium">{statusAttributes[status].text}</p>
    </div>
  );
}
