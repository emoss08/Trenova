import type { AppDialog } from "@/stores/app-dialogs-store";
import type { GlobalSearchEntityType } from "@/services/global-search";
import type { AssistantEntityRef } from "@/types/assistant";
import type { BadgeVariant } from "@trenova/shared/types/badge";
import type { Notification } from "@trenova/shared/types/notification";

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
  | { type: "copy"; text: string }
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

/** The intents that leave the palette open, because they change nothing on screen. */
export function intentKeepsPaletteOpen(intent: PaletteIntent): boolean {
  return (
    intent.type === "copy" ||
    intent.type === "copy-link" ||
    intent.type === "toggle-pin" ||
    intent.type === "mark-notification-read"
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
  keywords: string[];
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
  /** Where the page sits, "Billing > Configuration files > Customers". */
  trail: string;
  href: string;
  icon: PaletteIcon;
  keywords: string[];
  module: string;
}

export interface PaletteAttention {
  key: string;
  label: string;
  module: string;
  count: number;
  href: string;
  tone: BadgeVariant;
}

export type PaletteItem =
  | { kind: "record"; key: string; record: PaletteRecord; recent?: boolean }
  | { kind: "page"; key: string; page: PalettePage; pinned?: boolean; recent?: boolean }
  | { kind: "command"; key: string; command: PaletteCommand }
  | { kind: "attention"; key: string; attention: PaletteAttention }
  | { kind: "notification"; key: string; notification: Notification; href: string | null }
  | { kind: "ask"; key: string; question: string };

export interface PaletteSection {
  id: string;
  heading: string;
  items: PaletteItem[];
  /** A total larger than what is shown, such as every unread notification. */
  count?: number;
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
