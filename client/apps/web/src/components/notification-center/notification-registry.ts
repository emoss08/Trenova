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
import { defineLabels } from "@trenova/shared/i18n/labels";
import type { Notification } from "@trenova/shared/types/notification";
import {
  AlertOctagonIcon,
  AlertTriangleIcon,
  AtSignIcon,
  Bell01Icon,
  Building08Icon,
  CalendarClockIcon,
  CalendarMinus01Icon,
  CheckCircleIcon,
  ClockAlertIcon,
  Coins01Icon,
  CurrencyDollarCircleIcon,
  Database02Icon,
  FileAlertIcon,
  FileCheck02Icon,
  FileDownload02Icon,
  FileLock02Icon,
  FileX02Icon,
  FilterFunnelXIcon,
  type IconComponent,
  IdCardIcon,
  InvoiceIcon,
  ListChecksIcon,
  MailAlertIcon,
  PauseCircleIcon,
  Share07Icon,
  ShieldAlertIcon,
  ShieldBanIcon,
  ShieldQuestionIcon,
  SlashCircle01Icon,
  SwitchHorizontal01Icon,
  TrendDown01Icon,
  TrendUp01Icon,
  Truck01Icon,
} from "@trenova/shared/components/icons";
import { workerRecordHref } from "@/lib/route-utils";
import { recordPath } from "@/config/record-links";

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

/**
 * Something a row offers beyond opening it. `open` goes somewhere; the others change the
 * record the notification is about, and the row moves to Done once they succeed. An `open`
 * that `settles` is the whole of what the notification asked for (reading a reply), so the
 * row goes to Done with it; any other `open` only marks the row read, because the work it
 * leads to is what finishes it.
 */
export type NotificationRowAction =
  | { kind: "open"; label: string; href: string; primary?: boolean; settles?: boolean }
  | { kind: "acknowledge-exception"; label: string; itemId: string }
  | { kind: "approve-time-off"; label: string; timeOffId: string };

const WHY = defineLabels({
  assigned: "Assigned to you",
  woke: "Back from being set aside",
  commented: "New comment on your exception",
  escalated: "Yours, and now overdue",
  mentioned: "You were mentioned",
  approval: "Waits on your approval",
  deciders: "Anyone who can approve",
  asked: "You asked for it",
  shared: "Shared with you",
  ran: "You ran it",
});

const ACTION = defineLabels({
  acknowledge: "Acknowledge",
  openException: "Open exception",
  openQueue: "Open the queue",
  review: "Review changes",
  approve: "Approve",
  openRequest: "Open request",
  readReply: "Read the reply",
});

const EXCEPTIONS_PATH = "/desk/needs";

const exceptionLink = (notification: Notification) =>
  notificationDataString(notification, "link") ?? EXCEPTIONS_PATH;

/** A row about one exception that is the person's own: acknowledge it, or open it. */
function exceptionActions(notification: Notification): NotificationRowAction[] {
  const itemId = notificationDataString(notification, "itemId");
  const open: NotificationRowAction = {
    kind: "open",
    label: ACTION.openException,
    href: exceptionLink(notification),
  };

  return itemId
    ? [{ kind: "acknowledge-exception", label: ACTION.acknowledge, itemId }, open]
    : [{ ...open, primary: true }];
}

/**
 * An exception on the Desk. It is drawn as a warning so the catalog does not call the kind
 * urgent, and takes the danger tone only for a notice whose own severity is critical.
 */
const EXCEPTION_DESCRIPTOR: NotificationDescriptor = {
  category: "Exceptions",
  icon: ClockAlertIcon,
  iconClass: "text-warning",
  tileClass: "bg-warning-subtle",
  getLink: exceptionLink,
  refine: (n) =>
    notificationDataString(n, "severity") === "Critical"
      ? { iconClass: "text-destructive", tileClass: "bg-danger-subtle" }
      : null,
};

/** A state drawn as a badge on the row. `label` is English source; translate it with `args`. */
export interface NotificationStatus {
  tone: "danger" | "warning" | "neutral" | "success";
  label: string;
  args?: unknown[];
}

const STATUS = defineLabels({
  waitingDays: "Waiting {0, plural, one {# day} other {# days}}",
});

const HOURS_PER_DAY = 24;

