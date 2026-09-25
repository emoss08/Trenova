import { ASSISTANT_REPLY_READY, parseReplyReady } from "@/components/assistant/reply-ready";
import {
  AI_AUDIT_CHAIN_MISMATCH_EVENT,
  AI_AUDIT_EXPORT_FAILED_EVENT,
  AI_AUDIT_EXPORT_READY_EVENT,
  AI_AUDIT_TRAIL_PATH,
  aiAuditExportNotice,
} from "@/lib/ai-audit-exports";
import { invoicePanelPath } from "@/lib/invoice-links";
import { AssistMark } from "@trenova/shared/components/ui/assist-mark";
import type { Notification } from "@trenova/shared/types/notification";
import {
  ArrowLeftRightIcon,
  CirclePauseIcon,
  CoinsIcon,
  ListChecksIcon,
  ShieldBanIcon,
  ShieldQuestionIcon,
  TruckIcon,
  AtSignIcon,
  BanIcon,
  BellIcon,
  CalendarClockIcon,
  CalendarOffIcon,
  CircleCheckIcon,
  CircleDollarSignIcon,
  DatabaseZapIcon,
  FileCheckIcon,
  FileDownIcon,
  FileLockIcon,
  FileWarningIcon,
  FileXIcon,
  FilterXIcon,
  IdCardIcon,
  LandmarkIcon,
  MailWarningIcon,
  OctagonAlertIcon,
  ReceiptTextIcon,
  Share2Icon,
  ShieldAlertIcon,
  ClockAlertIcon,
  TrendingDownIcon,
  TrendingUpIcon,
  TriangleAlertIcon,
  type LucideIcon,
} from "lucide-react";
import { workerRecordHref } from "@/lib/route-utils";

export function notificationDataString(
  notification: Pick<Notification, "data">,
  key: string,
): string | null {
  const value = notification.data?.[key];
  return typeof value === "string" && value.length > 0 ? value : null;
}

export function notificationDataNumber(
  notification: Pick<Notification, "data">,
  key: string,
): number | null {
  const value = notification.data?.[key];
  return typeof value === "number" && Number.isFinite(value) ? value : null;
}

export function notificationDataBoolean(
  notification: Pick<Notification, "data">,
  key: string,
): boolean {
  return notification.data?.[key] === true;
}

export function notificationRelatedId(
  notification: Pick<Notification, "relatedEntities">,
  key: string,
): string | null {
  const value = notification.relatedEntities?.[key];
  return typeof value === "string" && value.length > 0 ? value : null;
}

function entityPanelLink(
  basePath: string,
  entityId: string | null,
  extraParams?: Record<string, string>,
): string {
  if (!entityId) return basePath;

  const params = new URLSearchParams({
    panelType: "edit",
    panelEntityId: entityId,
    ...extraParams,
  });
  return `${basePath}?${params.toString()}`;
}

export interface NotificationAvatar {
  userId?: string;
  name?: string;
}

export interface NotificationDescriptor {
  category: string;
  icon: LucideIcon;
  iconClass: string;
  tileClass: string;
  hideMessage?: boolean;
  disableRowNavigation?: boolean;
  avatar?: (notification: Notification) => NotificationAvatar | null;
  getLink?: (notification: Notification) => string | null;
  /** The look for one notification, when its kind alone does not settle it. */
  refine?: (
    notification: Notification,
  ) => Partial<Pick<NotificationDescriptor, "icon" | "iconClass" | "tileClass">> | null;
}

const reportRunsLink = () => "/reports/runs";

const invoiceLink = (notification: Notification) => {
  const invoiceId = notificationRelatedId(notification, "invoiceId");
  return invoiceId ? invoicePanelPath(invoiceId) : "/billing/invoices";
};

const dispatchConsoleLink = (notification: Notification) =>
  notificationDataString(notification, "link") ?? "/dispatch/console";

const CARRIER_MONITORING_PATH = "/dispatch/carrier-monitoring";

const carrierIntelLink = (notification: Notification) => {
  const link = notificationDataString(notification, "link");
  if (link) return link;
  const carrierId = notificationRelatedId(notification, "carrierId");
  return carrierId
    ? entityPanelLink("/dispatch/carriers", carrierId, { tab: "intelligence" })
    : CARRIER_MONITORING_PATH;
};

