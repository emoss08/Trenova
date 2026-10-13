import type { AppDialog } from "@/stores/app-dialogs-store";
import type { GlobalSearchEntityType } from "@/services/global-search";
import type { AssistantEntityRef } from "@/types/assistant";
import type { FieldsMatch } from "@trenova/shared/lib/fuzzy-score";
import type { PageGlanceKey } from "./route-command-data";

export type PaletteIcon = React.ComponentType<{ className?: string; strokeWidth?: number }>;

export type PaletteTheme = "light" | "dark" | "system";

/**
 * Everything the palette can do, as data. Rows, row actions and commands all
 * describe what they do with one of these, and one runner carries them out,
 * so "open in a new tab" means the same thing on a shipment as on a page.
 */
export type PaletteIntent =
  | { type: "navigate"; href: string }
  | { type: "new-tab"; href: string }
  | { type: "copy"; text: string; what: PaletteCopyKind }
  | { type: "copy-link"; href: string }
  | { type: "ask-about"; subject: AssistantEntityRef }
  | { type: "toggle-pin"; pageUrl: string; pageTitle: string }
  | { type: "set-theme"; theme: PaletteTheme }
  | { type: "sign-out" }
  | { type: "open-dialog"; dialog: AppDialog }
  | { type: "assistant"; mode: "open" | "new-chat" }
  | { type: "toggle-sidebar" }
  | { type: "mark-notification-read"; notificationId: string }
  | { type: "mark-all-notifications-read" }
  | { type: "open-document"; documentId: string; disposition: "view" | "download" };

/** What a copy put on the clipboard, so the confirmation can say so. */
export type PaletteCopyKind = "pro" | "bol" | "name" | "file-name" | "answer" | "text";

/**
 * The intents that leave the palette open: they change nothing on the page
 * behind it, or, for the theme, change it in a way the person wants to see
 * while the palette is still up. Each confirms in the footer instead.
 */
export function intentKeepsPaletteOpen(intent: PaletteIntent): boolean {
  return (
    intent.type === "copy" ||
    intent.type === "copy-link" ||
    intent.type === "toggle-pin" ||
    intent.type === "mark-notification-read" ||
    intent.type === "set-theme"
  );
}

export type PaletteCommandGroup =
  | "create"
  | "assistant"
  | "navigation"
  | "notifications"
  | "appearance"
  | "account";

export interface PaletteCommand {
  id: string;
  label: string;
  description: string;
  group: PaletteCommandGroup;
  icon: PaletteIcon;
  keywords: readonly string[];
  intent: PaletteIntent;
  /** Keys as they are pressed, one `Kbd` each. */
  shortcut?: string[];
  /** Marks the current choice among exclusive commands, such as the theme. */
  checked?: boolean;
  /** Draws the command in the danger tone, for the one that ends the session. */
  destructive?: boolean;
}

export interface PaletteRecord {
  entityType: GlobalSearchEntityType;
  id: string;
  title: string;
  subtitle?: string;
  href: string;
  metadata: Record<string, string>;
}

export interface PalettePage {
  id: string;
  title: string;
  /** Where the page sits, without the page itself: ["Billing", "Configuration files"]. */
  trail: readonly string[];
  href: string;
  icon: PaletteIcon;
  /** Other words people use for the page. A match through one is shown as its own tag. */
  keywords: readonly string[];
  module: string;
  /** The live counts the preview shows for the page, when it has any. */
  glance?: PageGlanceKey;
  /** The kind of record the page lists, for the recent records its preview offers. */
  recordType?: GlobalSearchEntityType;
}

export type PaletteAttentionTone = "neutral" | "warning" | "danger";

export interface PaletteAttention {
  key: string;
  label: string;
  module: string;
  count: number;
  href: string;
  tone: PaletteAttentionTone;
  /** One line on why these are waiting, for the preview. */
  reason: string;
}

/** Which characters a row's query landed on, field by field, and through what. */
export type PaletteMatch = FieldsMatch<string>;

export type PaletteItem =
  | {
      kind: "record";
      key: string;
      record: PaletteRecord;
      recent?: boolean;
      /** When the record was last opened, for "Jump back in". */
      visitedAt?: number;
      match?: PaletteMatch;
    }
  | { kind: "page"; key: string; page: PalettePage; pinned?: boolean; match?: PaletteMatch }
  | { kind: "command"; key: string; command: PaletteCommand; match?: PaletteMatch }
  | { kind: "attention"; key: string; attention: PaletteAttention }
  | { kind: "ask"; key: string; question: string; subject: AssistantEntityRef | null }
  | { kind: "widen"; key: string; query: string };

export interface PaletteSection {
  id: string;
  heading: string;
  items: PaletteItem[];
  /** How many there are in all, shown beside the heading when above one. */
  count?: number;
  /** The scope that lists every one of them, when the section shows only the first few. */
  seeAll?: PaletteScope;
  loading?: boolean;
  error?: boolean;
  /** How many skeleton rows stand in while loading. */
  placeholderRows?: number;
}

export interface PaletteAction {
  id: string;
  label: string;
  icon: PaletteIcon;
  intent: PaletteIntent;
  shortcut?: string[];
  destructive?: boolean;
}

/**
 * What narrows the palette: one kind of record, the app's pages, or its
 * commands. The record scopes are the global search's entity types, so the
 * tab strip and an `@` mention set the same thing.
 */
export type PaletteScope = "all" | GlobalSearchEntityType | "pages" | "commands";

export function isRecordScope(scope: PaletteScope): scope is GlobalSearchEntityType {
  return scope !== "all" && scope !== "pages" && scope !== "commands";
}

export function recordItemKey(entityType: GlobalSearchEntityType, id: string): string {
  return `record:${entityType}:${id}`;
}

/** The record a row stands for, whether it was found or remembered. */
export function itemRecord(item: PaletteItem | null): PaletteRecord | null {
  return item?.kind === "record" ? item.record : null;
}

export function itemTitle(item: PaletteItem): string {
  switch (item.kind) {
    case "record":
      return item.record.title;
    case "page":
      return item.page.title;
    case "command":
      return item.command.label;
    case "attention":
      return item.attention.label;
    case "ask":
      return item.question;
    case "widen":
      return item.query;
  }
}
