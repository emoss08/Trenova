import { useT } from "@trenova/shared/i18n/use-t";
import { Badge } from "@trenova/shared/components/ui/badge";
import { cn } from "@trenova/shared/lib/utils";
import {
  phaseTone,
  type BadgeAttrProps,
  type BadgeClassAttrProps,
  type StatusPhase,
} from "@trenova/shared/lib/status-phase";
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
import type {
  InvoiceDisputeCaseStatus,
  InvoiceEdiSendStatus,
  InvoiceScope,
  InvoiceStatus,
  SettlementStatus,
} from "@trenova/shared/types/invoice";
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
import { CheckCheckIcon, CheckIcon, ClockIcon, LockIcon, XIcon } from "lucide-react";
import type React from "react";



type StatusBadgeProps = {
  status: string;
  className?: string;
};

export type { BadgeAttrProps, BadgeClassAttrProps, StatusPhase };

export type PlainBadgeAttrProps = {
  text: string;
  description?: string;
  className?: string;
};

const STATUS_PHASES: Record<string, StatusPhase> = {
  active: "active",
  inactive: "failed",
  draft: "draft",
  pending: "awaiting",
  completed: "complete",
  cancelled: "failed",
  processing: "active",
  inreview: "awaiting",
  // Compliance statuses
  compliant: "complete",
  noncompliant: "failed",
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
  const phase = STATUS_PHASES[normalizedStatus];

  return (
    <Badge
      variant={phase ? phaseTone(phase) : "neutral"}
      appearance={phase ? "subtle" : "outline"}
      className={cn("capitalize", className)}
    >
      {STATUS_ICONS[normalizedStatus]}
      {status}
    </Badge>
  );
}

export function BooleanBadge({ value }: { value: boolean }) {
  const t = useT();

  return (
    <Badge variant={value ? "success" : "danger"} className="max-h-5">
      {value ? t("Yes") : t("No")}
    </Badge>
  );
}

export function PTOStatusBadge({ status }: { status: PTOStatus }) {
  const t = useT();

  const ptoStatusAttrs: Record<PTOStatus, BadgeAttrProps> = {
    Requested: {
      phase: "awaiting",
      text: t("Requested"),
    },
    Approved: {
      phase: "complete",
      text: t("Approved"),
    },
    Cancelled: {
      phase: "failed",
      text: t("Cancelled"),
    },
    Rejected: {
      phase: "failed",
      text: t("Rejected"),
    },
  };

  return (
    <Badge variant={phaseTone(ptoStatusAttrs[status].phase)} className="max-h-5">
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
      phase: "draft",
      icon: <CheckIcon />,
    },
    restricted: {
      text: t("Restricted"),
      phase: "draft",
      icon: <LockIcon />,
    },
  };

  return (
    <Badge variant={phaseTone(valueAttrs[scope].phase)} className="max-h-5">
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
      phase: "draft",
      text: t("New"),
      description: t("Shipment has been created and is pending initial assignment."),
    },
    [shipmentStatusSchema.enum.PartiallyAssigned]: {
      phase: "active",
      text: t("Partially Assigned"),
      description: t(
        "Equipment or worker assignments are pending for one or more moves within this shipment.",
      ),
    },
    [shipmentStatusSchema.enum.PartiallyCompleted]: {
      phase: "active",
      text: t("Partially Completed"),
      description: t("Some moves within this shipment have been completed, but not all."),
    },
    [shipmentStatusSchema.enum.Assigned]: {
      phase: "awaiting",
      text: t("Assigned"),
      description: t(
        "All required equipment and workers have been assigned to this shipment's moves.",
      ),
    },
    [shipmentStatusSchema.enum.InTransit]: {
      phase: "active",
      text: t("In Transit"),
      description: t(
        "Active shipment with cargo currently in transport between designated locations.",
      ),
    },
    [shipmentStatusSchema.enum.Delayed]: {
      phase: "attention",
      text: t("Delayed"),
      description: t(
        "Shipment has exceeded scheduled arrival or delivery timeframes at one or more stops.",
      ),
    },
    [shipmentStatusSchema.enum.Completed]: {
      phase: "complete",
      text: t("Completed"),
      description: t(
        "All transportation activities for this shipment have been successfully completed.",
      ),
    },
    [shipmentStatusSchema.enum.Invoiced]: {
      phase: "complete",
      text: t("Invoiced"),
      description: t(
        "Invoice has been generated and posted for completed transportation services.",
      ),
    },
    [shipmentStatusSchema.enum.ReadyToInvoice]: {
      phase: "awaiting",
      text: t("Ready to Invoice"),
      description: t(
        "All moves within this shipment have been completed, and the shipment is ready to be invoiced.",
      ),
    },
    [shipmentStatusSchema.enum.Canceled]: {
      phase: "failed",
      text: t("Canceled"),
      description: t(
        "Shipment has been terminated and will not be completed as originally planned.",
      ),
    },
  };

  return (
    <Badge
      variant={phaseTone(statusAttributes[status].phase)}
      className={cn(className, "max-h-5")}
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
      phase: "active",
      text: t("Tendered"),
    },
    Accepted: {
      phase: "complete",
      text: t("Accepted"),
    },
    Rejected: {
      phase: "failed",
      text: t("Rejected"),
    },
    Expired: {
      phase: "failed",
      text: t("Expired"),
    },
    Canceled: {
      phase: "failed",
      text: t("Canceled"),
    },
  };

  return (
    <Badge
      variant={phaseTone(statusAttributes[status].phase)}
      className={cn(className, "max-h-5")}
    >
      {statusAttributes[status].text}
    </Badge>
  );
}

