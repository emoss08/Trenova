import { useT } from "@trenova/shared/i18n/use-t";
import { Badge, badgeVariants } from "@trenova/shared/components/ui/badge";
import { cn } from "@trenova/shared/lib/utils";
import type { BillingQueueStatus } from "@trenova/shared/types/billing-queue";
import type { CustomerPaymentStatus } from "@trenova/shared/types/customer-payment";
import type {
  DriverPayEventStatus,
  DriverSettlementStatus,
  EscrowAccountStatus,
  PayAdvanceStatus,
  PayeeClassification,
  RecurringDeductionStatus,
  RecurringEarningStatus,
  SettlementBatchStatus,
} from "@trenova/shared/types/driver-pay";
import type { CarrierComplianceStatus, CarrierSafetyRating } from "@trenova/shared/types/carrier";
import type {
  CarrierCostEventStatus,
  CarrierInvoiceMatchStatus,
  CarrierSettlementBatchStatus,
  CarrierSettlementStatus,
} from "@trenova/shared/types/carrier-settlement";
import type { InvoiceStatus, SettlementStatus } from "@trenova/shared/types/invoice";
import type { RateConfirmationStatus } from "@trenova/shared/types/rate-confirmation";
import type { OrderStatus } from "@trenova/shared/types/order";
import {
  shipmentStatusSchema,
  type CarrierAssignmentStatus,
  type ShipmentStatus,
  type ShipmentTenderStatus,
} from "@trenova/shared/types/shipment";
import type {
  EDIInboundFileStatus,
  EDIMessageAcknowledgmentStatus,
  EDIMessageDeliveryStatus,
  EDITransferStatus,
} from "@trenova/shared/types/edi";
import type { TenderOfferStatus, TenderStatus } from "@trenova/shared/types/tender";
import type {
  FuelCardStatus,
  FuelPurchaseImportStatus,
  IftaReturnStatus,
} from "@trenova/shared/types/fuel-ifta-enums";
import { ptoTypeMeta } from "../lib/pto";
import type { PTOStatus, PTOType } from "@trenova/shared/types/worker";
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
  const t = useT();

  return (
    <Badge variant={value ? "active" : "inactive"} className="max-h-5">
      {value ? t("Yes") : t("No")}
    </Badge>
  );
}

export function PTOStatusBadge({ status }: { status: PTOStatus }) {
  const t = useT();

  const ptoStatusAttrs: Record<PTOStatus, BadgeAttrProps> = {
    Requested: {
      variant: "purple",
      text: t("Requested"),
    },
    Approved: {
      variant: "active",
      text: t("Approved"),
    },
    Cancelled: {
      variant: "inactive",
      text: t("Cancelled"),
    },
    Rejected: {
      variant: "inactive",
      text: t("Rejected"),
    },
  };

  return (
    <Badge variant={ptoStatusAttrs[status].variant} className="max-h-5">
      {ptoStatusAttrs[status].text}
    </Badge>
  );
}

