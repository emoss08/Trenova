import { buildNotificationCatalog } from "@/components/notification-center/notification-registry";
import type { SettingsSection } from "@/lib/settings-path";
import {
  Bell01Icon,
  type IconComponent,
  ShieldTickIcon,
  Sliders01Icon,
  User01Icon,
} from "@trenova/shared/components/icons";

export type SettingsSectionMeta = {
  id: SettingsSection;
  /** English source; translate at render. */
  label: string;
  /** English source; translate at render. */
  description: string;
  icon: IconComponent;
};

export const SETTINGS_SECTION_META: readonly SettingsSectionMeta[] = [
  {
    id: "profile",
    label: "Profile",
    description: "Who you are and where you are.",
    icon: User01Icon,
  },
  {
    id: "preferences",
    label: "Preferences",
    description: "How Trenova looks and keeps time for you.",
    icon: Sliders01Icon,
  },
  {
    id: "notifications",
    label: "Notifications",
    description:
      "Choose what reaches you, where, and when. Every notification Trenova sends can be turned off here, except a few your organization requires.",
    icon: Bell01Icon,
  },
  {
    id: "security",
    label: "Security",
    description: "Your password, second factor and every place you're signed in.",
    icon: ShieldTickIcon,
  },
];

export type SettingsIndexEntry = {
  /** English source; translate at render and when matching. */
  label: string;
  section: SettingsSection;
  /** The row to scroll to and flash. */
  row: string;
  /** The notification category a type belongs to, English source. */
  detail?: string;
};

const FIXED_ENTRIES: readonly SettingsIndexEntry[] = [
  { label: "Photo", section: "profile", row: "photo" },
  { label: "Avatar", section: "profile", row: "avatar" },
  { label: "Job title", section: "profile", row: "job-title" },
  { label: "Phone", section: "profile", row: "phone" },
  { label: "Email address", section: "profile", row: "email" },
  { label: "Availability · focus · away", section: "profile", row: "availability" },
  { label: "Hand off while away", section: "profile", row: "availability" },
  { label: "Theme · dark mode", section: "preferences", row: "theme" },
  { label: "Language", section: "preferences", row: "language" },
  { label: "Time zone", section: "preferences", row: "timezone" },
  { label: "Time format · 24-hour", section: "preferences", row: "time-format" },
  { label: "Sidebar", section: "preferences", row: "sidebar" },
  { label: "Reduce motion", section: "preferences", row: "motion" },
  { label: "Assistant · Nova · AI", section: "preferences", row: "assistant" },
  { label: "Desktop notifications", section: "notifications", row: "desktop" },
  { label: "Email notifications · digest", section: "notifications", row: "email" },
  { label: "Sound", section: "notifications", row: "sound" },
  { label: "Quiet hours", section: "notifications", row: "quiet" },
  { label: "Pause notifications", section: "notifications", row: "pause" },
  { label: "What you hear about", section: "notifications", row: "types" },
  { label: "Password", section: "security", row: "password" },
  { label: "Two-factor authentication", section: "security", row: "two-factor" },
  { label: "Recovery codes", section: "security", row: "two-factor" },
  { label: "Sessions · sign out", section: "security", row: "sessions" },
  { label: "Sign-in activity", section: "security", row: "sign-ins" },
];

/** Every setting the search finds: the fixed rows and every kind of notification. */
export function buildSettingsIndex(): SettingsIndexEntry[] {
  const notificationTypes = buildNotificationCatalog().flatMap((category) =>
    category.types.map((type) => ({
      label: type.label,
      section: "notifications" as const,
      row: "types",
      detail: category.label,
    })),
  );
  return [...FIXED_ENTRIES, ...notificationTypes];
}

export type SettingsSearchResult = SettingsIndexEntry & { text: string; detailText?: string };

/** The entries whose label or category contains the query, read in the language on screen. */
export function searchSettings(
  entries: readonly SettingsIndexEntry[],
  query: string,
  translateText: (source: string) => string,
  limit = 8,
): SettingsSearchResult[] {
  const needle = query.trim().toLocaleLowerCase();
  if (!needle) return [];

  const results: SettingsSearchResult[] = [];
  for (const entry of entries) {
    const text = translateText(entry.label);
    const detailText = entry.detail ? translateText(entry.detail) : undefined;
    if (
      text.toLocaleLowerCase().includes(needle) ||
      detailText?.toLocaleLowerCase().includes(needle)
    ) {
      results.push({ ...entry, text, detailText });
      if (results.length >= limit) break;
    }
  }
  return results;
}