export const billingQueueStatusBadges: Record<BillingQueueStatus, BadgeAttrProps> = {
  ReadyForReview: {
    phase: "active",
    text: "Ready for Review",
  },
  InReview: {
    phase: "awaiting",
    text: "In Review",
  },
  Approved: {
    phase: "complete",
    text: "Approved",
  },
  Posted: {
    phase: "complete",
    text: "Posted",
  },
  OnHold: {
    phase: "awaiting",
    text: "On Hold",
  },
  SentBackToOps: {
    phase: "attention",
    text: "Sent Back to Ops",
  },
  Exception: {
    phase: "failed",
    text: "Exception",
  },
  Canceled: {
    phase: "failed",
    text: "Canceled",
  },
};

export function BillingQueueStatusBadge({
  status,
  className,
  title,
}: {
  status?: BillingQueueStatus | null;
  className?: string;
  title?: string;
}) {
  const t = useT();

  if (!status) return null;

  const { phase, text } = billingQueueStatusBadges[status];

  return (
    <Badge variant={phaseTone(phase)} title={title} className={cn(className, "max-h-5")}>
      {t(text)}
    </Badge>
  );
}

export function PlainBillingQueueStatusBadge({ status }: { status: BillingQueueStatus }) {
  const t = useT();

  const statusAttributes: Record<BillingQueueStatus, PlainBadgeAttrProps> = {
    ReadyForReview: {
      className: "bg-info-subtle text-info-foreground dark:bg-info-subtle dark:text-info-foreground",
      text: t("Ready for Review"),
    },
    InReview: {
      className: "bg-accent-indigo-subtle text-accent-indigo-on-subtle dark:bg-accent-indigo-subtle dark:text-accent-indigo-on-subtle",
      text: t("In Review"),
    },
    Approved: {
      className: "bg-success-subtle text-success-foreground dark:bg-success-subtle dark:text-success-foreground",
      text: t("Approved"),
    },
    Posted: {
      className: "bg-success-subtle text-success-foreground dark:bg-success-subtle dark:text-success-foreground",
      text: t("Posted"),
    },
    OnHold: {
      className: "bg-warning-subtle text-warning-foreground dark:bg-warning-subtle dark:text-warning-foreground",
      text: t("On Hold"),
    },
    SentBackToOps: {
      className: "bg-warning-subtle text-warning-foreground dark:bg-warning-subtle dark:text-warning-foreground",
      text: t("Sent Back to Ops"),
    },
    Exception: {
      className: "bg-danger-subtle text-danger-foreground dark:bg-danger-subtle dark:text-danger-foreground",
      text: t("Exception"),
    },
    Canceled: {
      className: "bg-sunken text-foreground-subtle dark:text-foreground-subtle",
      text: t("Canceled"),
    },
  };

  return (
    <span
      className={cn(
        "inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium",
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
      phase: "draft",
      text: t("Draft"),
    },
    Posted: {
      phase: "complete",
      text: t("Posted"),
    },
    Voided: {
      phase: "failed",
      text: t("Voided"),
    },
  };

  return (
    <Badge variant={phaseTone(statusAttributes[status].phase)} className={cn(className, "max-h-5")}>
      {statusAttributes[status].text}
    </Badge>
  );
}

export function PlainSettlementStatusBadge({ status }: { status: SettlementStatus }) {
  const t = useT();

  const statusAttributes: Record<SettlementStatus, PlainBadgeAttrProps> = {
    Paid: {
      className: "bg-success-subtle text-success-foreground dark:bg-success-subtle dark:text-success-foreground",
      text: t("Paid"),
    },
    PartiallyPaid: {
      className: "bg-warning-subtle text-warning-foreground dark:bg-warning-subtle dark:text-warning-foreground",
      text: t("Partial"),
    },
    Unpaid: {
      className: "bg-sunken text-foreground-subtle dark:text-foreground-subtle",
      text: t("Unpaid"),
    },
  };

  return (
    <span
      className={cn(
        "inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium",
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
      className: "bg-success-subtle text-success-foreground dark:bg-success-subtle dark:text-success-foreground",
      text: t("Posted"),
    },
    Reversed: {
      className: "bg-danger-subtle text-danger-foreground dark:bg-danger-subtle dark:text-danger-foreground",
      text: t("Reversed"),
    },
  };

  return (
    <span
      className={cn(
        "inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium",
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
      className: "bg-sunken text-foreground dark:text-foreground-subtle",
      text: t("Draft"),
    },
    Posted: {
      className: "bg-success-subtle text-success-foreground dark:bg-success-subtle dark:text-success-foreground",
      text: t("Posted"),
    },
    Voided: {
      className: "bg-danger-subtle text-danger-foreground line-through dark:bg-danger-subtle dark:text-danger-foreground",
      text: t("Voided"),
    },
  };

  return (
    <span
      className={cn(
        "inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium",
        statusAttributes[status].className,
      )}
    >
      {statusAttributes[status].text}
    </span>
  );
}

/**
 * The invoice's quick-read dispute flag: Disputed while a case is open, and
 * nothing at all otherwise, so a register row only draws the eye when it
 * should.
 */
export function PlainInvoiceDisputeBadge({
  disputeStatus,
}: {
  disputeStatus: "None" | "Disputed" | null | undefined;
}) {
  const t = useT();

  if (disputeStatus !== "Disputed") {
    return null;
  }

  return (
    <span className="inline-flex items-center rounded-full bg-warning-subtle px-2 py-0.5 text-xs font-medium text-warning-foreground dark:bg-warning-subtle dark:text-warning-foreground">
      {t("Disputed")}
    </span>
  );
}

export function InvoiceDisputeCaseStatusBadge({
  status,
  className,
}: {
  status: InvoiceDisputeCaseStatus;
  className?: string;
}) {
  const t = useT();

  const statusAttributes: Record<InvoiceDisputeCaseStatus, BadgeAttrProps> = {
    Open: { phase: "active", text: t("Open") },
    Resolved: { phase: "complete", text: t("Resolved") },
    Withdrawn: { phase: "draft", text: t("Withdrawn") },
  };

  return (
    <Badge variant={phaseTone(statusAttributes[status].phase)} className={cn("max-h-5", className)}>
      {statusAttributes[status].text}
    </Badge>
  );
}

/**
 * Where an invoice's outbound 210 stands. NotSent is the resting state of an
 * invoice whose customer takes EDI and has not been sent yet; NotConfigured
 * says the customer cannot take EDI at all, which is a setup problem rather
 * than a delivery one.
 */
export function InvoiceEdiSendStatusBadge({
  status,
  className,
}: {
  status: InvoiceEdiSendStatus;
  className?: string;
}) {
  const t = useT();

  const statusAttributes: Record<InvoiceEdiSendStatus, BadgeAttrProps> = {
    NotSent: { phase: "draft", text: t("Not sent") },
    NotConfigured: { phase: "draft", text: t("Not configured") },
    Queued: { phase: "active", text: t("Queued") },
    Generated: { phase: "active", text: t("Generated") },
    Sending: { phase: "active", text: t("Sending") },
    Sent: { phase: "complete", text: t("Sent") },
    Failed: { phase: "failed", text: t("Failed") },
    DeadLettered: { phase: "failed", text: t("Dead-lettered") },
  };

  return (
    <Badge variant={phaseTone(statusAttributes[status].phase)} className={cn("max-h-5", className)}>
      {statusAttributes[status].text}
    </Badge>
  );
}

/**
 * Marks an invoice that bills only part of what its shipments charged, because
 * another payer carries the rest. Reads as a warning to anyone reconciling it
 * against the shipment total.
 */
export function PlainInvoiceSplitBadge({
  isSplitBill,
}: {
  isSplitBill: boolean | null | undefined;
}) {
  const t = useT();

  if (!isSplitBill) {
    return null;
  }

  return (
    <span className="inline-flex items-center rounded-full bg-accent-indigo-subtle px-2 py-0.5 text-xs font-medium text-accent-indigo-on-subtle dark:bg-accent-indigo-subtle dark:text-accent-indigo-on-subtle">
      {t("Split bill")}
    </span>
  );
}

/**
 * Names what an invoice covers when it is anything other than one shipment. A
 * single-shipment invoice is the ordinary case and carries no badge, so the
 * badge is what makes a statement or an order invoice stand out in a list.
 */
export function PlainInvoiceScopeBadge({ scope }: { scope: InvoiceScope }) {
  const t = useT();

  const scopeAttributes: Record<Exclude<InvoiceScope, "Shipment">, PlainBadgeAttrProps> = {
    Order: {
      className: "bg-info-subtle text-info-foreground dark:bg-info-subtle dark:text-info-foreground",
      text: t("Order"),
    },
    Consolidated: {
      className: "bg-accent-violet-subtle text-accent-violet-on-subtle dark:bg-accent-violet-subtle dark:text-accent-violet-on-subtle",
      text: t("Consolidated"),
    },
    Adjustment: {
      className: "bg-warning-subtle text-warning-foreground dark:bg-warning-subtle dark:text-warning-foreground",
      text: t("Adjustment"),
    },
    Memo: {
      className: "bg-accent-teal-subtle text-accent-teal-on-subtle dark:bg-accent-teal-subtle dark:text-accent-teal-on-subtle",
      text: t("Memo"),
    },
  };

  if (scope === "Shipment") {
    return null;
  }

  return (
    <span
      className={cn(
        "inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium",
        scopeAttributes[scope].className,
      )}
    >
      {scopeAttributes[scope].text}
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
      phase: "draft",
      text: t("Draft"),
      description: t("Order has been created but not yet confirmed."),
    },
    Confirmed: {
      phase: "complete",
      text: t("Confirmed"),
      description: t("Order has been confirmed and is ready to be worked."),
    },
    InProgress: {
      phase: "active",
      text: t("In Progress"),
      description: t("Order is actively being fulfilled."),
    },
    Completed: {
      phase: "complete",
      text: t("Completed"),
      description: t("Order fulfillment has been completed."),
    },
    Billed: {
      phase: "complete",
      text: t("Billed"),
      description: t("Order has been billed to the customer."),
    },
    Closed: {
      phase: "draft",
      text: t("Closed"),
      description: t("Order has been closed and finalized."),
    },
    Canceled: {
      phase: "failed",
      text: t("Canceled"),
      description: t("Order has been canceled and will not be fulfilled."),
    },
  };

  return (
    <Badge variant={phaseTone(statusAttributes[status].phase)} className={cn(className, "max-h-5")}>
      {statusAttributes[status].text}
    </Badge>
  );
}

export function EDITransferStatusBadge({ status }: { status?: EDITransferStatus | string }) {
  const t = useT();

  if (!status) return null;

  const attrs: Record<EDITransferStatus, BadgeAttrProps> = {
    Submitted: {
      phase: "awaiting",
      text: t("Submitted"),
      description: t("Tender has been submitted and is awaiting review by the receiving side."),
    },
    MappingRequired: {
      phase: "awaiting",
      text: t("Mapping Required"),
      description: t("Tender references entities that are not mapped for this partner yet."),
    },
    PendingApproval: {
      phase: "active",
      text: t("Pending Approval"),
      description: t("Tender is ready for the receiving organization to approve or reject."),
    },
    Processing: {
      phase: "draft",
      text: t("Processing"),
      description: t("Approval is running and the target shipment is being created."),
    },
    Approved: {
      phase: "complete",
      text: t("Approved"),
      description: t("Tender was accepted and the target shipment has been created."),
    },
    Rejected: {
      phase: "failed",
      text: t("Rejected"),
      description: t("Tender was rejected by the receiving side."),
    },
    Expired: {
      phase: "draft",
      text: t("Expired"),
      description: t("Tender expired before it was actioned."),
    },
    Canceled: {
      phase: "draft",
      text: t("Canceled"),
      description: t("Tender was canceled or superseded."),
    },
    Failed: {
      phase: "failed",
      text: t("Failed"),
      description: t("Tender processing failed. Review the failure reason for details."),
    },
  };
  const attr = attrs[status as EDITransferStatus];
  if (!attr) {
    return <Badge variant="neutral" appearance="outline">{status}</Badge>;
  }
  return (
    <Badge variant={phaseTone(attr.phase)} className="max-h-5" title={t(attr.description)}>
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
        variant="success"
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
      variant={passed ? "success" : "danger"}
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
      phase: "queued",
      text: t("Queued"),
      description: t("Message is queued for delivery to the trading partner."),
    },
    Sending: {
      phase: "active",
      text: t("Sending"),
      description: t("Delivery to the trading partner is in progress."),
    },
    Sent: {
      phase: "complete",
      text: t("Sent"),
      description: t("Message was delivered to the trading partner."),
    },
    Failed: {
      phase: "awaiting",
      text: t("Failed"),
      description: t("The last delivery attempt failed. Retries are scheduled automatically."),
    },
    DeadLettered: {
      phase: "failed",
      text: t("Dead Lettered"),
      description: t("Delivery retries were exhausted. Retry manually after fixing the cause."),
    },
  };
  const attr = attrs[status as EDIMessageDeliveryStatus];
  if (!attr) {
    return <Badge variant="neutral" appearance="outline">{status}</Badge>;
  }
  return (
    <Badge variant={phaseTone(attr.phase)} className="max-h-5" title={t(attr.description)}>
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
      phase: "draft",
      text: t("Not Expected"),
      description: t("No acknowledgment is expected for this message."),
    },
    Pending: {
      phase: "awaiting",
      text: t("Ack Pending"),
      description: t("Waiting for the trading partner to acknowledge this message."),
    },
    Accepted: {
      phase: "complete",
      text: t("Accepted"),
      description: t("The trading partner acknowledged and accepted this message."),
    },
    Rejected: {
      phase: "failed",
      text: t("Rejected"),
      description: t(
        "The trading partner rejected this message. Review the acknowledgment errors.",
      ),
    },
    Failed: {
      phase: "failed",
      text: t("Ack Failed"),
      description: t("Acknowledgment processing failed."),
    },
  };
  const attr = attrs[status as EDIMessageAcknowledgmentStatus];
  if (!attr) {
    return <Badge variant="neutral" appearance="outline">{status}</Badge>;
  }
  return (
    <Badge variant={phaseTone(attr.phase)} className="max-h-5" title={t(attr.description)}>
      {attr.text}
    </Badge>
  );
}