/** Proposals nobody has decided for a day or more say how long they have waited. */
function proposalsStatus(notification: Notification): NotificationStatus | null {
  const hours = notificationDataNumber(notification, "waitingHours");
  if (hours === null || hours < HOURS_PER_DAY) return null;

  return { tone: "warning", label: STATUS.waitingDays, args: [Math.floor(hours / HOURS_PER_DAY)] };
}

export interface NotificationAvatar {
  userId?: string;
  name?: string;
}

export interface NotificationDescriptor {
  /** What happened, as a person turning it off reads it: "A carrier accepts a tender". */
  label?: string;
  category: string;
  icon: IconComponent;
  iconClass: string;
  tileClass: string;
  hideMessage?: boolean;
  avatar?: (notification: Notification) => NotificationAvatar | null;
  getLink?: (notification: Notification) => string | null;
  /**
   * Why the person got it, in the language on screen: "Assigned to you", "You were
   * mentioned". A kind with none shows its category for a broadcast and nothing for a
   * notification that is plainly the person's own.
   */
  why?: (notification: Notification) => string | null;
  /** What can be done about it from the row itself, the one that settles it first. */
  actions?: (notification: Notification) => NotificationRowAction[];
  /** A state the row should carry as a badge, beyond its priority: how long it has waited. */
  status?: (notification: Notification) => NotificationStatus | null;
  /** The look for one notification, when its kind alone does not settle it. */
  refine?: (
    notification: Notification,
  ) => Partial<Pick<NotificationDescriptor, "icon" | "iconClass" | "tileClass">> | null;
}

const reportRunsLink = () => "/reports/runs";

/** One report opens in the builder; several open the library, which badges each. */
const needsAttentionLink = (notification: Notification) => {
  const ids = notification.data?.definitionIds;
  if (Array.isArray(ids) && ids.length === 1 && typeof ids[0] === "string" && ids[0]) {
    return recordPath("report", ids[0]);
  }
  return "/reports";
};

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
  why: () => WHY.ran,
  category: "Reports",
  icon: FileDownload02Icon,
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
  label: "A reply you left is ready",
  category: "Assistant",
  icon: AssistMark,
  iconClass: "text-muted-foreground",
  tileClass: "bg-muted",
  getLink: (n) => parseReplyReady(n)?.link ?? null,
  why: () => WHY.asked,
  actions: (n) => {
    const href = parseReplyReady(n)?.link;
    return href
      ? [{ kind: "open", label: ACTION.readReply, href, primary: true, settles: true }]
      : [];
  },
  refine: (n) =>
    parseReplyReady(n)?.status === "Failed"
      ? { iconClass: "text-destructive", tileClass: "bg-danger-subtle" }
      : null,
};