const carrierIntelUsageLink = (notification: Notification) =>
  notificationDataString(notification, "link") ?? `${CARRIER_MONITORING_PATH}?tab=usage`;

const REPORT_READY: NotificationDescriptor = {
  category: "Reports",
  icon: FileDownIcon,
  iconClass: "text-success",
  tileClass: "bg-success-subtle",
  getLink: reportRunsLink,
};

function workerCredentialsLink(notification: Pick<Notification, "data">): string {
  const workerId = notificationDataString(notification, "workerId");
  return workerId ? workerRecordHref(workerId, "credentials") : "/hr/workers";
}

const aiControlLink = (notification: Notification) =>
  notificationDataString(notification, "link") ?? "/admin/agent-control";

/**
 * A reply the person walked away from, finished. The mark is the assistant's
 * own; a reply that failed takes the danger tone because it needs asking
 * again, while a completed or refused one is simply news.
 */
const ASSISTANT_REPLY_DESCRIPTOR: NotificationDescriptor = {
  category: "Assistant",
  icon: AssistMark,
  iconClass: "text-muted-foreground",
  tileClass: "bg-muted",
  getLink: (n) => parseReplyReady(n)?.link ?? null,
  refine: (n) =>
    parseReplyReady(n)?.status === "Failed"
      ? { iconClass: "text-destructive", tileClass: "bg-danger-subtle" }
      : null,
};