export function EDIInboundFileStatusBadge({ status }: { status?: EDIInboundFileStatus | string }) {
  const t = useT();

  if (!status) return null;

  const attrs: Record<EDIInboundFileStatus, BadgeAttrProps> = {
    Received: {
      phase: "active",
      text: t("Received"),
      description: t("File was pulled from the partner mailbox and is awaiting processing."),
    },
    Parsed: {
      phase: "active",
      text: t("Parsed"),
      description: t("File envelope was parsed and transactions are being processed."),
    },
    Processed: {
      phase: "complete",
      text: t("Processed"),
      description: t("Every transaction in this file was processed successfully."),
    },
    PartiallyProcessed: {
      phase: "awaiting",
      text: t("Partial"),
      description: t("Some transactions processed with warnings. Review the failure reason."),
    },
    Quarantined: {
      phase: "failed",
      text: t("Quarantined"),
      description: t("The file could not be processed. Fix the cause and reprocess."),
    },
    Duplicate: {
      phase: "draft",
      text: t("Duplicate"),
      description: t("This interchange was already processed and was skipped."),
    },
  };
  const attr = attrs[status as EDIInboundFileStatus];
  if (!attr) {
    return <Badge variant="neutral" appearance="outline">{status}</Badge>;
  }
  return (
    <Badge variant={phaseTone(attr.phase)} className="max-h-5" title={t(attr.description)}>
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
      phase: "draft",
      text: t("Draft"),
      description: t(
        "Settlement is being assembled — pay, earnings, and deductions can still change.",
      ),
    },
    PendingApproval: {
      phase: "awaiting",
      text: t("Pending Approval"),
      description: t("Submitted for review and waiting on an approver."),
    },
    Approved: {
      phase: "active",
      text: t("Approved"),
      description: t("Approved and locked; deduction side effects have been applied."),
    },
    Posted: {
      phase: "complete",
      text: t("Posted"),
      description: t("Journalized to the general ledger and awaiting payment."),
    },
    Paid: {
      phase: "complete",
      text: t("Paid"),
      description: t("Paid out to the driver; the settlement is final."),
    },
    Voided: {
      phase: "failed",
      text: t("Voided"),
      description: t("Reversed — pay events returned to the pool and side effects were undone."),
    },
  };

  return (
    <Badge
      variant={phaseTone(statusAttributes[status].phase)}
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
      phase: "active",
      text: t("Open"),
      description: t("Batch is accepting settlements; generation tops it up as pay accrues."),
    },
    Completed: {
      phase: "complete",
      text: t("Completed"),
      description: t("Batch is closed; late accruals settle individually or in the next period."),
    },
    Canceled: {
      phase: "failed",
      text: t("Canceled"),
      description: t("Batch was canceled and no longer collects settlements."),
    },
  };

  return (
    <Badge
      variant={phaseTone(statusAttributes[status].phase)}
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
      phase: "awaiting",
      text: t("Outstanding"),
      description: t("Nothing recovered yet — the full amount comes out of upcoming settlements."),
    },
    PartiallyRecovered: {
      phase: "active",
      text: t("Partially Recovered"),
      description: t(
        "Some of the advance has been recovered; the rest is withheld from future settlements.",
      ),
    },
    Recovered: {
      phase: "complete",
      text: t("Recovered"),
      description: t("Fully recovered from the driver's settlements."),
    },
    WrittenOff: {
      phase: "failed",
      text: t("Written Off"),
      description: t("Remaining balance was written off and will not be recovered."),
    },
  };

  return (
    <Badge
      variant={phaseTone(statusAttributes[status].phase)}
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
      phase: "complete",
      text: t("Active"),
      description: t("Added automatically to each qualifying settlement."),
    },
    Paused: {
      phase: "awaiting",
      text: t("Paused"),
      description: t("Temporarily skipped by settlements; resume to start paying again."),
    },
    Completed: {
      phase: "draft",
      text: t("Completed"),
      description: t("Reached its lifetime cap and stopped permanently."),
    },
  };

  return (
    <Badge
      variant={phaseTone(statusAttributes[status].phase)}
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
      phase: "complete",
      text: t("Active"),
      description: t("Withheld automatically from each qualifying settlement."),
    },
    Paused: {
      phase: "awaiting",
      text: t("Paused"),
      description: t("Temporarily skipped by settlements; resume to start withholding again."),
    },
    Completed: {
      phase: "draft",
      text: t("Completed"),
      description: t("Reached its lifetime cap and stopped permanently."),
    },
  };

  return (
    <Badge
      variant={phaseTone(statusAttributes[status].phase)}
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
      phase: "complete",
      text: t("Active"),
      description: t("Accepting contributions and accruing interest per 49 CFR 376.12(k)."),
    },
    Closed: {
      phase: "draft",
      text: t("Closed"),
      description: t("Closed out — the balance was refunded or applied."),
    },
  };

  return (
    <Badge
      variant={phaseTone(statusAttributes[status].phase)}
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
      phase: "active",
      text: t("Accrued"),
      description: t("Earned but not yet on a settlement — waiting in the unsettled pool."),
    },
    Settled: {
      phase: "complete",
      text: t("Settled"),
      description: t("Attached to a settlement as earning lines."),
    },
    Voided: {
      phase: "failed",
      text: t("Voided"),
      description: t("Canceled (move canceled or reverted) and excluded from pay."),
    },
  };

  return (
    <Badge
      variant={phaseTone(statusAttributes[status].phase)}
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

  const attributes: Record<PayeeClassification, BadgeClassAttrProps> = {
    CompanyDriver: {
      accent: "accent-sky",
      text: t("Company Driver"),
      description: t("W-2 employee — settlements post to the driver pay expense account."),
    },
    OwnerOperator: {
      accent: "accent-violet",
      text: t("Owner-Operator"),
      description: t("1099 contractor — settlements post to the purchased transportation account."),
    },
  };

  return (
    <Badge
      variant={attributes[classification].accent}
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
      phase: "draft",
      text: t("Draft"),
      description: t(
        "Statement is being assembled — cost events and adjustments can still change.",
      ),
    },
    PendingApproval: {
      phase: "awaiting",
      text: t("Pending Approval"),
      description: t("Submitted for review and waiting on an approver."),
    },
    Approved: {
      phase: "active",
      text: t("Approved"),
      description: t("Approved and locked, ready to post to the general ledger."),
    },
    Posted: {
      phase: "complete",
      text: t("Posted"),
      description: t(
        "Journalized as purchased transportation against accounts payable; awaiting payment.",
      ),
    },
    Paid: {
      phase: "complete",
      text: t("Paid"),
      description: t(
        "Disbursed to the carrier — the cash journal and ledger payment are recorded.",
      ),
    },
    Voided: {
      phase: "failed",
      text: t("Voided"),
      description: t("Reversed — cost events returned to the pool and GL postings were reversed."),
    },
  };

  return (
    <Badge
      variant={phaseTone(statusAttributes[status].phase)}
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
      phase: "active",
      text: t("Open"),
      description: t("AP run is accepting settlements; generation tops it up as cost accrues."),
    },
    Completed: {
      phase: "complete",
      text: t("Completed"),
      description: t("AP run is closed; late accruals settle in the next period."),
    },
    Canceled: {
      phase: "failed",
      text: t("Canceled"),
      description: t("Batch was canceled and no longer collects settlements."),
    },
  };

  return (
    <Badge
      variant={phaseTone(statusAttributes[status].phase)}
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
      phase: "active",
      text: t("Pending"),
      description: t("Accrued purchased-transportation cost not yet on a settlement."),
    },
    Attached: {
      phase: "awaiting",
      text: t("Attached"),
      description: t("On a draft settlement — locked until the settlement is processed or voided."),
    },
    Settled: {
      phase: "complete",
      text: t("Settled"),
      description: t("Included on a posted carrier settlement."),
    },
    Voided: {
      phase: "failed",
      text: t("Voided"),
      description: t("Canceled (assignment or shipment canceled) and excluded from settlement."),
    },
  };

  return (
    <Badge
      variant={phaseTone(statusAttributes[status].phase)}
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
      phase: "draft",
      text: t("Suggested"),
      description: t("System-proposed pairing of a carrier invoice with an assignment."),
    },
    Matched: {
      phase: "active",
      text: t("Matched"),
      description: t("Invoice total agrees with the negotiated buy rate within tolerance."),
    },
    Variance: {
      phase: "awaiting",
      text: t("Variance"),
      description: t("Invoice total differs from the buy rate beyond the configured tolerance."),
    },
    Resolved: {
      phase: "complete",
      text: t("Resolved"),
      description: t("Accepted — the invoice is reconciled against the assignment."),
    },
    Rejected: {
      phase: "failed",
      text: t("Rejected"),
      description: t("Dismissed — the invoice does not bill this assignment."),
    },
  };

  return (
    <Badge
      variant={phaseTone(statusAttributes[status].phase)}
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
      phase: "awaiting",
      text: t("Pending"),
      description: t("Compliance review has not been completed for this carrier."),
    },
    Qualified: {
      phase: "complete",
      text: t("Qualified"),
      description: t("The carrier passed the compliance review and can be assigned freight."),
    },
    Disqualified: {
      phase: "failed",
      text: t("Disqualified"),
      description: t("The carrier failed compliance and must not be assigned freight."),
    },
    Expired: {
      phase: "failed",
      text: t("Expired"),
      description: t("The carrier's qualification lapsed and must be renewed before assignment."),
    },
  };

  return (
    <Badge
      variant={phaseTone(statusAttributes[status].phase)}
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
      phase: "complete",
      text: t("Satisfactory"),
      description: t("FMCSA rated the carrier satisfactory."),
    },
    Conditional: {
      phase: "awaiting",
      text: t("Conditional"),
      description: t("FMCSA found deficiencies — review before assigning freight."),
    },
    Unsatisfactory: {
      phase: "failed",
      text: t("Unsatisfactory"),
      description: t("FMCSA rated the carrier unsatisfactory — do not assign freight."),
    },
    NotRated: {
      phase: "draft",
      text: t("Not Rated"),
      description: t("FMCSA has not issued a safety rating for this carrier."),
    },
  };

  return (
    <Badge
      variant={phaseTone(statusAttributes[status].phase)}
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
      phase: "awaiting",
      text: t("Pending"),
      description: t("The carrier has been assigned but has not confirmed the rate yet."),
    },
    Confirmed: {
      phase: "complete",
      text: t("Confirmed"),
      description: t("The carrier confirmed the negotiated rate for this move."),
    },
    Canceled: {
      phase: "failed",
      text: t("Canceled"),
      description: t("The carrier assignment was canceled or replaced."),
    },
  };

  return (
    <Badge
      variant={phaseTone(statusAttributes[status].phase)}
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
      phase: "active",
      text: t("Active"),
      description: t("Offers are out to carriers and a response is pending."),
    },
    Accepted: {
      phase: "complete",
      text: t("Accepted"),
      description: t("A carrier accepted the tender and the move is covered."),
    },
    Exhausted: {
      phase: "failed",
      text: t("Exhausted"),
      description: t("Every carrier declined or timed out — the move is still uncovered."),
    },
    Canceled: {
      phase: "failed",
      text: t("Canceled"),
      description: t("The tender was canceled by a dispatcher."),
    },
    NeedsReview: {
      phase: "awaiting",
      text: t("Needs Review"),
      description: t(
        "A carrier accepted but auto-assignment failed — assign the move manually or cancel.",
      ),
    },
  };

  return (
    <Badge
      variant={phaseTone(statusAttributes[status].phase)}
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
      phase: "draft",
      text: t("Pending"),
      description: t("Queued behind a higher-ranked carrier; nothing has been sent yet."),
    },
    Sent: {
      phase: "active",
      text: t("Sent"),
      description: t("Delivered to the carrier and awaiting their response."),
    },
    Accepted: {
      phase: "complete",
      text: t("Accepted"),
      description: t("The carrier accepted this offer."),
    },
    Declined: {
      phase: "failed",
      text: t("Declined"),
      description: t("The carrier declined this offer."),
    },
    Expired: {
      phase: "failed",
      text: t("Expired"),
      description: t("The offer window elapsed without a response."),
    },
    Withdrawn: {
      phase: "draft",
      text: t("Withdrawn"),
      description: t("The offer was withdrawn when the tender was canceled."),
    },
    Superseded: {
      phase: "draft",
      text: t("Superseded"),
      description: t("Another carrier accepted first; this offer no longer stands."),
    },
    Skipped: {
      phase: "draft",
      text: t("Skipped"),
      description: t("The waterfall skipped this carrier."),
    },
    DeliveryFailed: {
      phase: "failed",
      text: t("Delivery Failed"),
      description: t("The offer could not be delivered on its channel."),
    },
  };

  return (
    <Badge
      variant={phaseTone(statusAttributes[status].phase)}
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
      phase: "draft",
      text: t("Generated"),
      description: t("Rate confirmation PDF is filed but has not been sent to the carrier."),
    },
    Sent: {
      phase: "active",
      text: t("Sent"),
      description: t("Emailed to the carrier's rate confirmation contacts."),
    },
    Confirmed: {
      phase: "complete",
      text: t("Confirmed"),
      description: t("The carrier confirmed the negotiated rate."),
    },
    Voided: {
      phase: "failed",
      text: t("Voided"),
      description: t("Superseded by a newer revision or voided manually."),
    },
  };

  return (
    <Badge
      variant={phaseTone(statusAttributes[status].phase)}
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
      phase: "draft",
      text: t("Draft"),
      description: t("Worksheet can still change"),
      icon: <ClockIcon />,
    },
    Finalized: {
      phase: "active",
      text: t("Finalized"),
      description: t("Locked; reopen with a reason to change"),
      icon: <LockIcon />,
    },
    Filed: {
      phase: "complete",
      text: t("Filed"),
      description: t("Submitted to the base jurisdiction"),
      icon: <CheckCheckIcon />,
    },
  };

  return (
    <Badge
      variant={phaseTone(statusAttributes[status].phase)}
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
      phase: "awaiting",
      text: t("Pending"),
      description: t("Statement received; waiting to be parsed."),
    },
    Parsed: {
      phase: "active",
      text: t("Parsed"),
      description: t("Rows are ready for review; nothing is imported until you commit."),
    },
    Committed: {
      phase: "complete",
      text: t("Committed"),
      description: t("New rows became fuel purchases."),
    },
    Discarded: {
      phase: "draft",
      text: t("Discarded"),
      description: t("Thrown away without importing anything."),
    },
    Failed: {
      phase: "failed",
      text: t("Failed"),
      description: t("The statement could not be parsed; fix the file and stage it again."),
    },
  };

  return (
    <Badge
      variant={phaseTone(statusAttributes[status].phase)}
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
      phase: "complete",
      text: t("Active"),
      description: t("Purchases on this card import and record normally."),
    },
    Suspended: {
      phase: "awaiting",
      text: t("Suspended"),
      description: t(
        "Temporarily on hold; imported purchases are flagged until it is reactivated.",
      ),
    },
    Cancelled: {
      phase: "failed",
      text: t("Cancelled"),
      description: t("Closed with the provider; it cannot be reactivated."),
    },
  };

  return (
    <Badge
      variant={phaseTone(statusAttributes[status].phase)}
      className={cn("max-h-5", className)}
      title={t(statusAttributes[status].description)}
    >
      {statusAttributes[status].text}
    </Badge>
  );
}
