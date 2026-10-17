import { recordPath } from "@/config/record-links";
import { newShipmentForCustomerPath, SHIPMENTS_PATH } from "@/lib/shipment-links";
import { NovaBadge } from "@trenova/shared/components/ui/nova-badge";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import {
  ArrowUpRightIcon,
  BookOpen02Icon,
  Building03Icon,
  CalendarDateIcon,
  Copy01Icon,
  CornerDownLeftIcon,
  Download01Icon,
  EyeIcon,
  FolderIcon,
  Link01Icon,
  Pin01Icon,
  PinOffIcon,
  PlusIcon,
  ShieldTickIcon,
  Truck01Icon,
  User01Icon,
} from "@trenova/shared/components/icons";
import type {
  PaletteAction,
  PaletteCommand,
  PaletteItem,
  PalettePage,
  PaletteRecord,
} from "./palette-model";

/**
 * The ids of the actions a key reaches from the result list without opening
 * the actions menu. A row that has no such action does nothing on that key.
 */
export const ACTION_OPEN = "open";
export const ACTION_ASK = "ask";
export const ACTION_NEW_TAB = "new-tab";
export const ACTION_COPY_LINK = "copy-link";
export const ACTION_COPY_ID = "copy-id";
export const ACTION_PIN = "toggle-pin";

export const CUSTOMER_LEDGER_PATH = "/accounting/ar/customer-ledger";

const UNASSIGNED_SHIPMENT_STATUSES = new Set(["New", "PartiallyAssigned"]);

export interface ActionContext {
  mac: boolean;
  /** Whether a page is one this person may open, from the same filtered navigation the sidebar draws. */
  canReach: (path: string) => boolean;
  canUseAssistant: boolean;
  canCreateShipment: boolean;
  pinnedUrls: ReadonlySet<string>;
  /** The commands that start something on a page, such as "New shipment" on Shipments. */
  relatedCommands: (href: string) => readonly PaletteCommand[];
}

export function modKey(context: Pick<ActionContext, "mac">): string {
  return context.mac ? "⌘" : "Ctrl";
}

function altKey(context: Pick<ActionContext, "mac">): string {
  return context.mac ? "⌥" : "Alt";
}

function openAction(label: string, intent: PaletteAction["intent"], icon = CornerDownLeftIcon) {
  return { id: ACTION_OPEN, label, icon, intent, shortcut: ["↵"] } satisfies PaletteAction;
}

function linkActions(href: string, context: ActionContext, t: TranslateFn): PaletteAction[] {
  const mod = modKey(context);
  return [
    {
      id: ACTION_NEW_TAB,
      label: t("Open in new tab"),
      icon: ArrowUpRightIcon,
      intent: { type: "new-tab", href },
      shortcut: [mod, "↵"],
    },
    {
      id: ACTION_COPY_LINK,
      label: t("Copy link"),
      icon: Link01Icon,
      intent: { type: "copy-link", href },
      shortcut: [mod, "L"],
    },
  ];
}

/** Whole phrases per type: a noun spliced into a sentence does not translate. */
function openPhrase(record: PaletteRecord, t: TranslateFn): string {
  switch (record.entityType) {
    case "shipment":
      return t("Open shipment");
    case "customer":
      return t("Open customer");
    case "worker":
      return t("Open worker");
    case "document":
      return t("Preview document");
  }
}

function copyIdAction(
  record: PaletteRecord,
  context: ActionContext,
  t: TranslateFn,
): PaletteAction {
  const shortcut = [altKey(context), "C"];
  switch (record.entityType) {
    case "shipment":
      return {
        id: ACTION_COPY_ID,
        label: t("Copy PRO number"),
        icon: Copy01Icon,
        intent: { type: "copy", text: record.metadata.proNumber || record.title, what: "pro" },
        shortcut,
      };
    case "document":
      return {
        id: ACTION_COPY_ID,
        label: t("Copy file name"),
        icon: Copy01Icon,
        intent: { type: "copy", text: record.title, what: "file-name" },
        shortcut,
      };
    default:
      return {
        id: ACTION_COPY_ID,
        label: t("Copy name"),
        icon: Copy01Icon,
        intent: { type: "copy", text: record.title, what: "name" },
        shortcut,
      };
  }
}

function askAction(record: PaletteRecord, context: ActionContext, t: TranslateFn): PaletteAction[] {
  if (!context.canUseAssistant) {
    return [];
  }
  return [
    {
      id: ACTION_ASK,
      label: t("Ask Nova about this"),
      icon: NovaBadge,
      intent: {
        type: "ask-about",
        subject: { type: record.entityType, id: record.id, label: record.title },
      },
      shortcut: [modKey(context), "I"],
    },
  ];
}

function shipmentActions(record: PaletteRecord, t: TranslateFn): PaletteAction[] {
  const meta = record.metadata;
  const actions: PaletteAction[] = [];
  if (!meta.driverName && UNASSIGNED_SHIPMENT_STATUSES.has(meta.status ?? "")) {
    actions.push({
      id: "assign-driver",
      label: t("Assign a driver"),
      icon: User01Icon,
      intent: { type: "navigate", href: record.href },
    });
  }
  if (meta.customerId && meta.customerName) {
    actions.push({
      id: "open-customer",
      label: t("Open {0}", meta.customerName),
      icon: Building03Icon,
      intent: { type: "navigate", href: recordPath("customer", meta.customerId) },
    });
  }
  if (meta.bol) {
    actions.push({
      id: "copy-bol",
      label: t("Copy BOL"),
      icon: Copy01Icon,
      intent: { type: "copy", text: meta.bol, what: "bol" },
    });
  }
  return actions;
}