const EXACT_REGISTRY: Record<string, NotificationDescriptor> = {
  [ASSISTANT_REPLY_READY]: ASSISTANT_REPLY_DESCRIPTOR,
  "agent.proposals_pending": {
    label: "Agent proposals wait for you",
    category: "AI Control",
    icon: ListChecksIcon,
    iconClass: "text-brand",
    tileClass: "bg-brand-subtle",
    getLink: aiControlLink,
    why: () => WHY.deciders,
    status: proposalsStatus,
    actions: (n) => [{ kind: "open", label: ACTION.review, href: aiControlLink(n), primary: true }],
  },
  "agent.proposals_reminder": {
    label: "A reminder about waiting proposals",
    category: "AI Control",
    icon: ClockAlertIcon,
    iconClass: "text-warning",
    tileClass: "bg-warning-subtle",
    getLink: aiControlLink,
    why: () => WHY.deciders,
    status: proposalsStatus,
    actions: (n) => [{ kind: "open", label: ACTION.review, href: aiControlLink(n), primary: true }],
  },
  "watchtower.critical": {
    ...EXCEPTION_DESCRIPTOR,
    label: "A critical exception reaches your team",
    actions: (n) => [
      { kind: "open", label: ACTION.openQueue, href: exceptionLink(n), primary: true },
    ],
  },
  "watchtower.assigned": {
    ...EXCEPTION_DESCRIPTOR,
    label: "An exception is assigned to you",
    why: (n) => (notificationDataString(n, "notice") === "woke" ? WHY.woke : WHY.assigned),
    actions: exceptionActions,
  },
  "watchtower.commented": {
    ...EXCEPTION_DESCRIPTOR,
    label: "Someone comments on your exception",
    why: () => WHY.commented,
    actions: exceptionActions,
  },
  "watchtower.escalated": {
    ...EXCEPTION_DESCRIPTOR,
    label: "Your exception goes overdue",
    why: () => WHY.escalated,
    actions: exceptionActions,
  },
  "agent.tool_promoted": {
    label: "A tool earns more trust",
    category: "AI Control",
    icon: TrendUp01Icon,
    iconClass: "text-success",
    tileClass: "bg-success-subtle",
    getLink: aiControlLink,
  },
  "agent.tool_demoted": {
    label: "A tool loses trust",
    category: "AI Control",
    icon: TrendDown01Icon,
    iconClass: "text-warning",
    tileClass: "bg-warning-subtle",
    getLink: aiControlLink,
  },
  "agent.quality_regression": {
    label: "An agent’s quality drops",
    category: "AI Control",
    icon: TrendDown01Icon,
    iconClass: "text-warning",
    tileClass: "bg-warning-subtle",
    getLink: aiControlLink,
    refine: (n) =>
      notificationDataString(n, "severity") === "Critical"
        ? { iconClass: "text-destructive", tileClass: "bg-danger-subtle" }
        : null,
  },
  [AI_AUDIT_EXPORT_READY_EVENT]: {
    label: "An audit export is ready",
    category: "AI Control",
    icon: FileDownload02Icon,
    iconClass: "text-success",
    tileClass: "bg-success-subtle",
    getLink: (n) => aiAuditExportNotice(n)?.link ?? null,
  },
  [AI_AUDIT_EXPORT_FAILED_EVENT]: {
    label: "An audit export fails",
    category: "AI Control",
    icon: FileX02Icon,
    iconClass: "text-destructive",
    tileClass: "bg-danger-subtle",
    getLink: (n) => aiAuditExportNotice(n)?.link ?? null,
  },
  [AI_AUDIT_CHAIN_MISMATCH_EVENT]: {
    label: "The audit chain doesn’t verify",
    category: "AI Control",
    icon: FileLock02Icon,
    iconClass: "text-destructive",
    tileClass: "bg-danger-subtle",
    getLink: () => AI_AUDIT_TRAIL_PATH,
  },
  "dash.pto_requested": {
    label: "A worker requests time off",
    category: "Workers",
    icon: CalendarClockIcon,
    iconClass: "text-brand",
    tileClass: "bg-brand/10",
    getLink: () => "/hr/workers?pageTab=pto",
    why: () => WHY.approval,
    actions: (n) => {
      const timeOffId = notificationDataString(n, "ptoId");
      const open: NotificationRowAction = {
        kind: "open",
        label: ACTION.openRequest,
        href: "/hr/workers?pageTab=pto",
      };

      return timeOffId
        ? [{ kind: "approve-time-off", label: ACTION.approve, timeOffId }, open]
        : [{ ...open, primary: true }];
    },
  },
  "dash.credential_expiring": {
    label: "A worker’s credential is about to expire",
    category: "Workers",
    icon: IdCardIcon,
    iconClass: "text-warning",
    tileClass: "bg-warning-subtle",
    getLink: workerCredentialsLink,
  },
  credential_expiring: {
    label: "A credential is about to expire",
    category: "Workers",
    icon: IdCardIcon,
    iconClass: "text-warning",
    tileClass: "bg-warning-subtle",
    getLink: workerCredentialsLink,
  },
  credential_expired: {
    label: "A credential expired",
    category: "Workers",
    icon: ShieldAlertIcon,
    iconClass: "text-destructive",
    tileClass: "bg-danger-subtle",
    getLink: workerCredentialsLink,
  },
  credential_document_uploaded: {
    label: "A credential document is uploaded",
    category: "Workers",
    icon: FileCheck02Icon,
    iconClass: "text-brand",
    tileClass: "bg-brand/10",
    getLink: workerCredentialsLink,
  },
  report_run_completed: { ...REPORT_READY, label: "A report finishes" },
  report_run_delivered: {
    ...REPORT_READY,
    label: "A scheduled report is delivered",
    iconClass: "text-brand",
    tileClass: "bg-brand/10",
  },
  report_run_failed: {
    label: "A report fails",
    category: "Reports",
    icon: FileX02Icon,
    iconClass: "text-destructive",
    tileClass: "bg-danger-subtle",
    getLink: reportRunsLink,
  },
  report_run_canceled: {
    label: "A report run is canceled",
    category: "Reports",
    icon: SlashCircle01Icon,
    iconClass: "text-muted-foreground",
    tileClass: "bg-muted",
    getLink: reportRunsLink,
  },
  report_schedule_skipped: {
    label: "A schedule is skipped",
    category: "Reports",
    icon: CalendarMinus01Icon,
    iconClass: "text-warning",
    tileClass: "bg-warning-subtle",
    getLink: () => "/reports",
  },
  report_needs_attention: {
    label: "A report needs attention",
    category: "Reports",
    icon: FileAlertIcon,
    iconClass: "text-warning",
    tileClass: "bg-warning-subtle",
    getLink: needsAttentionLink,
  },
  report_delivery_email_failed: {
    label: "A report email bounces",
    category: "Reports",
    icon: MailAlertIcon,
    iconClass: "text-warning",
    tileClass: "bg-warning-subtle",
    getLink: reportRunsLink,
  },
  invoice_reconciliation_warning: {
    label: "An invoice won’t reconcile",
    category: "Billing",
    icon: InvoiceIcon,
    iconClass: "text-warning",
    tileClass: "bg-warning-subtle",
    getLink: invoiceLink,
  },
  record_shared: {
    why: () => WHY.shared,
    label: "Someone shares a record with you",
    category: "Shared with you",
    icon: Share07Icon,
    iconClass: "text-brand",
    tileClass: "bg-brand/10",
    avatar: (n) => {
      const userId = notificationDataString(n, "sharedById");
      const name = notificationDataString(n, "sharedByName");
      return userId || name ? { userId: userId ?? undefined, name: name ?? undefined } : null;
    },
    getLink: (n) => notificationDataString(n, "link") ?? null,
  },
  billing_exception_recorded: {
    label: "A billing exception is recorded",
    category: "Billing",
    icon: CurrencyDollarCircleIcon,
    iconClass: "text-warning",
    tileClass: "bg-warning-subtle",
    getLink: (n) =>
      entityPanelLink("/shipment-management/shipments", notificationRelatedId(n, "shipmentId")),
  },
  bank_receipt_reconciliation_exception: {
    label: "A bank receipt doesn’t match",
    category: "Accounting",
    icon: Building08Icon,
    iconClass: "text-warning",
    tileClass: "bg-warning-subtle",
    getLink: (n) =>
      entityPanelLink(
        "/accounting/reconciliation/bank-receipts",
        notificationRelatedId(n, "bankReceiptId"),
      ),
  },
  shipment_comment_mention: {
    why: () => WHY.mentioned,
    label: "Someone @mentions you on a shipment",
    category: "Mentions",
    icon: AtSignIcon,
    iconClass: "text-brand",
    tileClass: "bg-brand/10",
    hideMessage: true,
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
    label: "An EDI message can’t be processed",
    category: "EDI",
    icon: SwitchHorizontal01Icon,
    iconClass: "text-destructive",
    tileClass: "bg-danger-subtle",
    getLink: (n) => notificationDataString(n, "link") ?? "/edi/messages",
  },
  "edi.inbound_file.quarantined": {
    label: "An inbound file is quarantined",
    category: "EDI",
    icon: FileAlertIcon,
    iconClass: "text-destructive",
    tileClass: "bg-danger-subtle",
    getLink: (n) => notificationDataString(n, "link") ?? "/edi/inbound-files",
  },
  carrier_intel_block: {
    label: "A carrier is blocked",
    category: "Carrier Intelligence",
    icon: ShieldBanIcon,
    iconClass: "text-destructive",
    tileClass: "bg-danger-subtle",
    getLink: carrierIntelLink,
  },
  carrier_intel_change: {
    label: "A monitored carrier changes",
    category: "Carrier Intelligence",
    icon: ShieldQuestionIcon,
    iconClass: "text-warning",
    tileClass: "bg-warning-subtle",
    getLink: carrierIntelLink,
  },
  carrier_intel_digest: {
    label: "The weekly monitoring digest",
    category: "Carrier Intelligence",
    icon: ListChecksIcon,
    iconClass: "text-brand",
    tileClass: "bg-brand/10",
    getLink: (n) => notificationDataString(n, "link") ?? CARRIER_MONITORING_PATH,
  },
  carrier_intel_provider_paused: {
    label: "A data provider pauses",
    category: "Carrier Intelligence",
    icon: PauseCircleIcon,
    iconClass: "text-destructive",
    tileClass: "bg-danger-subtle",
    getLink: (n) =>
      notificationDataString(n, "link") ?? "/admin/integrations?category=CarrierCompliance",
  },
  carrier_equipment_mismatch: {
    label: "Equipment doesn’t match the load",
    category: "Carrier Intelligence",
    icon: Truck01Icon,
    iconClass: "text-destructive",
    tileClass: "bg-danger-subtle",
    getLink: carrierIntelLink,
  },
  carrier_intel_spend_soft_cap: {
    label: "Lookup spend nears its cap",
    category: "Carrier Intelligence",
    icon: Coins01Icon,
    iconClass: "text-warning",
    tileClass: "bg-warning-subtle",
    getLink: carrierIntelUsageLink,
  },
  carrier_intel_spend_cap: {
    label: "Lookup spend reaches its cap",
    category: "Carrier Intelligence",
    icon: Coins01Icon,
    iconClass: "text-destructive",
    tileClass: "bg-danger-subtle",
    getLink: carrierIntelUsageLink,
  },
  tender_accepted: {
    label: "A carrier accepts a tender",
    category: "Dispatch",
    icon: CheckCircleIcon,
    iconClass: "text-success",
    tileClass: "bg-success-subtle",
    getLink: dispatchConsoleLink,
  },
  tender_needs_review: {
    label: "A tender needs your review",
    category: "Dispatch",
    icon: AlertTriangleIcon,
    iconClass: "text-warning",
    tileClass: "bg-warning-subtle",
    getLink: dispatchConsoleLink,
  },
  tender_waterfall_exhausted: {
    label: "A waterfall runs out of carriers",
    category: "Dispatch",
    icon: AlertOctagonIcon,
    iconClass: "text-destructive",
    tileClass: "bg-danger-subtle",
    getLink: dispatchConsoleLink,
  },
  rate_confirmation_issue_failed: {
    label: "A rate confirmation fails to issue",
    category: "Dispatch",
    icon: FileAlertIcon,
    iconClass: "text-destructive",
    tileClass: "bg-danger-subtle",
    getLink: dispatchConsoleLink,
  },
  tender_delivery_failed: {
    label: "A tender can’t be delivered",
    category: "Dispatch",
    icon: MailAlertIcon,
    iconClass: "text-destructive",
    tileClass: "bg-danger-subtle",
    getLink: dispatchConsoleLink,
  },
  tender_entries_skipped: {
    label: "Carriers are skipped in a waterfall",
    category: "Dispatch",
    icon: FilterFunnelXIcon,
    iconClass: "text-warning",
    tileClass: "bg-warning-subtle",
    getLink: dispatchConsoleLink,
  },
};

