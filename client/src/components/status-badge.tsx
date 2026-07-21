import { Badge, badgeVariants } from "@/components/ui/badge";
import { cn } from "@/lib/utils";
import type { BillingQueueStatus } from "@/types/billing-queue";
import type { CustomerPaymentStatus } from "@/types/customer-payment";
import type { InvoiceStatus, SettlementStatus } from "@/types/invoice";
import type { OrderStatus } from "@/types/order";
import {
  shipmentStatusSchema,
  type ShipmentStatus,
  type ShipmentTenderStatus,
} from "@/types/shipment";
import type {
  EDIInboundFileStatus,
  EDIMessageAcknowledgmentStatus,
  EDIMessageDeliveryStatus,
  EDITransferStatus,
} from "@/types/edi";
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
  inreview: "warning",
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
  inreview: <ClockIcon />,
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
    <Badge variant={value ? "active" : "inactive"} className="max-h-5">
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
    <Badge variant={ptoStatusAttrs[status].variant} className="max-h-5">
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
    <Badge variant={valueAttrs[scope].variant} className="max-h-5">
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
    <Badge variant={ptoTypeAttributes[type].variant} className="max-h-5">
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
    <Badge
      variant={statusAttributes[status].variant}
      className={cn(className, "max-h-5 uppercase")}
    >
      {statusAttributes[status].text}
    </Badge>
  );
}