function customerActions(
  record: PaletteRecord,
  context: ActionContext,
  t: TranslateFn,
): PaletteAction[] {
  const actions: PaletteAction[] = [];
  if (context.canReach(CUSTOMER_LEDGER_PATH)) {
    const params = new URLSearchParams({ customerId: record.id });
    actions.push({
      id: "customer-ledger",
      label: t("Open customer ledger"),
      icon: BookOpen02Icon,
      intent: { type: "navigate", href: `${CUSTOMER_LEDGER_PATH}?${params.toString()}` },
    });
  }
  if (context.canCreateShipment && context.canReach(SHIPMENTS_PATH)) {
    actions.push({
      id: "new-shipment-for-customer",
      label: t("New shipment for {0}", record.title),
      icon: PlusIcon,
      intent: { type: "navigate", href: newShipmentForCustomerPath(record.id) },
    });
  }
  return actions;
}

function workerActions(record: PaletteRecord, t: TranslateFn): PaletteAction[] {
  const tab = (name: string) => recordPath("worker", record.id, { tab: name });
  const actions: PaletteAction[] = [
    {
      id: "worker-compliance",
      label: t("Open compliance"),
      icon: ShieldTickIcon,
      intent: { type: "navigate", href: tab("compliance") },
    },
    {
      id: "worker-pto",
      label: t("Open time off"),
      icon: CalendarDateIcon,
      intent: { type: "navigate", href: tab("pto") },
    },
  ];
  const loadId = record.metadata.currentLoadId;
  const load = record.metadata.currentLoad;
  if (loadId && load) {
    actions.push({
      id: "open-current-load",
      label: t("Open {0}", load),
      icon: Truck01Icon,
      intent: { type: "navigate", href: recordPath("shipment", loadId) },
    });
  }
  return actions;
}

function documentActions(record: PaletteRecord, t: TranslateFn): PaletteAction[] {
  const attached = record.metadata.attachedLabel;
  return [
    {
      id: "document-download",
      label: t("Download"),
      icon: Download01Icon,
      intent: { type: "open-document", documentId: record.id, disposition: "download" },
    },
    {
      id: "document-record",
      label: attached ? t("Open {0}", attached) : t("Open the record it's attached to"),
      icon: FolderIcon,
      intent: { type: "navigate", href: record.href },
    },
  ];
}

function recordActions(
  record: PaletteRecord,
  context: ActionContext,
  t: TranslateFn,
): PaletteAction[] {
  const open =
    record.entityType === "document"
      ? openAction(
          openPhrase(record, t),
          { type: "open-document", documentId: record.id, disposition: "view" },
          EyeIcon,
        )
      : openAction(openPhrase(record, t), { type: "navigate", href: record.href });

  const own = (() => {
    switch (record.entityType) {
      case "shipment":
        return shipmentActions(record, t);
      case "customer":
        return customerActions(record, context, t);
      case "worker":
        return workerActions(record, t);
      case "document":
        return documentActions(record, t);
    }
  })();

  return [
    open,
    ...askAction(record, context, t),
    ...linkActions(record.href, context, t),
    copyIdAction(record, context, t),
    ...own,
  ];
}

function pageActions(page: PalettePage, context: ActionContext, t: TranslateFn): PaletteAction[] {
  const pinned = context.pinnedUrls.has(page.href);
  return [
    openAction(t("Open page"), { type: "navigate", href: page.href }),
    ...linkActions(page.href, context, t),
    {
      id: ACTION_PIN,
      label: pinned ? t("Unpin from home") : t("Pin to home"),
      icon: pinned ? PinOffIcon : Pin01Icon,
      intent: { type: "toggle-pin", pageUrl: page.href, pageTitle: page.title },
      shortcut: [modKey(context), "P"],
    },
    ...context.relatedCommands(page.href).map((command): PaletteAction => ({
      id: `related:${command.id}`,
      label: command.label,
      icon: command.icon,
      intent: command.intent,
      shortcut: command.shortcut,
    })),
  ];
}

/**
 * Everything that can be done with one row, first action first. The first
 * is what Enter does; the rest are the actions menu, and the few with a
 * shortcut also answer that shortcut from the result list.
 */
export function buildItemActions(
  item: PaletteItem,
  context: ActionContext,
  t: TranslateFn,
): PaletteAction[] {
  switch (item.kind) {
    case "record":
      return recordActions(item.record, context, t);
    case "page":
      return pageActions(item.page, context, t);
    case "command":
      return [
        {
          id: ACTION_OPEN,
          label: item.command.label,
          icon: item.command.icon,
          intent: item.command.intent,
          shortcut: ["↵"],
          destructive: item.command.destructive,
        },
      ];
    case "attention":
      return [
        openAction(t("Open the queue"), { type: "navigate", href: item.attention.href }),
        ...linkActions(item.attention.href, context, t),
      ];
    case "ask":
    case "widen":
      return [];
  }
}

export function findAction(
  actions: readonly PaletteAction[],
  id: string,
): PaletteAction | undefined {
  return actions.find((action) => action.id === id);
}