const TCA_DESCRIPTOR: NotificationDescriptor = {
  label: "A table you watch changes",
  category: "Data alert",
  icon: Database02Icon,
  iconClass: "text-info",
  tileClass: "bg-info-subtle",
  getLink: (n) => notificationDataString(n, "link"),
};

const FALLBACK_DESCRIPTOR: NotificationDescriptor = {
  label: "Account and product notices",
  category: "System",
  icon: Bell01Icon,
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

/** The preference key that turns every table change alert off at once. */
export const TCA_PREFERENCE_KEY = "tca.*";
/** The preference key for notices no producer registers: account and product news. */
export const SYSTEM_PREFERENCE_KEY = "system";

/**
 * Kinds an organization requires its people to receive. The bell always shows them and
 * their switch stays on; the reason is shown where the switch would be.
 */
export const REQUIRED_NOTIFICATION_TYPES = defineLabels({
  [AI_AUDIT_CHAIN_MISMATCH_EVENT]: "Your organization requires this for admins",
  [SYSTEM_PREFERENCE_KEY]: "Security notices always reach you in Trenova",
});

function requiredReason(key: string): string | null {
  return Object.hasOwn(REQUIRED_NOTIFICATION_TYPES, key)
    ? (REQUIRED_NOTIFICATION_TYPES as Readonly<Record<string, string>>)[key]
    : null;
}

interface CategoryLook {
  /** The registry's category, which the look is matched by. */
  name: string;
  /** The category as a person reads it; its renderer translates it. */
  label: string;
  /** One line on what the category covers; its renderer translates it. */
  description: string;
  icon: IconComponent;
  iconClass: string;
  tileClass: string;
}

const CATEGORY_LOOKS: readonly CategoryLook[] = [
  {
    name: "Mentions",
    label: "Mentions",
    description: "When a teammate @mentions you",
    icon: AtSignIcon,
    iconClass: "text-brand",
    tileClass: "bg-brand-subtle",
  },
  {
    name: "Shared with you",
    label: "Shared with you",
    description: "When a teammate shares a record with you",
    icon: Share07Icon,
    iconClass: "text-brand",
    tileClass: "bg-brand-subtle",
  },
  {
    name: "Assistant",
    label: "Assistant",
    description: "Replies you walked away from",
    icon: AssistMark,
    iconClass: "text-muted-foreground",
    tileClass: "bg-muted",
  },
  {
    name: "Dispatch",
    label: "Dispatch",
    description: "Tenders, waterfalls and rate confirmations",
    icon: Truck01Icon,
    iconClass: "text-brand",
    tileClass: "bg-brand-subtle",
  },
  {
    name: "Exceptions",
    label: "Exceptions",
    description: "Late, missing and at-risk work on the Desk",
    icon: ClockAlertIcon,
    iconClass: "text-warning",
    tileClass: "bg-warning-subtle",
  },
  {
    name: "Billing",
    label: "Billing",
    description: "Invoices and billing exceptions",
    icon: InvoiceIcon,
    iconClass: "text-warning",
    tileClass: "bg-warning-subtle",
  },
  {
    name: "Accounting",
    label: "Accounting",
    description: "Bank receipt reconciliation",
    icon: Building08Icon,
    iconClass: "text-warning",
    tileClass: "bg-warning-subtle",
  },
  {
    name: "Carrier Intelligence",
    label: "Carrier intelligence",
    description: "Monitoring, blocks and spend caps",
    icon: ShieldQuestionIcon,
    iconClass: "text-destructive",
    tileClass: "bg-danger-subtle",
  },
  {
    name: "Workers",
    label: "Workers",
    description: "Time off and driver credentials",
    icon: IdCardIcon,
    iconClass: "text-brand",
    tileClass: "bg-brand-subtle",
  },
  {
    name: "Reports",
    label: "Reports",
    description: "Runs and scheduled deliveries",
    icon: FileDownload02Icon,
    iconClass: "text-success",
    tileClass: "bg-success-subtle",
  },
  {
    name: "EDI",
    label: "EDI",
    description: "Messages and inbound files",
    icon: SwitchHorizontal01Icon,
    iconClass: "text-destructive",
    tileClass: "bg-danger-subtle",
  },
  {
    name: "AI Control",
    label: "AI Control",
    description: "Proposals, tool trust and audit exports",
    icon: ListChecksIcon,
    iconClass: "text-brand",
    tileClass: "bg-brand-subtle",
  },
  {
    name: "Data alert",
    label: "Data alerts",
    description: "Table change alerts you subscribed to",
    icon: Database02Icon,
    iconClass: "text-info",
    tileClass: "bg-info-subtle",
  },
  {
    name: "System",
    label: "System",
    description: "Product and account notices",
    icon: Bell01Icon,
    iconClass: "text-muted-foreground",
    tileClass: "bg-muted",
  },
];

export interface NotificationCatalogType {
  /** The key a preference stores: the event type, or a wildcard for a family of them. */
  key: string;
  /** English source; translate at render. */
  label: string;
  urgent: boolean;
  /** Why the type cannot be turned off, when it cannot, in the language on screen. */
  requiredReason: string | null;
}

export interface NotificationCatalogCategory {
  /** The registry's category. */
  name: string;
  /** English source; translate at render. */
  label: string;
  /** English source; translate at render. */
  description: string;
  icon: IconComponent;
  iconClass: string;
  tileClass: string;
  types: NotificationCatalogType[];
}

/** A kind is urgent when its descriptor is drawn in the danger tone. */
export function isUrgentDescriptor(descriptor: Pick<NotificationDescriptor, "iconClass">): boolean {
  return descriptor.iconClass === "text-destructive";
}

function catalogType(key: string, descriptor: NotificationDescriptor): NotificationCatalogType {
  return {
    key,
    label: descriptor.label ?? key,
    urgent: isUrgentDescriptor(descriptor),
    requiredReason: requiredReason(key),
  };
}

/**
 * Every kind of notification Trenova sends, grouped by the registry's category in the
 * order a person reads them: what needs them first, news last. A category the looks
 * above do not name yet takes the look of its first kind and sorts last.
 */
export function buildNotificationCatalog(): NotificationCatalogCategory[] {
  const entries: Array<[string, NotificationDescriptor]> = [
    ...Object.entries(EXACT_REGISTRY),
    [TCA_PREFERENCE_KEY, TCA_DESCRIPTOR],
    [SYSTEM_PREFERENCE_KEY, FALLBACK_DESCRIPTOR],
  ];
  const looks = new Map(CATEGORY_LOOKS.map((look, index) => [look.name, { look, index }]));
  const byCategory = new Map<string, NotificationCatalogCategory & { order: number }>();
  for (const [key, descriptor] of entries) {
    let category = byCategory.get(descriptor.category);
    if (!category) {
      const known = looks.get(descriptor.category);
      category = {
        name: descriptor.category,
        label: known?.look.label ?? descriptor.category,
        description: known?.look.description ?? "",
        icon: known?.look.icon ?? descriptor.icon,
        iconClass: known?.look.iconClass ?? descriptor.iconClass,
        tileClass: known?.look.tileClass ?? descriptor.tileClass,
        types: [],
        order: known?.index ?? CATEGORY_LOOKS.length,
      };
      byCategory.set(descriptor.category, category);
    }
    category.types.push(catalogType(key, descriptor));
  }

  return [...byCategory.values()]
    .sort((a, b) => a.order - b.order || a.name.localeCompare(b.name))
    .map(({ order: _order, ...category }) => category);
}

/** Why the person got this notification, or null when the row needs no line for it. */
export function notificationWhy(notification: Notification): string | null {
  return resolveNotificationDescriptor(notification).why?.(notification) ?? null;
}

/** What the row offers beyond opening it; empty for a kind that is only news. */
export function notificationActions(notification: Notification): NotificationRowAction[] {
  return resolveNotificationDescriptor(notification).actions?.(notification) ?? [];
}

/** A state the row carries as a badge, or null when its kind has none to show. */
export function notificationStatus(notification: Notification): NotificationStatus | null {
  return resolveNotificationDescriptor(notification).status?.(notification) ?? null;
}

/**
 * The key a preference stores to turn this kind of notification off, or null when the
 * organization requires it and it cannot be turned off.
 */
export function notificationPreferenceKey(
  notification: Pick<Notification, "eventType">,
): string | null {
  const { eventType } = notification;
  const key = Object.hasOwn(EXACT_REGISTRY, eventType)
    ? eventType
    : eventType.startsWith("tca.")
      ? TCA_PREFERENCE_KEY
      : SYSTEM_PREFERENCE_KEY;

  return requiredReason(key) === null ? key : null;
}

/** One earlier time a counted notification's subject came up, newest first. */
export interface NotificationOccurrence {
  label: string;
  /** The record's own number, where the producer gave one. */
  ref: string | null;
  link: string | null;
  at: number;
}

/** The last few times the subject of a counted notification came up. */
export function notificationRecent(
  notification: Pick<Notification, "data">,
): NotificationOccurrence[] {
  const recent = notification.data?.recent;
  if (!Array.isArray(recent)) return [];

  return recent.flatMap((entry) => {
    if (typeof entry !== "object" || entry === null) return [];
    const { label, ref, link, at } = entry as Record<string, unknown>;
    if (typeof label !== "string" || label === "" || typeof at !== "number") return [];
    return [
      {
        label,
        ref: typeof ref === "string" && ref !== "" ? ref : null,
        link: typeof link === "string" && link !== "" ? link : null,
        at,
      },
    ];
  });
}