export function ShipmentTenderStatusBadge({
  status,
  className,
}: {
  status?: ShipmentTenderStatus | null;
  className?: string;
}) {
  if (!status) return null;

  const statusAttributes: Record<ShipmentTenderStatus, BadgeAttrProps> = {
    Tendered: {
      variant: "info",
      text: "Tendered",
    },
    Accepted: {
      variant: "active",
      text: "Accepted",
    },
    Rejected: {
      variant: "inactive",
      text: "Rejected",
    },
    Expired: {
      variant: "orange",
      text: "Expired",
    },
    Canceled: {
      variant: "inactive",
      text: "Canceled",
    },
  };

  return (
    <Badge
      variant={statusAttributes[status].variant}
      className={cn(className, "max-h-5 uppercase")}
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
    <Badge variant={statusAttributes[status].variant} className={cn(className, "max-h-5")}>
      {statusAttributes[status].text}
    </Badge>
  );
}
export function PlainBillingQueueStatusBadge({ status }: { status: BillingQueueStatus }) {
  const statusAttributes: Record<BillingQueueStatus, PlainBadgeAttrProps> = {
    ReadyForReview: {
      className: "bg-blue-50 text-blue-700 dark:bg-blue-950 dark:text-blue-300",
      text: "Ready for Review",
    },
    InReview: {
      className: "bg-indigo-50 text-indigo-700 dark:bg-indigo-950 dark:text-indigo-300",
      text: "In Review",
    },
    Approved: {
      className: "bg-green-50 text-green-700 dark:bg-green-950 dark:text-green-300",
      text: "Approved",
    },
    Posted: {
      className: "bg-green-50 text-green-700 dark:bg-green-950 dark:text-green-300",
      text: "Posted",
    },
    OnHold: {
      className: "bg-amber-50 text-amber-700 dark:bg-amber-950 dark:text-amber-300",
      text: "On Hold",
    },
    SentBackToOps: {
      className: "bg-orange-50 text-orange-700 dark:bg-orange-950 dark:text-orange-300",
      text: "Sent Back to Ops",
    },
    Exception: {
      className: "bg-red-50 text-red-700 dark:bg-red-950 dark:text-red-300",
      text: "Exception",
    },
    Canceled: {
      className: "bg-gray-100 text-gray-500 dark:bg-gray-800 dark:text-gray-400",
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
    <Badge variant={statusAttributes[status].variant} className={cn(className, "max-h-5")}>
      {statusAttributes[status].text}
    </Badge>
  );
}

export function PlainSettlementStatusBadge({ status }: { status: SettlementStatus }) {
  const statusAttributes: Record<SettlementStatus, PlainBadgeAttrProps> = {
    Paid: {
      className: "bg-green-50 text-green-700 dark:bg-green-950 dark:text-green-300",
      text: "Paid",
    },
    PartiallyPaid: {
      className: "bg-amber-50 text-amber-700 dark:bg-amber-950 dark:text-amber-300",
      text: "Partial",
    },
    Unpaid: {
      className: "bg-gray-100 text-gray-500 dark:bg-gray-800 dark:text-gray-400",
      text: "Unpaid",
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

export function PlainCustomerPaymentStatusBadge({ status }: { status: CustomerPaymentStatus }) {
  const statusAttributes: Record<CustomerPaymentStatus, PlainBadgeAttrProps> = {
    Posted: {
      className: "bg-green-50 text-green-700 dark:bg-green-950 dark:text-green-300",
      text: "Posted",
    },
    Reversed: {
      className: "bg-red-50 text-red-700 dark:bg-red-950 dark:text-red-300",
      text: "Reversed",
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

export function PlainInvoiceStatusBadge({ status }: { status: InvoiceStatus }) {
  const statusAttributes: Record<InvoiceStatus, PlainBadgeAttrProps> = {
    Draft: {
      className: "bg-gray-100 text-gray-700 dark:bg-gray-800 dark:text-gray-300",
      text: "Draft",
    },
    Posted: {
      className: "bg-green-50 text-green-700 dark:bg-green-950 dark:text-green-300",
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

export function OrderStatusBadge({
  status,
  className,
}: {
  status?: OrderStatus;
  className?: string;
}) {
  if (!status) return null;

  const statusAttributes: Record<OrderStatus, BadgeAttrProps> = {
    Draft: {
      variant: "secondary",
      text: "Draft",
      description: "Order has been created but not yet confirmed.",
    },
    Confirmed: {
      variant: "purple",
      text: "Confirmed",
      description: "Order has been confirmed and is ready to be worked.",
    },
    InProgress: {
      variant: "info",
      text: "In Progress",
      description: "Order is actively being fulfilled.",
    },
    Completed: {
      variant: "active",
      text: "Completed",
      description: "Order fulfillment has been completed.",
    },
    Billed: {
      variant: "teal",
      text: "Billed",
      description: "Order has been billed to the customer.",
    },
    Closed: {
      variant: "outline",
      text: "Closed",
      description: "Order has been closed and finalized.",
    },
    Canceled: {
      variant: "inactive",
      text: "Canceled",
      description: "Order has been canceled and will not be fulfilled.",
    },
  };

  return (
    <Badge variant={statusAttributes[status].variant} className={cn(className, "max-h-5")}>
      {statusAttributes[status].text}
    </Badge>
  );
}

export function EDITransferStatusBadge({ status }: { status?: EDITransferStatus | string }) {
  if (!status) return null;

  const attrs: Record<EDITransferStatus, BadgeAttrProps> = {
    Submitted: {
      variant: "purple",
      text: "Submitted",
      description: "Tender has been submitted and is awaiting review by the receiving side.",
    },
    MappingRequired: {
      variant: "warning",
      text: "Mapping Required",
      description: "Tender references entities that are not mapped for this partner yet.",
    },
    PendingApproval: {
      variant: "info",
      text: "Pending Approval",
      description: "Tender is ready for the receiving organization to approve or reject.",
    },
    Processing: {
      variant: "secondary",
      text: "Processing",
      description: "Approval is running and the target shipment is being created.",
    },
    Approved: {
      variant: "active",
      text: "Approved",
      description: "Tender was accepted and the target shipment has been created.",
    },
    Rejected: {
      variant: "inactive",
      text: "Rejected",
      description: "Tender was rejected by the receiving side.",
    },
    Expired: {
      variant: "outline",
      text: "Expired",
      description: "Tender expired before it was actioned.",
    },
    Canceled: {
      variant: "outline",
      text: "Canceled",
      description: "Tender was canceled or superseded.",
    },
    Failed: {
      variant: "inactive",
      text: "Failed",
      description: "Tender processing failed. Review the failure reason for details.",
    },
  };
  const attr = attrs[status as EDITransferStatus];
  if (!attr) {
    return <Badge variant="outline">{status}</Badge>;
  }
  return (
    <Badge variant={attr.variant} className="max-h-5" title={attr.description}>
      {attr.text}
    </Badge>
  );
}

export function EDIPartnerReadinessBadge({
  ready,
  completedCount,
  totalCount,
}: {
  ready: boolean;
  completedCount: number;
  totalCount: number;
}) {
  if (ready) {
    return (
      <Badge
        variant="active"
        className="max-h-5"
        title="All onboarding checklist items are complete."
      >
        Ready
      </Badge>
    );
  }
  return (
    <Badge
      variant="warning"
      className="max-h-5 tabular-nums"
      title="Open the partner to see the remaining onboarding checklist items."
    >
      {completedCount}/{totalCount} ready
    </Badge>
  );
}

export function EDITestCaseVerdictBadge({ passed }: { passed: boolean }) {
  return (
    <Badge
      variant={passed ? "active" : "inactive"}
      className="max-h-5"
      title={
        passed
          ? "The preview diagnostics match the expected warning and error counts."
          : "The preview diagnostics do not match the expected warning and error counts."
      }
    >
      {passed ? "Pass" : "Fail"}
    </Badge>
  );
}

export function EDIMessageDeliveryStatusBadge({
  status,
}: {
  status?: EDIMessageDeliveryStatus | string | null;
}) {
  if (!status) return null;

  const attrs: Record<EDIMessageDeliveryStatus, BadgeAttrProps> = {
    Queued: {
      variant: "purple",
      text: "Queued",
      description: "Message is queued for delivery to the trading partner.",
    },
    Sending: {
      variant: "info",
      text: "Sending",
      description: "Delivery to the trading partner is in progress.",
    },
    Sent: {
      variant: "active",
      text: "Sent",
      description: "Message was delivered to the trading partner.",
    },
    Failed: {
      variant: "warning",
      text: "Failed",
      description: "The last delivery attempt failed. Retries are scheduled automatically.",
    },
    DeadLettered: {
      variant: "inactive",
      text: "Dead Lettered",
      description: "Delivery retries were exhausted. Retry manually after fixing the cause.",
    },
  };
  const attr = attrs[status as EDIMessageDeliveryStatus];
  if (!attr) {
    return <Badge variant="outline">{status}</Badge>;
  }
  return (
    <Badge variant={attr.variant} className="max-h-5" title={attr.description}>
      {attr.text}
    </Badge>
  );
}

export function EDIMessageAckStatusBadge({
  status,
}: {
  status?: EDIMessageAcknowledgmentStatus | string | null;
}) {
  if (!status) return null;

  const attrs: Record<EDIMessageAcknowledgmentStatus, BadgeAttrProps> = {
    NotExpected: {
      variant: "outline",
      text: "Not Expected",
      description: "No acknowledgment is expected for this message.",
    },
    Pending: {
      variant: "warning",
      text: "Ack Pending",
      description: "Waiting for the trading partner to acknowledge this message.",
    },
    Accepted: {
      variant: "active",
      text: "Accepted",
      description: "The trading partner acknowledged and accepted this message.",
    },
    Rejected: {
      variant: "inactive",
      text: "Rejected",
      description: "The trading partner rejected this message. Review the acknowledgment errors.",
    },
    Failed: {
      variant: "inactive",
      text: "Ack Failed",
      description: "Acknowledgment processing failed.",
    },
  };
  const attr = attrs[status as EDIMessageAcknowledgmentStatus];
  if (!attr) {
    return <Badge variant="outline">{status}</Badge>;
  }
  return (
    <Badge variant={attr.variant} className="max-h-5" title={attr.description}>
      {attr.text}
    </Badge>
  );
}

export function EDIInboundFileStatusBadge({ status }: { status?: EDIInboundFileStatus | string }) {
  if (!status) return null;

  const attrs: Record<EDIInboundFileStatus, BadgeAttrProps> = {
    Received: {
      variant: "purple",
      text: "Received",
      description: "File was pulled from the partner mailbox and is awaiting processing.",
    },
    Parsed: {
      variant: "info",
      text: "Parsed",
      description: "File envelope was parsed and transactions are being processed.",
    },
    Processed: {
      variant: "active",
      text: "Processed",
      description: "Every transaction in this file was processed successfully.",
    },
    PartiallyProcessed: {
      variant: "warning",
      text: "Partial",
      description: "Some transactions processed with warnings. Review the failure reason.",
    },
    Quarantined: {
      variant: "inactive",
      text: "Quarantined",
      description: "The file could not be processed. Fix the cause and reprocess.",
    },
    Duplicate: {
      variant: "outline",
      text: "Duplicate",
      description: "This interchange was already processed and was skipped.",
    },
  };
  const attr = attrs[status as EDIInboundFileStatus];
  if (!attr) {
    return <Badge variant="outline">{status}</Badge>;
  }
  return (
    <Badge variant={attr.variant} className="max-h-5" title={attr.description}>
      {attr.text}
    </Badge>
  );
}
