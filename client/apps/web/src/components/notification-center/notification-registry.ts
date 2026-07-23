import type { Notification } from "@trenova/shared/types/notification";
import {
  ArrowLeftRightIcon,
  AtSignIcon,
  BanIcon,
  BellIcon,
  CalendarOffIcon,
  CircleDollarSignIcon,
  DatabaseZapIcon,
  FileDownIcon,
  FileWarningIcon,
  FileXIcon,
  LandmarkIcon,
  MailWarningIcon,
  ReceiptTextIcon,
  type LucideIcon,
} from "lucide-react";

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
}

const reportRunsLink = () => "/reports/runs";

const REPORT_READY: NotificationDescriptor = {
  category: "Reports",
  icon: FileDownIcon,
  iconClass: "text-success",
  tileClass: "bg-success/10",
  getLink: reportRunsLink,
};

const EXACT_REGISTRY: Record<string, NotificationDescriptor> = {
  report_run_completed: REPORT_READY,
  report_run_delivered: { ...REPORT_READY, iconClass: "text-brand", tileClass: "bg-brand/10" },
  report_run_failed: {
    category: "Reports",
    icon: FileXIcon,
    iconClass: "text-destructive",
    tileClass: "bg-destructive/10",
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
    tileClass: "bg-warning/10",
    getLink: () => "/reports",
  },
  report_delivery_email_failed: {
    category: "Reports",
    icon: MailWarningIcon,
    iconClass: "text-warning",
    tileClass: "bg-warning/10",
    getLink: reportRunsLink,
  },
  invoice_reconciliation_warning: {
    category: "Billing",
    icon: ReceiptTextIcon,
    iconClass: "text-warning",
    tileClass: "bg-warning/10",
    getLink: (n) => entityPanelLink("/billing/invoices", notificationRelatedId(n, "invoiceId")),
  },
  billing_exception_recorded: {
    category: "Billing",
    icon: CircleDollarSignIcon,
    iconClass: "text-warning",
    tileClass: "bg-warning/10",
    getLink: (n) =>
      entityPanelLink(
        "/shipment-management/shipments",
        notificationRelatedId(n, "shipmentId"),
      ),
  },
  bank_receipt_reconciliation_exception: {
    category: "Accounting",
    icon: LandmarkIcon,
    iconClass: "text-warning",
    tileClass: "bg-warning/10",
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
    tileClass: "bg-destructive/10",
    getLink: (n) => notificationDataString(n, "link") ?? "/edi/messages",
  },
  "edi.inbound_file.quarantined": {
    category: "EDI",
    icon: FileWarningIcon,
    iconClass: "text-destructive",
    tileClass: "bg-destructive/10",
    getLink: (n) => notificationDataString(n, "link") ?? "/edi/inbound-files",
  },
};

const TCA_DESCRIPTOR: NotificationDescriptor = {
  category: "Data alert",
  icon: DatabaseZapIcon,
  iconClass: "text-info",
  tileClass: "bg-info/10",
};

const FALLBACK_DESCRIPTOR: NotificationDescriptor = {
  category: "System",
  icon: BellIcon,
  iconClass: "text-muted-foreground",
  tileClass: "bg-muted",
};

export function getNotificationDescriptor(eventType: string): NotificationDescriptor {
  const exact = EXACT_REGISTRY[eventType];
  if (exact) return exact;
  if (eventType.startsWith("tca.")) return TCA_DESCRIPTOR;
  return FALLBACK_DESCRIPTOR;
}

export function getNotificationLink(notification: Notification): string | null {
  return getNotificationDescriptor(notification.eventType).getLink?.(notification) ?? null;
}
