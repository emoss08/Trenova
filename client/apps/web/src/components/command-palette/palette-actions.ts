import { recordPath } from "@/config/record-links";
import { AssistMark } from "@trenova/shared/components/ui/assist-mark";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import {
  ArrowUpRightIcon,
  BookOpenTextIcon,
  CalendarDaysIcon,
  CheckIcon,
  CopyIcon,
  CornerDownLeftIcon,
  DownloadIcon,
  EyeIcon,
  FolderOpenIcon,
  LinkIcon,
  PinIcon,
  PinOffIcon,
  ShieldCheckIcon,
} from "lucide-react";
import type { PaletteAction, PaletteItem, PaletteRecord } from "./palette-model";

/**
 * The ids of the actions a key reaches without opening the action list. A
 * row that has no such action simply does nothing on that key.
 */
export const ACTION_OPEN = "open";
export const ACTION_NEW_TAB = "new-tab";
export const ACTION_COPY_LINK = "copy-link";
export const ACTION_COPY_ID = "copy-id";

export const CUSTOMER_LEDGER_PATH = "/accounting/ar/customer-ledger";

export interface ActionContext {
  mac: boolean;
  /** Whether a page is one this person may open, from the same filtered navigation the sidebar draws. */
  canReach: (path: string) => boolean;
  canUseAssistant: boolean;
  pinnedUrls: ReadonlySet<string>;
}

function linkActions(href: string, context: ActionContext, t: TranslateFn): PaletteAction[] {
  const mod = context.mac ? "⌘" : "Ctrl";
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
      icon: LinkIcon,
      intent: { type: "copy-link", href },
      shortcut: [mod, "L"],
    },
  ];
}

function copyIdAction(label: string, text: string, context: ActionContext): PaletteAction {
  return {
    id: ACTION_COPY_ID,
    label,
    icon: CopyIcon,
    intent: { type: "copy", text },
    shortcut: [context.mac ? "⌥" : "Alt", "C"],
  };
}

/** Whole phrases per type: a noun spliced into a sentence does not translate. */
function recordPhrases(record: PaletteRecord, t: TranslateFn): { open: string; ask: string } {
  switch (record.entityType) {
    case "shipment":
      return { open: t("Open shipment"), ask: t("Ask the assistant about this shipment") };
    case "customer":
      return { open: t("Open customer"), ask: t("Ask the assistant about this customer") };
    case "worker":
      return { open: t("Open worker"), ask: t("Ask the assistant about this worker") };
    case "document":
      return { open: t("Preview document"), ask: t("Ask the assistant about this document") };
  }
}

function askAction(record: PaletteRecord, context: ActionContext, t: TranslateFn): PaletteAction[] {
  if (!context.canUseAssistant) {
    return [];
  }
  return [
    {
      id: "ask",
      label: recordPhrases(record, t).ask,
      icon: AssistMark,
      intent: {
        type: "ask-about",
        subject: { type: record.entityType, id: record.id, label: record.title },
      },
    },
  ];
}