const EXACT_REGISTRY: Record<string, NotificationDescriptor> = {
  [ASSISTANT_REPLY_READY]: ASSISTANT_REPLY_DESCRIPTOR,
  "agent.proposals_pending": {
    category: "AI Control",
    icon: ListChecksIcon,
    iconClass: "text-brand",
    tileClass: "bg-brand-subtle",
    getLink: aiControlLink,
  },
  "agent.proposals_reminder": {
    category: "AI Control",
    icon: ClockAlertIcon,
    iconClass: "text-warning",
    tileClass: "bg-warning-subtle",
    getLink: aiControlLink,
  },
  "agent.tool_promoted": {
    category: "AI Control",
    icon: TrendingUpIcon,
    iconClass: "text-success",
    tileClass: "bg-success-subtle",
    getLink: aiControlLink,
  },
  "agent.tool_demoted": {
    category: "AI Control",
    icon: TrendingDownIcon,
    iconClass: "text-warning",
    tileClass: "bg-warning-subtle",
    getLink: aiControlLink,
  },
  "agent.quality_regression": {
    category: "AI Control",
    icon: TrendingDownIcon,
    iconClass: "text-warning",
    tileClass: "bg-warning-subtle",
    getLink: aiControlLink,
    refine: (n) =>
      notificationDataString(n, "severity") === "Critical"
        ? { iconClass: "text-destructive", tileClass: "bg-danger-subtle" }
        : null,
  },
  [AI_AUDIT_EXPORT_READY_EVENT]: {
    category: "AI Control",
    icon: FileDownIcon,
    iconClass: "text-success",
    tileClass: "bg-success-subtle",
    getLink: (n) => aiAuditExportNotice(n)?.link ?? null,
  },
  [AI_AUDIT_EXPORT_FAILED_EVENT]: {
    category: "AI Control",
    icon: FileXIcon,
    iconClass: "text-destructive",
    tileClass: "bg-danger-subtle",
    getLink: (n) => aiAuditExportNotice(n)?.link ?? null,
  },
  [AI_AUDIT_CHAIN_MISMATCH_EVENT]: {
    category: "AI Control",
    icon: FileLockIcon,
    iconClass: "text-destructive",
    tileClass: "bg-danger-subtle",
    getLink: () => AI_AUDIT_TRAIL_PATH,
  },
  "dash.pto_requested": {
    category: "Workers",
    icon: CalendarClockIcon,
    iconClass: "text-brand",
    tileClass: "bg-brand/10",
    getLink: () => "/hr/workers?pageTab=pto",
  },
  "dash.credential_expiring": {
    category: "Workers",
    icon: IdCardIcon,
    iconClass: "text-warning",
    tileClass: "bg-warning-subtle",
    getLink: workerCredentialsLink,
  },
  credential_expiring: {
    category: "Workers",
    icon: IdCardIcon,
    iconClass: "text-warning",
    tileClass: "bg-warning-subtle",
    getLink: workerCredentialsLink,
  },
  credential_expired: {
    category: "Workers",
    icon: ShieldAlertIcon,
    iconClass: "text-destructive",
    tileClass: "bg-danger-subtle",
    getLink: workerCredentialsLink,
  },
  credential_document_uploaded: {
    category: "Workers",
    icon: FileCheckIcon,
    iconClass: "text-brand",
    tileClass: "bg-brand/10",
    getLink: workerCredentialsLink,
  },
  report_run_completed: REPORT_READY,
  report_run_delivered: { ...REPORT_READY, iconClass: "text-brand", tileClass: "bg-brand/10" },
  report_run_failed: {
    category: "Reports",
    icon: FileXIcon,
    iconClass: "text-destructive",
    tileClass: "bg-danger-subtle",
    getLink: reportRunsLink,
  },
  report_run_canceled: {
    category: "Reports",
    icon: BanIcon,
    iconClass: "text-muted-foreground",
    tileClass: "bg-muted",
    getLink: reportRunsLink,
  },
  report_schedule_skipped: {
    category: "Reports",
    icon: CalendarOffIcon,
    iconClass: "text-warning",
    tileClass: "bg-warning-subtle",
    getLink: () => "/reports",
  },
  report_delivery_email_failed: {
    category: "Reports",
    icon: MailWarningIcon,
    iconClass: "text-warning",
    tileClass: "bg-warning-subtle",
    getLink: reportRunsLink,
  },
  invoice_reconciliation_warning: {
    category: "Billing",
    icon: ReceiptTextIcon,
    iconClass: "text-warning",
    tileClass: "bg-warning-subtle",
    getLink: invoiceLink,
  },
  invoice_shared: {
    category: "Billing",
    icon: Share2Icon,
    iconClass: "text-brand",
    tileClass: "bg-brand/10",
    avatar: (n) => {
      const userId = notificationDataString(n, "sharedById");
      const name = notificationDataString(n, "sharedByName");
      return userId || name ? { userId: userId ?? undefined, name: name ?? undefined } : null;
    },
    getLink: (n) => notificationDataString(n, "link") ?? invoiceLink(n),
  },
  billing_exception_recorded: {
    category: "Billing",
    icon: CircleDollarSignIcon,
    iconClass: "text-warning",
    tileClass: "bg-warning-subtle",
    getLink: (n) =>
      entityPanelLink("/shipment-management/shipments", notificationRelatedId(n, "shipmentId")),
  },
  bank_receipt_reconciliation_exception: {
    category: "Accounting",
    icon: LandmarkIcon,
    iconClass: "text-warning",
    tileClass: "bg-warning-subtle",
    getLink: (n) =>
      entityPanelLink(
        "/accounting/reconciliation/bank-receipts",
        notificationRelatedId(n, "bankReceiptId"),
      ),
  },
  shipment_comment_mention: {
    category: "Mentions",
    icon: AtSignIcon,
    iconClass: "text-brand",
    tileClass: "bg-brand/10",
    hideMessage: true,
    disableRowNavigation: true,
    avatar: (n) => {
      const userId = notificationDataString(n, "authorId");
      const name = notificationDataString(n, "authorName");
      return userId || name ? { userId: userId ?? undefined, name: name ?? undefined } : null;
    },
    getLink: (n) =>
      entityPanelLink("/shipment-management/shipments", notificationRelatedId(n, "shipmentId"), {
        tab: "comments",
      }),
  },
  "edi.message.dead_lettered": {
    category: "EDI",
    icon: ArrowLeftRightIcon,
    iconClass: "text-destructive",
    tileClass: "bg-danger-subtle",
    getLink: (n) => notificationDataString(n, "link") ?? "/edi/messages",
  },
  "edi.inbound_file.quarantined": {
    category: "EDI",
    icon: FileWarningIcon,
    iconClass: "text-destructive",
    tileClass: "bg-danger-subtle",
    getLink: (n) => notificationDataString(n, "link") ?? "/edi/inbound-files",
  },
  carrier_intel_block: {
    category: "Carrier Intelligence",
    icon: ShieldBanIcon,
    iconClass: "text-destructive",
    tileClass: "bg-danger-subtle",
    getLink: carrierIntelLink,
  },
  carrier_intel_change: {
    category: "Carrier Intelligence",
    icon: ShieldQuestionIcon,
    iconClass: "text-warning",
    tileClass: "bg-warning-subtle",
    getLink: carrierIntelLink,
  },
  carrier_intel_digest: {
    category: "Carrier Intelligence",
    icon: ListChecksIcon,
    iconClass: "text-brand",
    tileClass: "bg-brand/10",
    getLink: (n) => notificationDataString(n, "link") ?? CARRIER_MONITORING_PATH,
  },
  carrier_intel_provider_paused: {
    category: "Carrier Intelligence",
    icon: CirclePauseIcon,
    iconClass: "text-destructive",
    tileClass: "bg-danger-subtle",
    getLink: (n) =>
      notificationDataString(n, "link") ?? "/admin/integrations?category=CarrierCompliance",
  },
  carrier_equipment_mismatch: {
    category: "Carrier Intelligence",
    icon: TruckIcon,
    iconClass: "text-destructive",
    tileClass: "bg-danger-subtle",
    getLink: carrierIntelLink,
  },
  carrier_intel_spend_soft_cap: {
    category: "Carrier Intelligence",
    icon: CoinsIcon,
    iconClass: "text-warning",
    tileClass: "bg-warning-subtle",
    getLink: carrierIntelUsageLink,
  },
  carrier_intel_spend_cap: {
    category: "Carrier Intelligence",
    icon: CoinsIcon,
    iconClass: "text-destructive",
    tileClass: "bg-danger-subtle",
    getLink: carrierIntelUsageLink,
  },
  tender_accepted: {
    category: "Dispatch",
    icon: CircleCheckIcon,
    iconClass: "text-success",
    tileClass: "bg-success-subtle",
    getLink: dispatchConsoleLink,
  },
  tender_needs_review: {
    category: "Dispatch",
    icon: TriangleAlertIcon,
    iconClass: "text-warning",
    tileClass: "bg-warning-subtle",
    getLink: dispatchConsoleLink,
  },
  tender_waterfall_exhausted: {
    category: "Dispatch",
    icon: OctagonAlertIcon,
    iconClass: "text-destructive",
    tileClass: "bg-danger-subtle",
    getLink: dispatchConsoleLink,
  },
  rate_confirmation_issue_failed: {
    category: "Dispatch",
    icon: FileWarningIcon,
    iconClass: "text-destructive",
    tileClass: "bg-danger-subtle",
    getLink: dispatchConsoleLink,
  },
  tender_delivery_failed: {
    category: "Dispatch",
    icon: MailWarningIcon,
    iconClass: "text-destructive",
    tileClass: "bg-danger-subtle",
    getLink: dispatchConsoleLink,
  },
  tender_entries_skipped: {
    category: "Dispatch",
    icon: FilterXIcon,
    iconClass: "text-warning",
    tileClass: "bg-warning-subtle",
    getLink: dispatchConsoleLink,
  },
};

