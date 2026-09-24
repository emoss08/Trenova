import type { QuickActionCommand } from "@/config/navigation.types";
import { canAccessQuickAction, type NavAccessContext } from "@/hooks/use-filtered-navigation";
import { AssistMark } from "@trenova/shared/components/ui/assist-mark";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { Operation } from "@trenova/shared/types/permission";
import {
  BellIcon,
  CheckCheckIcon,
  KeyboardIcon,
  LinkIcon,
  LogOutIcon,
  MessageSquarePlusIcon,
  MonitorIcon,
  MoonIcon,
  PanelLeftIcon,
  PlusIcon,
  SettingsIcon,
  SunIcon,
  UserRoundIcon,
} from "lucide-react";
import { buildCommandHref } from "./route-command-data";
import type {
  PaletteCommand,
  PaletteCommandGroup,
  PaletteIcon,
  PaletteTheme,
} from "./palette-model";

export const COMMAND_GROUP_ORDER: readonly PaletteCommandGroup[] = [
  "create",
  "assistant",
  "navigation",
  "notifications",
  "appearance",
  "account",
];

export function commandGroupLabel(group: PaletteCommandGroup, t: TranslateFn): string {
  switch (group) {
    case "create":
      return t("Create");
    case "assistant":
      return t("Assistant");
    case "navigation":
      return t("Go to");
    case "notifications":
      return t("Notifications");
    case "appearance":
      return t("Appearance");
    case "account":
      return t("Account");
  }
}

/** The primary modifier as a key cap: the command glyph on a Mac, Ctrl elsewhere. */
export function modifierKey(mac: boolean): string {
  return mac ? "⌘" : "Ctrl";
}

/**
 * Quick actions from the navigation config, as palette commands. Most create
 * a record and are gated on Create; the few that open something instead
 * (the assistant, the insights page) are filed where a person would look for
 * them rather than under "Create".
 */
export function buildQuickActionCommands(
  definitions: readonly QuickActionCommand[],
  iconForPath: (path: string) => PaletteIcon | undefined,
  access: NavAccessContext,
  t: TranslateFn,
): PaletteCommand[] {
  return definitions
    .filter((definition) => canAccessQuickAction(definition, access))
    .map((definition) => {
      const href = buildCommandHref(definition.path, definition.query);
      const opensAssistant = definition.action === "open-assistant";
      const creates = !opensAssistant && definition.requiredOperation !== Operation.Read;
      const group: PaletteCommandGroup = opensAssistant
        ? "assistant"
        : creates
          ? "create"
          : "navigation";

      return {
        id: `quick:${definition.id}`,
        label: t(definition.label),
        description: t(definition.description),
        group,
        icon: opensAssistant
          ? AssistMark
          : creates
            ? PlusIcon
            : (iconForPath(buildCommandHref(definition.path)) ?? LinkIcon),
        keywords: [...(definition.keywords ?? []), creates ? "new create add" : ""].filter(Boolean),
        intent: opensAssistant ? { type: "assistant", mode: "open" } : { type: "navigate", href },
      } satisfies PaletteCommand;
    });
}

export interface AppCommandContext {
  theme: PaletteTheme;
  unreadNotifications: number;
  canUseAssistant: boolean;
  currentHref: string;
  mac: boolean;
}

/** The commands that act on the app itself rather than on a record or a page. */
export function buildAppCommands(context: AppCommandContext, t: TranslateFn): PaletteCommand[] {
  const mod = modifierKey(context.mac);
  const commands: PaletteCommand[] = [];

  if (context.canUseAssistant) {
    commands.push({
      id: "app:assistant-new",
      label: t("New conversation"),
      description: t("Start a fresh conversation with the assistant"),
      group: "assistant",
      icon: MessageSquarePlusIcon,
      keywords: ["assistant", "chat", "ai", "new", "thread", "conversation"],
      intent: { type: "assistant", mode: "new-chat" },
      shortcut: [mod, "J"],
    });
  }

  commands.push(
    {
      id: "app:notifications",
      label: t("Open notifications"),
      description:
        context.unreadNotifications > 0
          ? t(
              "{0, plural, one {# unread notification} other {# unread notifications}}",
              context.unreadNotifications,
            )
          : t("You're all caught up"),
      group: "notifications",
      icon: BellIcon,
      keywords: ["notifications", "inbox", "alerts", "bell", "unread"],
      intent: { type: "open-dialog", dialog: "notifications" },
    },
    {
      id: "app:copy-page-link",
      label: t("Copy link to this page"),
      description: t("Share exactly what you're looking at, filters included"),
      group: "navigation",
      icon: LinkIcon,
      keywords: ["copy", "link", "url", "share", "page"],
      intent: { type: "copy-link", href: context.currentHref },
    },
    {
      id: "app:toggle-sidebar",
      label: t("Toggle sidebar"),
      description: t("Show or hide the navigation sidebar"),
      group: "navigation",
      icon: PanelLeftIcon,
      keywords: ["sidebar", "collapse", "expand", "navigation", "hide", "show"],
      intent: { type: "toggle-sidebar" },
      shortcut: [mod, "B"],
    },
    {
      id: "app:shortcuts",
      label: t("Keyboard shortcuts"),
      description: t("Every shortcut that works across the app"),
      group: "navigation",
      icon: KeyboardIcon,
      keywords: ["keyboard", "shortcuts", "hotkeys", "keys", "help"],
      intent: { type: "open-dialog", dialog: "shortcuts" },
      shortcut: [mod, "/"],
    },
  );

  if (context.unreadNotifications > 0) {
    commands.push({
      id: "app:notifications-read-all",
      label: t("Mark all notifications as read"),
      description: t("Clear the unread count without opening each one"),
      group: "notifications",
      icon: CheckCheckIcon,
      keywords: ["notifications", "read", "clear", "dismiss", "all"],
      intent: { type: "mark-all-notifications-read" },
    });
  }

  const themes: { theme: PaletteTheme; label: string; icon: PaletteIcon }[] = [
    { theme: "light", label: t("Use light theme"), icon: SunIcon },
    { theme: "dark", label: t("Use dark theme"), icon: MoonIcon },
    { theme: "system", label: t("Match system theme"), icon: MonitorIcon },
  ];
  for (const option of themes) {
    commands.push({
      id: `app:theme-${option.theme}`,
      label: option.label,
      description: t("Change how Trenova looks on this device"),
      group: "appearance",
      icon: option.icon,
      keywords: ["theme", "appearance", "mode", option.theme],
      intent: { type: "set-theme", theme: option.theme },
      checked: context.theme === option.theme,
    });
  }

  commands.push(
    {
      id: "app:profile",
      label: t("Your profile"),
      description: t("Your details, sessions and security"),
      group: "account",
      icon: UserRoundIcon,
      keywords: ["profile", "account", "me", "user"],
      intent: { type: "navigate", href: "/profile" },
    },
    {
      id: "app:settings",
      label: t("Settings"),
      description: t("Preferences, time zone and language"),
      group: "account",
      icon: SettingsIcon,
      keywords: ["settings", "preferences", "timezone", "language", "account"],
      intent: { type: "open-dialog", dialog: "settings" },
      shortcut: [mod, "Shift", "S"],
    },
    {
      id: "app:sign-out",
      label: t("Log out"),
      description: t("End this session on this device"),
      group: "account",
      icon: LogOutIcon,
      keywords: ["log out", "logout", "sign out", "exit"],
      intent: { type: "sign-out" },
      destructive: true,
    },
  );

  return commands;
}