function recordActions(
  record: PaletteRecord,
  context: ActionContext,
  t: TranslateFn,
): PaletteAction[] {
  const phrases = recordPhrases(record, t);

  switch (record.entityType) {
    case "shipment": {
      const proNumber = record.metadata.proNumber || record.title;
      const actions: PaletteAction[] = [
        {
          id: ACTION_OPEN,
          label: phrases.open,
          icon: CornerDownLeftIcon,
          intent: { type: "navigate", href: record.href },
          shortcut: ["↵"],
        },
        ...linkActions(record.href, context, t),
        copyIdAction(t("Copy PRO number"), proNumber, context),
      ];
      if (record.metadata.bol) {
        actions.push({
          id: "copy-bol",
          label: t("Copy BOL"),
          icon: CopyIcon,
          intent: { type: "copy", text: record.metadata.bol },
        });
      }
      return [...actions, ...askAction(record, context, t)];
    }
    case "customer": {
      const actions: PaletteAction[] = [
        {
          id: ACTION_OPEN,
          label: phrases.open,
          icon: CornerDownLeftIcon,
          intent: { type: "navigate", href: record.href },
          shortcut: ["↵"],
        },
      ];
      if (context.canReach(CUSTOMER_LEDGER_PATH)) {
        const params = new URLSearchParams({ customerId: record.id });
        actions.push({
          id: "customer-ledger",
          label: t("Open customer ledger"),
          icon: BookOpenTextIcon,
          intent: { type: "navigate", href: `${CUSTOMER_LEDGER_PATH}?${params.toString()}` },
        });
      }
      return [
        ...actions,
        ...linkActions(record.href, context, t),
        copyIdAction(t("Copy name"), record.title, context),
        ...askAction(record, context, t),
      ];
    }
    case "worker": {
      const tab = (name: string) => recordPath("worker", record.id, { tab: name });
      return [
        {
          id: ACTION_OPEN,
          label: phrases.open,
          icon: CornerDownLeftIcon,
          intent: { type: "navigate", href: record.href },
          shortcut: ["↵"],
        },
        {
          id: "worker-compliance",
          label: t("Open compliance"),
          icon: ShieldCheckIcon,
          intent: { type: "navigate", href: tab("compliance") },
        },
        {
          id: "worker-pto",
          label: t("Open time off"),
          icon: CalendarDaysIcon,
          intent: { type: "navigate", href: tab("pto") },
        },
        {
          id: "worker-documents",
          label: t("Open documents"),
          icon: FolderOpenIcon,
          intent: { type: "navigate", href: tab("documents") },
        },
        ...linkActions(record.href, context, t),
        copyIdAction(t("Copy name"), record.title, context),
        ...askAction(record, context, t),
      ];
    }
    case "document":
      return [
        {
          id: ACTION_OPEN,
          label: phrases.open,
          icon: EyeIcon,
          intent: { type: "open-document", documentId: record.id, disposition: "view" },
          shortcut: ["↵"],
        },
        {
          id: "document-download",
          label: t("Download"),
          icon: DownloadIcon,
          intent: { type: "open-document", documentId: record.id, disposition: "download" },
        },
        {
          id: "document-record",
          label: t("Open the record it's attached to"),
          icon: FolderOpenIcon,
          intent: { type: "navigate", href: record.href },
        },
        ...linkActions(record.href, context, t),
        copyIdAction(t("Copy file name"), record.title, context),
        ...askAction(record, context, t),
      ];
  }
}

/**
 * Everything that can be done with one row, first action first. The first
 * is what Enter does; the rest are the action list, and the few with a
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
    case "page": {
      const pinned = context.pinnedUrls.has(item.page.href);
      return [
        {
          id: ACTION_OPEN,
          label: t("Open page"),
          icon: CornerDownLeftIcon,
          intent: { type: "navigate", href: item.page.href },
          shortcut: ["↵"],
        },
        ...linkActions(item.page.href, context, t),
        {
          id: "toggle-pin",
          label: pinned ? t("Unpin page") : t("Pin page"),
          icon: pinned ? PinOffIcon : PinIcon,
          intent: { type: "toggle-pin", pageUrl: item.page.href, pageTitle: item.page.title },
        },
      ];
    }
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
        {
          id: ACTION_OPEN,
          label: t("Open {0}", item.attention.label),
          icon: CornerDownLeftIcon,
          intent: { type: "navigate", href: item.attention.href },
          shortcut: ["↵"],
        },
        ...linkActions(item.attention.href, context, t),
      ];
    case "notification": {
      const actions: PaletteAction[] = [];
      if (item.href) {
        actions.push(
          {
            id: ACTION_OPEN,
            label: t("Open"),
            icon: CornerDownLeftIcon,
            intent: { type: "navigate", href: item.href },
            shortcut: ["↵"],
          },
          ...linkActions(item.href, context, t),
        );
      } else {
        actions.push({
          id: ACTION_OPEN,
          label: t("Open notifications"),
          icon: CornerDownLeftIcon,
          intent: { type: "open-dialog", dialog: "notifications" },
          shortcut: ["↵"],
        });
      }
      if (item.notification.readAt === null) {
        actions.push({
          id: "mark-read",
          label: t("Mark as read"),
          icon: CheckIcon,
          intent: { type: "mark-notification-read", notificationId: item.notification.id },
        });
      }
      return actions;
    }
    case "ask":
      return [];
  }
}

export function findAction(
  actions: readonly PaletteAction[],
  id: string,
): PaletteAction | undefined {
  return actions.find((action) => action.id === id);
}