export function PermissionScopeBadge({ scope }: { scope?: string }) {
  const t = useT();

  if (!scope) {
    return "-";
  }

  const valueAttrs: Record<string, BadgeAttrProps> = {
    full: {
      text: t("Full Access"),
      variant: "secondary",
      icon: <CheckIcon />,
    },
    restricted: {
      text: t("Restricted"),
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
  const t = useT();

  const meta = ptoTypeMeta(type);

  return (
    <Badge variant={meta.badgeVariant} className="max-h-5">
      {t(meta.label)}
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
  const t = useT();

  if (!status) return null;

  const statusAttributes: Record<ShipmentStatus, BadgeAttrProps> = {
    ["New"]: {
      variant: "purple",
      text: t("New"),
      description: t("Shipment has been created and is pending initial assignment."),
    },
    [shipmentStatusSchema.enum.PartiallyAssigned]: {
      variant: "indigo",
      text: t("Partially Assigned"),
      description: t(
        "Equipment or worker assignments are pending for one or more moves within this shipment.",
      ),
    },
    [shipmentStatusSchema.enum.PartiallyCompleted]: {
      variant: "indigo",
      text: t("Partially Completed"),
      description: t("Some moves within this shipment have been completed, but not all."),
    },
    [shipmentStatusSchema.enum.Assigned]: {
      variant: "warning",
      text: t("Assigned"),
      description: t(
        "All required equipment and workers have been assigned to this shipment's moves.",
      ),
    },
    [shipmentStatusSchema.enum.InTransit]: {
      variant: "info",
      text: t("In Transit"),
      description: t(
        "Active shipment with cargo currently in transport between designated locations.",
      ),
    },
    [shipmentStatusSchema.enum.Delayed]: {
      variant: "orange",
      text: t("Delayed"),
      description: t(
        "Shipment has exceeded scheduled arrival or delivery timeframes at one or more stops.",
      ),
    },
    [shipmentStatusSchema.enum.Completed]: {
      variant: "active",
      text: t("Completed"),
      description: t(
        "All transportation activities for this shipment have been successfully completed.",
      ),
    },
    [shipmentStatusSchema.enum.Invoiced]: {
      variant: "teal",
      text: t("Invoiced"),
      description: t(
        "Invoice has been generated and posted for completed transportation services.",
      ),
    },
    [shipmentStatusSchema.enum.ReadyToInvoice]: {
      variant: "pink",
      text: t("Ready to Invoice"),
      description: t(
        "All moves within this shipment have been completed, and the shipment is ready to be invoiced.",
      ),
    },
    [shipmentStatusSchema.enum.Canceled]: {
      variant: "inactive",
      text: t("Canceled"),
      description: t(
        "Shipment has been terminated and will not be completed as originally planned.",
      ),
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
  const t = useT();

  if (!status) return null;

  const statusAttributes: Record<ShipmentTenderStatus, BadgeAttrProps> = {
    Tendered: {
      variant: "info",
      text: t("Tendered"),
    },
    Accepted: {
      variant: "active",
      text: t("Accepted"),
    },
    Rejected: {
      variant: "inactive",
      text: t("Rejected"),
    },
    Expired: {
      variant: "orange",
      text: t("Expired"),
    },
    Canceled: {
      variant: "inactive",
      text: t("Canceled"),
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
  const t = useT();

  if (!status) return null;

  const statusAttributes: Record<BillingQueueStatus, BadgeAttrProps> = {
    ReadyForReview: {
      variant: "info",
      text: t("Ready for Review"),
    },
    InReview: {
      variant: "purple",
      text: t("In Review"),
    },
    Approved: {
      variant: "active",
      text: t("Approved"),
    },
    Posted: {
      variant: "teal",
      text: t("Posted"),
    },
    OnHold: {
      variant: "warning",
      text: t("On Hold"),
    },
    SentBackToOps: {
      variant: "orange",
      text: t("Sent Back to Ops"),
    },
    Exception: {
      variant: "inactive",
      text: t("Exception"),
    },
    Canceled: {
      variant: "inactive",
      text: t("Canceled"),
    },
  };

  return (
    <Badge variant={statusAttributes[status].variant} className={cn(className, "max-h-5")}>
      {statusAttributes[status].text}
    </Badge>
  );
}
export function PlainBillingQueueStatusBadge({ status }: { status: BillingQueueStatus }) {
  const t = useT();

  const statusAttributes: Record<BillingQueueStatus, PlainBadgeAttrProps> = {
    ReadyForReview: {
      className: "bg-blue-50 text-blue-700 dark:bg-blue-950 dark:text-blue-300",
      text: t("Ready for Review"),
    },
    InReview: {
      className: "bg-indigo-50 text-indigo-700 dark:bg-indigo-950 dark:text-indigo-300",
      text: t("In Review"),
    },
    Approved: {
      className: "bg-green-50 text-green-700 dark:bg-green-950 dark:text-green-300",
      text: t("Approved"),
    },
    Posted: {
      className: "bg-green-50 text-green-700 dark:bg-green-950 dark:text-green-300",
      text: t("Posted"),
    },
    OnHold: {
      className: "bg-amber-50 text-amber-700 dark:bg-amber-950 dark:text-amber-300",
      text: t("On Hold"),
    },
    SentBackToOps: {
      className: "bg-orange-50 text-orange-700 dark:bg-orange-950 dark:text-orange-300",
      text: t("Sent Back to Ops"),
    },
    Exception: {
      className: "bg-red-50 text-red-700 dark:bg-red-950 dark:text-red-300",
      text: t("Exception"),
    },
    Canceled: {
      className: "bg-gray-100 text-gray-500 dark:bg-gray-800 dark:text-gray-400",
      text: t("Canceled"),
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
  const t = useT();

  if (!status) return null;

  const statusAttributes: Record<InvoiceStatus, BadgeAttrProps> = {
    Draft: {
      variant: "secondary",
      text: t("Draft"),
    },
    Posted: {
      variant: "active",
      text: t("Posted"),
    },
  };

  return (
    <Badge variant={statusAttributes[status].variant} className={cn(className, "max-h-5")}>
      {statusAttributes[status].text}
    </Badge>
  );
}

export function PlainSettlementStatusBadge({ status }: { status: SettlementStatus }) {
  const t = useT();

  const statusAttributes: Record<SettlementStatus, PlainBadgeAttrProps> = {
    Paid: {
      className: "bg-green-50 text-green-700 dark:bg-green-950 dark:text-green-300",
      text: t("Paid"),
    },
    PartiallyPaid: {
      className: "bg-amber-50 text-amber-700 dark:bg-amber-950 dark:text-amber-300",
      text: t("Partial"),
    },
    Unpaid: {
      className: "bg-gray-100 text-gray-500 dark:bg-gray-800 dark:text-gray-400",
      text: t("Unpaid"),
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
  const t = useT();

  const statusAttributes: Record<CustomerPaymentStatus, PlainBadgeAttrProps> = {
    Posted: {
      className: "bg-green-50 text-green-700 dark:bg-green-950 dark:text-green-300",
      text: t("Posted"),
    },
    Reversed: {
      className: "bg-red-50 text-red-700 dark:bg-red-950 dark:text-red-300",
      text: t("Reversed"),
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
  const t = useT();

  const statusAttributes: Record<InvoiceStatus, PlainBadgeAttrProps> = {
    Draft: {
      className: "bg-gray-100 text-gray-700 dark:bg-gray-800 dark:text-gray-300",
      text: t("Draft"),
    },
    Posted: {
      className: "bg-green-50 text-green-700 dark:bg-green-950 dark:text-green-300",
      text: t("Posted"),
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
  const t = useT();

  if (!status) return null;

  const statusAttributes: Record<OrderStatus, BadgeAttrProps> = {
    Draft: {
      variant: "secondary",
      text: t("Draft"),
      description: t("Order has been created but not yet confirmed."),
    },
    Confirmed: {
      variant: "purple",
      text: t("Confirmed"),
      description: t("Order has been confirmed and is ready to be worked."),
    },
    InProgress: {
      variant: "info",
      text: t("In Progress"),
      description: t("Order is actively being fulfilled."),
    },
    Completed: {
      variant: "active",
      text: t("Completed"),
      description: t("Order fulfillment has been completed."),
    },
    Billed: {
      variant: "teal",
      text: t("Billed"),
      description: t("Order has been billed to the customer."),
    },
    Closed: {
      variant: "outline",
      text: t("Closed"),
      description: t("Order has been closed and finalized."),
    },
    Canceled: {
      variant: "inactive",
      text: t("Canceled"),
      description: t("Order has been canceled and will not be fulfilled."),
    },
  };

  return (
    <Badge variant={statusAttributes[status].variant} className={cn(className, "max-h-5")}>
      {statusAttributes[status].text}
    </Badge>
  );
}

export function EDITransferStatusBadge({ status }: { status?: EDITransferStatus | string }) {
  const t = useT();

  if (!status) return null;

  const attrs: Record<EDITransferStatus, BadgeAttrProps> = {
    Submitted: {
      variant: "purple",
      text: t("Submitted"),
      description: t("Tender has been submitted and is awaiting review by the receiving side."),
    },
    MappingRequired: {
      variant: "warning",
      text: t("Mapping Required"),
      description: t("Tender references entities that are not mapped for this partner yet."),
    },
    PendingApproval: {
      variant: "info",
      text: t("Pending Approval"),
      description: t("Tender is ready for the receiving organization to approve or reject."),
    },
    Processing: {
      variant: "secondary",
      text: t("Processing"),
      description: t("Approval is running and the target shipment is being created."),
    },
    Approved: {
      variant: "active",
      text: t("Approved"),
      description: t("Tender was accepted and the target shipment has been created."),
    },
    Rejected: {
      variant: "inactive",
      text: t("Rejected"),
      description: t("Tender was rejected by the receiving side."),
    },
    Expired: {
      variant: "outline",
      text: t("Expired"),
      description: t("Tender expired before it was actioned."),
    },
    Canceled: {
      variant: "outline",
      text: t("Canceled"),
      description: t("Tender was canceled or superseded."),
    },
    Failed: {
      variant: "inactive",
      text: t("Failed"),
      description: t("Tender processing failed. Review the failure reason for details."),
    },
  };
  const attr = attrs[status as EDITransferStatus];
  if (!attr) {
    return <Badge variant="outline">{status}</Badge>;
  }
  return (
    <Badge variant={attr.variant} className="max-h-5" title={t(attr.description)}>
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
  const t = useT();

  if (ready) {
    return (
      <Badge
        variant="active"
        className="max-h-5"
        title={t("All onboarding checklist items are complete.")}
      >
        {t("Ready")}
      </Badge>
    );
  }
  return (
    <Badge
      variant="warning"
      className="max-h-5 tabular-nums"
      title={t("Open the partner to see the remaining onboarding checklist items.")}
    >
      {t("{0}/{1} ready", completedCount, totalCount)}
    </Badge>
  );
}

export function EDITestCaseVerdictBadge({ passed }: { passed: boolean }) {
  const t = useT();

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
      {passed ? t("Pass") : t("Fail")}
    </Badge>
  );
}

export function EDIMessageDeliveryStatusBadge({
  status,
}: {
  status?: EDIMessageDeliveryStatus | string | null;
}) {
  const t = useT();

  if (!status) return null;

  const attrs: Record<EDIMessageDeliveryStatus, BadgeAttrProps> = {
    Queued: {
      variant: "purple",
      text: t("Queued"),
      description: t("Message is queued for delivery to the trading partner."),
    },
    Sending: {
      variant: "info",
      text: t("Sending"),
      description: t("Delivery to the trading partner is in progress."),
    },
    Sent: {
      variant: "active",
      text: t("Sent"),
      description: t("Message was delivered to the trading partner."),
    },
    Failed: {
      variant: "warning",
      text: t("Failed"),
      description: t("The last delivery attempt failed. Retries are scheduled automatically."),
    },
    DeadLettered: {
      variant: "inactive",
      text: t("Dead Lettered"),
      description: t("Delivery retries were exhausted. Retry manually after fixing the cause."),
    },
  };
  const attr = attrs[status as EDIMessageDeliveryStatus];
  if (!attr) {
    return <Badge variant="outline">{status}</Badge>;
  }
  return (
    <Badge variant={attr.variant} className="max-h-5" title={t(attr.description)}>
      {attr.text}
    </Badge>
  );
}

export function EDIMessageAckStatusBadge({
  status,
}: {
  status?: EDIMessageAcknowledgmentStatus | string | null;
}) {
  const t = useT();

  if (!status) return null;

  const attrs: Record<EDIMessageAcknowledgmentStatus, BadgeAttrProps> = {
    NotExpected: {
      variant: "outline",
      text: t("Not Expected"),
      description: t("No acknowledgment is expected for this message."),
    },
    Pending: {
      variant: "warning",
      text: t("Ack Pending"),
      description: t("Waiting for the trading partner to acknowledge this message."),
    },
    Accepted: {
      variant: "active",
      text: t("Accepted"),
      description: t("The trading partner acknowledged and accepted this message."),
    },
    Rejected: {
      variant: "inactive",
      text: t("Rejected"),
      description: t(
        "The trading partner rejected this message. Review the acknowledgment errors.",
      ),
    },
    Failed: {
      variant: "inactive",
      text: t("Ack Failed"),
      description: t("Acknowledgment processing failed."),
    },
  };
  const attr = attrs[status as EDIMessageAcknowledgmentStatus];
  if (!attr) {
    return <Badge variant="outline">{status}</Badge>;
  }
  return (
    <Badge variant={attr.variant} className="max-h-5" title={t(attr.description)}>
      {attr.text}
    </Badge>
  );
}

export function EDIInboundFileStatusBadge({ status }: { status?: EDIInboundFileStatus | string }) {
  const t = useT();

  if (!status) return null;

  const attrs: Record<EDIInboundFileStatus, BadgeAttrProps> = {
    Received: {
      variant: "purple",
      text: t("Received"),
      description: t("File was pulled from the partner mailbox and is awaiting processing."),
    },
    Parsed: {
      variant: "info",
      text: t("Parsed"),
      description: t("File envelope was parsed and transactions are being processed."),
    },
    Processed: {
      variant: "active",
      text: t("Processed"),
      description: t("Every transaction in this file was processed successfully."),
    },
    PartiallyProcessed: {
      variant: "warning",
      text: t("Partial"),
      description: t("Some transactions processed with warnings. Review the failure reason."),
    },
    Quarantined: {
      variant: "inactive",
      text: t("Quarantined"),
      description: t("The file could not be processed. Fix the cause and reprocess."),
    },
    Duplicate: {
      variant: "outline",
      text: t("Duplicate"),
      description: t("This interchange was already processed and was skipped."),
    },
  };
  const attr = attrs[status as EDIInboundFileStatus];
  if (!attr) {
    return <Badge variant="outline">{status}</Badge>;
  }
  return (
    <Badge variant={attr.variant} className="max-h-5" title={t(attr.description)}>
      {attr.text}
    </Badge>
  );
}

export function DriverSettlementStatusBadge({
  status,
  className,
}: {
  status: DriverSettlementStatus;
  className?: string;
}) {
  const t = useT();

  const statusAttributes: Record<DriverSettlementStatus, BadgeAttrProps> = {
    Draft: {
      variant: "secondary",
      text: t("Draft"),
      description: t(
        "Settlement is being assembled — pay, earnings, and deductions can still change.",
      ),
    },
    PendingApproval: {
      variant: "warning",
      text: t("Pending Approval"),
      description: t("Submitted for review and waiting on an approver."),
    },
    Approved: {
      variant: "info",
      text: t("Approved"),
      description: t("Approved and locked; deduction side effects have been applied."),
    },
    Posted: {
      variant: "purple",
      text: t("Posted"),
      description: t("Journalized to the general ledger and awaiting payment."),
    },
    Paid: {
      variant: "active",
      text: t("Paid"),
      description: t("Paid out to the driver; the settlement is final."),
    },
    Voided: {
      variant: "inactive",
      text: t("Voided"),
      description: t("Reversed — pay events returned to the pool and side effects were undone."),
    },
  };

  return (
    <Badge
      variant={statusAttributes[status].variant}
      className={cn("max-h-5", className)}
      title={t(statusAttributes[status].description)}
    >
      {statusAttributes[status].text}
    </Badge>
  );
}

export function SettlementBatchStatusBadge({ status }: { status: SettlementBatchStatus }) {
  const t = useT();

  const statusAttributes: Record<SettlementBatchStatus, BadgeAttrProps> = {
    Open: {
      variant: "info",
      text: t("Open"),
      description: t("Batch is accepting settlements; generation tops it up as pay accrues."),
    },
    Completed: {
      variant: "active",
      text: t("Completed"),
      description: t("Batch is closed; late accruals settle individually or in the next period."),
    },
    Canceled: {
      variant: "inactive",
      text: t("Canceled"),
      description: t("Batch was canceled and no longer collects settlements."),
    },
  };

  return (
    <Badge
      variant={statusAttributes[status].variant}
      className="max-h-5"
      title={t(statusAttributes[status].description)}
    >
      {statusAttributes[status].text}
    </Badge>
  );
}

export function PayAdvanceStatusBadge({ status }: { status: PayAdvanceStatus }) {
  const t = useT();

  const statusAttributes: Record<PayAdvanceStatus, BadgeAttrProps> = {
    Outstanding: {
      variant: "warning",
      text: t("Outstanding"),
      description: t("Nothing recovered yet — the full amount comes out of upcoming settlements."),
    },
    PartiallyRecovered: {
      variant: "info",
      text: t("Partially Recovered"),
      description: t(
        "Some of the advance has been recovered; the rest is withheld from future settlements.",
      ),
    },
    Recovered: {
      variant: "active",
      text: t("Recovered"),
      description: t("Fully recovered from the driver's settlements."),
    },
    WrittenOff: {
      variant: "inactive",
      text: t("Written Off"),
      description: t("Remaining balance was written off and will not be recovered."),
    },
  };

  return (
    <Badge
      variant={statusAttributes[status].variant}
      className="max-h-5"
      title={t(statusAttributes[status].description)}
    >
      {statusAttributes[status].text}
    </Badge>
  );
}

export function RecurringEarningStatusBadge({ status }: { status: RecurringEarningStatus }) {
  const t = useT();

  const statusAttributes: Record<RecurringEarningStatus, BadgeAttrProps> = {
    Active: {
      variant: "active",
      text: t("Active"),
      description: t("Added automatically to each qualifying settlement."),
    },
    Paused: {
      variant: "warning",
      text: t("Paused"),
      description: t("Temporarily skipped by settlements; resume to start paying again."),
    },
    Completed: {
      variant: "secondary",
      text: t("Completed"),
      description: t("Reached its lifetime cap and stopped permanently."),
    },
  };

  return (
    <Badge
      variant={statusAttributes[status].variant}
      className="max-h-5"
      title={t(statusAttributes[status].description)}
    >
      {statusAttributes[status].text}
    </Badge>
  );
}

export function RecurringDeductionStatusBadge({ status }: { status: RecurringDeductionStatus }) {
  const t = useT();

  const statusAttributes: Record<RecurringDeductionStatus, BadgeAttrProps> = {
    Active: {
      variant: "active",
      text: t("Active"),
      description: t("Withheld automatically from each qualifying settlement."),
    },
    Paused: {
      variant: "warning",
      text: t("Paused"),
      description: t("Temporarily skipped by settlements; resume to start withholding again."),
    },
    Completed: {
      variant: "secondary",
      text: t("Completed"),
      description: t("Reached its lifetime cap and stopped permanently."),
    },
  };

  return (
    <Badge
      variant={statusAttributes[status].variant}
      className="max-h-5"
      title={t(statusAttributes[status].description)}
    >
      {statusAttributes[status].text}
    </Badge>
  );
}

export function EscrowAccountStatusBadge({ status }: { status: EscrowAccountStatus }) {
  const t = useT();

  const statusAttributes: Record<EscrowAccountStatus, BadgeAttrProps> = {
    Active: {
      variant: "active",
      text: t("Active"),
      description: t("Accepting contributions and accruing interest per 49 CFR 376.12(k)."),
    },
    Closed: {
      variant: "secondary",
      text: t("Closed"),
      description: t("Closed out — the balance was refunded or applied."),
    },
  };

  return (
    <Badge
      variant={statusAttributes[status].variant}
      className="max-h-5"
      title={t(statusAttributes[status].description)}
    >
      {statusAttributes[status].text}
    </Badge>
  );
}

export function DriverPayEventStatusBadge({ status }: { status: DriverPayEventStatus }) {
  const t = useT();

  const statusAttributes: Record<DriverPayEventStatus, BadgeAttrProps> = {
    Accrued: {
      variant: "info",
      text: t("Accrued"),
      description: t("Earned but not yet on a settlement — waiting in the unsettled pool."),
    },
    Settled: {
      variant: "active",
      text: t("Settled"),
      description: t("Attached to a settlement as earning lines."),
    },
    Voided: {
      variant: "inactive",
      text: t("Voided"),
      description: t("Canceled (move canceled or reverted) and excluded from pay."),
    },
  };

  return (
    <Badge
      variant={statusAttributes[status].variant}
      className="max-h-5"
      title={t(statusAttributes[status].description)}
    >
      {statusAttributes[status].text}
    </Badge>
  );
}

export function PayeeClassificationBadge({
  classification,
}: {
  classification: PayeeClassification;
}) {
  const t = useT();

  const attributes: Record<PayeeClassification, BadgeAttrProps> = {
    CompanyDriver: {
      variant: "info",
      text: t("Company Driver"),
      description: t("W-2 employee — settlements post to the driver pay expense account."),
    },
    OwnerOperator: {
      variant: "purple",
      text: t("Owner-Operator"),
      description: t("1099 contractor — settlements post to the purchased transportation account."),
    },
  };

  return (
    <Badge
      variant={attributes[classification].variant}
      className="max-h-5"
      title={t(attributes[classification].description)}
    >
      {attributes[classification].text}
    </Badge>
  );
}

export function CarrierSettlementStatusBadge({
  status,
  className,
}: {
  status: CarrierSettlementStatus;
  className?: string;
}) {
  const t = useT();

  const statusAttributes: Record<CarrierSettlementStatus, BadgeAttrProps> = {
    Draft: {
      variant: "secondary",
      text: t("Draft"),
      description: t(
        "Statement is being assembled — cost events and adjustments can still change.",
      ),
    },
    PendingApproval: {
      variant: "warning",
      text: t("Pending Approval"),
      description: t("Submitted for review and waiting on an approver."),
    },
    Approved: {
      variant: "info",
      text: t("Approved"),
      description: t("Approved and locked, ready to post to the general ledger."),
    },
    Posted: {
      variant: "purple",
      text: t("Posted"),
      description: t(
        "Journalized as purchased transportation against accounts payable; awaiting payment.",
      ),
    },
    Paid: {
      variant: "active",
      text: t("Paid"),
      description: t(
        "Disbursed to the carrier — the cash journal and ledger payment are recorded.",
      ),
    },
    Voided: {
      variant: "inactive",
      text: t("Voided"),
      description: t("Reversed — cost events returned to the pool and GL postings were reversed."),
    },
  };

  return (
    <Badge
      variant={statusAttributes[status].variant}
      className={cn("max-h-5", className)}
      title={t(statusAttributes[status].description)}
    >
      {statusAttributes[status].text}
    </Badge>
  );
}

export function CarrierSettlementBatchStatusBadge({
  status,
}: {
  status: CarrierSettlementBatchStatus;
}) {
  const t = useT();

  const statusAttributes: Record<CarrierSettlementBatchStatus, BadgeAttrProps> = {
    Open: {
      variant: "info",
      text: t("Open"),
      description: t("AP run is accepting settlements; generation tops it up as cost accrues."),
    },
    Completed: {
      variant: "active",
      text: t("Completed"),
      description: t("AP run is closed; late accruals settle in the next period."),
    },
    Canceled: {
      variant: "inactive",
      text: t("Canceled"),
      description: t("Batch was canceled and no longer collects settlements."),
    },
  };

  return (
    <Badge
      variant={statusAttributes[status].variant}
      className="max-h-5"
      title={t(statusAttributes[status].description)}
    >
      {statusAttributes[status].text}
    </Badge>
  );
}

export function CarrierCostEventStatusBadge({ status }: { status: CarrierCostEventStatus }) {
  const t = useT();

  const statusAttributes: Record<CarrierCostEventStatus, BadgeAttrProps> = {
    Pending: {
      variant: "info",
      text: t("Pending"),
      description: t("Accrued purchased-transportation cost not yet on a settlement."),
    },
    Attached: {
      variant: "warning",
      text: t("Attached"),
      description: t("On a draft settlement — locked until the settlement is processed or voided."),
    },
    Settled: {
      variant: "active",
      text: t("Settled"),
      description: t("Included on a posted carrier settlement."),
    },
    Voided: {
      variant: "inactive",
      text: t("Voided"),
      description: t("Canceled (assignment or shipment canceled) and excluded from settlement."),
    },
  };

  return (
    <Badge
      variant={statusAttributes[status].variant}
      className="max-h-5"
      title={t(statusAttributes[status].description)}
    >
      {statusAttributes[status].text}
    </Badge>
  );
}

export function CarrierInvoiceMatchStatusBadge({
  status,
  className,
}: {
  status: CarrierInvoiceMatchStatus;
  className?: string;
}) {
  const t = useT();

  const statusAttributes: Record<CarrierInvoiceMatchStatus, BadgeAttrProps> = {
    Suggested: {
      variant: "secondary",
      text: t("Suggested"),
      description: t("System-proposed pairing of a carrier invoice with an assignment."),
    },
    Matched: {
      variant: "info",
      text: t("Matched"),
      description: t("Invoice total agrees with the negotiated buy rate within tolerance."),
    },
    Variance: {
      variant: "warning",
      text: t("Variance"),
      description: t("Invoice total differs from the buy rate beyond the configured tolerance."),
    },
    Resolved: {
      variant: "active",
      text: t("Resolved"),
      description: t("Accepted — the invoice is reconciled against the assignment."),
    },
    Rejected: {
      variant: "inactive",
      text: t("Rejected"),
      description: t("Dismissed — the invoice does not bill this assignment."),
    },
  };

  return (
    <Badge
      variant={statusAttributes[status].variant}
      className={cn("max-h-5", className)}
      title={t(statusAttributes[status].description)}
    >
      {statusAttributes[status].text}
    </Badge>
  );
}

export function CarrierComplianceStatusBadge({
  status,
  className,
}: {
  status: CarrierComplianceStatus;
  className?: string;
}) {
  const t = useT();

  const statusAttributes: Record<CarrierComplianceStatus, BadgeAttrProps> = {
    Pending: {
      variant: "warning",
      text: t("Pending"),
      description: t("Compliance review has not been completed for this carrier."),
    },
    Qualified: {
      variant: "active",
      text: t("Qualified"),
      description: t("The carrier passed the compliance review and can be assigned freight."),
    },
    Disqualified: {
      variant: "inactive",
      text: t("Disqualified"),
      description: t("The carrier failed compliance and must not be assigned freight."),
    },
    Expired: {
      variant: "inactive",
      text: t("Expired"),
      description: t("The carrier's qualification lapsed and must be renewed before assignment."),
    },
  };

  return (
    <Badge
      variant={statusAttributes[status].variant}
      className={cn("max-h-5", className)}
      title={t(statusAttributes[status].description)}
    >
      {statusAttributes[status].text}
    </Badge>
  );
}

export function CarrierSafetyRatingBadge({
  status,
  className,
}: {
  status: CarrierSafetyRating;
  className?: string;
}) {
  const t = useT();

  const statusAttributes: Record<CarrierSafetyRating, BadgeAttrProps> = {
    Satisfactory: {
      variant: "active",
      text: t("Satisfactory"),
      description: t("FMCSA rated the carrier satisfactory."),
    },
    Conditional: {
      variant: "warning",
      text: t("Conditional"),
      description: t("FMCSA found deficiencies — review before assigning freight."),
    },
    Unsatisfactory: {
      variant: "inactive",
      text: t("Unsatisfactory"),
      description: t("FMCSA rated the carrier unsatisfactory — do not assign freight."),
    },
    NotRated: {
      variant: "secondary",
      text: t("Not Rated"),
      description: t("FMCSA has not issued a safety rating for this carrier."),
    },
  };

  return (
    <Badge
      variant={statusAttributes[status].variant}
      className={cn("max-h-5", className)}
      title={t(statusAttributes[status].description)}
    >
      {statusAttributes[status].text}
    </Badge>
  );
}

export function CarrierAssignmentStatusBadge({
  status,
  className,
}: {
  status: CarrierAssignmentStatus;
  className?: string;
}) {
  const t = useT();

  const statusAttributes: Record<CarrierAssignmentStatus, BadgeAttrProps> = {
    Pending: {
      variant: "warning",
      text: t("Pending"),
      description: t("The carrier has been assigned but has not confirmed the rate yet."),
    },
    Confirmed: {
      variant: "active",
      text: t("Confirmed"),
      description: t("The carrier confirmed the negotiated rate for this move."),
    },
    Canceled: {
      variant: "inactive",
      text: t("Canceled"),
      description: t("The carrier assignment was canceled or replaced."),
    },
  };

  return (
    <Badge
      variant={statusAttributes[status].variant}
      className={cn("max-h-5", className)}
      title={t(statusAttributes[status].description)}
    >
      {statusAttributes[status].text}
    </Badge>
  );
}

export function TenderStatusBadge({
  status,
  className,
}: {
  status?: TenderStatus | null;
  className?: string;
}) {
  const t = useT();

  if (!status) return null;

  const statusAttributes: Record<TenderStatus, BadgeAttrProps> = {
    Active: {
      variant: "info",
      text: t("Active"),
      description: t("Offers are out to carriers and a response is pending."),
    },
    Accepted: {
      variant: "active",
      text: t("Accepted"),
      description: t("A carrier accepted the tender and the move is covered."),
    },
    Exhausted: {
      variant: "orange",
      text: t("Exhausted"),
      description: t("Every carrier declined or timed out — the move is still uncovered."),
    },
    Canceled: {
      variant: "inactive",
      text: t("Canceled"),
      description: t("The tender was canceled by a dispatcher."),
    },
    NeedsReview: {
      variant: "warning",
      text: t("Needs Review"),
      description: t(
        "A carrier accepted but auto-assignment failed — assign the move manually or cancel.",
      ),
    },
  };

  return (
    <Badge
      variant={statusAttributes[status].variant}
      className={cn("max-h-5", className)}
      title={t(statusAttributes[status].description)}
    >
      {statusAttributes[status].text}
    </Badge>
  );
}

export function TenderOfferStatusBadge({
  status,
  className,
}: {
  status?: TenderOfferStatus | null;
  className?: string;
}) {
  const t = useT();

  if (!status) return null;

  const statusAttributes: Record<TenderOfferStatus, BadgeAttrProps> = {
    Pending: {
      variant: "secondary",
      text: t("Pending"),
      description: t("Queued behind a higher-ranked carrier; nothing has been sent yet."),
    },
    Sent: {
      variant: "info",
      text: t("Sent"),
      description: t("Delivered to the carrier and awaiting their response."),
    },
    Accepted: {
      variant: "active",
      text: t("Accepted"),
      description: t("The carrier accepted this offer."),
    },
    Declined: {
      variant: "inactive",
      text: t("Declined"),
      description: t("The carrier declined this offer."),
    },
    Expired: {
      variant: "orange",
      text: t("Expired"),
      description: t("The offer window elapsed without a response."),
    },
    Withdrawn: {
      variant: "outline",
      text: t("Withdrawn"),
      description: t("The offer was withdrawn when the tender was canceled."),
    },
    Superseded: {
      variant: "outline",
      text: t("Superseded"),
      description: t("Another carrier accepted first; this offer no longer stands."),
    },
    Skipped: {
      variant: "outline",
      text: t("Skipped"),
      description: t("The waterfall skipped this carrier."),
    },
    DeliveryFailed: {
      variant: "inactive",
      text: t("Delivery Failed"),
      description: t("The offer could not be delivered on its channel."),
    },
  };

  return (
    <Badge
      variant={statusAttributes[status].variant}
      className={cn("max-h-5", className)}
      title={t(statusAttributes[status].description)}
    >
      {statusAttributes[status].text}
    </Badge>
  );
}

export function RateConfirmationStatusBadge({
  status,
  className,
}: {
  status: RateConfirmationStatus;
  className?: string;
}) {
  const t = useT();

  const statusAttributes: Record<RateConfirmationStatus, BadgeAttrProps> = {
    Generated: {
      variant: "secondary",
      text: t("Generated"),
      description: t("Rate confirmation PDF is filed but has not been sent to the carrier."),
    },
    Sent: {
      variant: "info",
      text: t("Sent"),
      description: t("Emailed to the carrier's rate confirmation contacts."),
    },
    Confirmed: {
      variant: "active",
      text: t("Confirmed"),
      description: t("The carrier confirmed the negotiated rate."),
    },
    Voided: {
      variant: "inactive",
      text: t("Voided"),
      description: t("Superseded by a newer revision or voided manually."),
    },
  };

  return (
    <Badge
      variant={statusAttributes[status].variant}
      className={cn("max-h-5", className)}
      title={t(statusAttributes[status].description)}
    >
      {statusAttributes[status].text}
    </Badge>
  );
}

export function IftaReturnStatusBadge({
  status,
  className,
}: {
  status: IftaReturnStatus;
  className?: string;
}) {
  const t = useT();

  const statusAttributes: Record<IftaReturnStatus, BadgeAttrProps> = {
    Draft: {
      variant: "secondary",
      text: t("Draft"),
      description: t("Worksheet can still change"),
      icon: <ClockIcon />,
    },
    Finalized: {
      variant: "info",
      text: t("Finalized"),
      description: t("Locked; reopen with a reason to change"),
      icon: <LockIcon />,
    },
    Filed: {
      variant: "active",
      text: t("Filed"),
      description: t("Submitted to the base jurisdiction"),
      icon: <CheckCheckIcon />,
    },
  };

  return (
    <Badge
      variant={statusAttributes[status].variant}
      className={cn("max-h-5", className)}
      title={t(statusAttributes[status].description)}
    >
      {statusAttributes[status].icon}
      {statusAttributes[status].text}
    </Badge>
  );
}

export function FuelPurchaseImportStatusBadge({
  status,
  className,
}: {
  status: FuelPurchaseImportStatus;
  className?: string;
}) {
  const t = useT();

  const statusAttributes: Record<FuelPurchaseImportStatus, BadgeAttrProps> = {
    Pending: {
      variant: "warning",
      text: t("Pending"),
      description: t("Statement received; waiting to be parsed."),
    },
    Parsed: {
      variant: "info",
      text: t("Parsed"),
      description: t("Rows are ready for review; nothing is imported until you commit."),
    },
    Committed: {
      variant: "active",
      text: t("Committed"),
      description: t("New rows became fuel purchases."),
    },
    Discarded: {
      variant: "outline",
      text: t("Discarded"),
      description: t("Thrown away without importing anything."),
    },
    Failed: {
      variant: "inactive",
      text: t("Failed"),
      description: t("The statement could not be parsed; fix the file and stage it again."),
    },
  };

  return (
    <Badge
      variant={statusAttributes[status].variant}
      className={cn("max-h-5", className)}
      title={t(statusAttributes[status].description)}
    >
      {statusAttributes[status].text}
    </Badge>
  );
}

export function FuelCardStatusBadge({
  status,
  className,
}: {
  status: FuelCardStatus;
  className?: string;
}) {
  const t = useT();

  const statusAttributes: Record<FuelCardStatus, BadgeAttrProps> = {
    Active: {
      variant: "active",
      text: t("Active"),
      description: t("Purchases on this card import and record normally."),
    },
    Suspended: {
      variant: "warning",
      text: t("Suspended"),
      description: t(
        "Temporarily on hold; imported purchases are flagged until it is reactivated.",
      ),
    },
    Cancelled: {
      variant: "inactive",
      text: t("Cancelled"),
      description: t("Closed with the provider; it cannot be reactivated."),
    },
  };

  return (
    <Badge
      variant={statusAttributes[status].variant}
      className={cn("max-h-5", className)}
      title={t(statusAttributes[status].description)}
    >
      {statusAttributes[status].text}
    </Badge>
  );
}