const TCA_DESCRIPTOR: NotificationDescriptor = {
  category: "Data alert",
  icon: DatabaseZapIcon,
  iconClass: "text-info",
  tileClass: "bg-info-subtle",
  getLink: (n) => notificationDataString(n, "link"),
};

const FALLBACK_DESCRIPTOR: NotificationDescriptor = {
  category: "System",
  icon: BellIcon,
  iconClass: "text-muted-foreground",
  tileClass: "bg-muted",
  getLink: (n) => notificationDataString(n, "link"),
};

export function getNotificationDescriptor(eventType: string): NotificationDescriptor {
  const exact = EXACT_REGISTRY[eventType];
  if (exact) return exact;
  if (eventType.startsWith("tca.")) return TCA_DESCRIPTOR;
  return FALLBACK_DESCRIPTOR;
}

/**
 * The descriptor for one notification: its kind's, refined by what it says.
 * A finished reply is recognised by the kind its data carries as well as by
 * its event type, since both name it.
 */
export function resolveNotificationDescriptor(notification: Notification): NotificationDescriptor {
  const base =
    parseReplyReady(notification) !== null
      ? ASSISTANT_REPLY_DESCRIPTOR
      : getNotificationDescriptor(notification.eventType);
  const refined = base.refine?.(notification);

  return refined ? { ...base, ...refined } : base;
}

export function getNotificationLink(notification: Notification): string | null {
  return resolveNotificationDescriptor(notification).getLink?.(notification) ?? null;
}
