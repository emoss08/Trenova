import { FieldWrapper } from "@/components/fields/field-components";
import { SegmentedField } from "@/components/fields/segmented-field";
import { SelectField } from "@/components/fields/select-field";
import { SwitchField } from "@/components/fields/switch-field";
import { TimezoneField } from "@/components/fields/timezone-field";
import { applyLocale } from "@/components/navigation/language-submenu";
import { usePermission } from "@/hooks/use-permission";
import { useAssistantStore } from "@/stores/assistant-store";
import { useMotionPreferenceStore } from "@/stores/motion-preference-store";
import { useNavigationStore } from "@/stores/navigation-store";
import { apiService } from "@/services/api";
import { useTheme } from "@trenova/shared/components/theme-provider";
import { Button } from "@trenova/shared/components/ui/button";
import { useNow } from "@trenova/shared/hooks/use-now";
import {
  DEFAULT_LOCALE,
  LOCALE_NAMES,
  LOCALES,
  type Locale,
} from "@trenova/shared/i18n/generated/locales";
import { useRichT } from "@trenova/shared/i18n/rich";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixTime } from "@trenova/shared/lib/date";
import { timezoneGroupedChoices } from "@/lib/choices";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { Operation, Resource } from "@trenova/shared/types/permission";
import type { TimeFormatType, User } from "@trenova/shared/types/user";
import { useForm, useWatch } from "react-hook-form";
import { SettingsCard, SettingsRow } from "./settings-layout";
import { ThemePicker, type ThemeChoice } from "./theme-picker";
import { useAutosaveForm } from "./use-autosave-form";

type RegionalValues = {
  locale: Locale;
  timezone: string;
  timeFormat: TimeFormatType;
};

type DisplayValues = {
  theme: ThemeChoice;
  sidebar: "open" | "folded";
  reduceMotion: boolean;
};

type AssistantValues = {
  showAssistant: boolean;
};

const SAMPLE_PICKUP = Date.UTC(2026, 0, 6, 14, 30) / 1000;

function isLocale(value: string | undefined): value is Locale {
  return (LOCALES as readonly string[]).includes(value ?? "");
}

const KNOWN_ZONES = new Set(
  timezoneGroupedChoices.flatMap((group) => group.options.map((option) => String(option.value))),
);

function deviceTimezone(): string {
  return Intl.DateTimeFormat().resolvedOptions().timeZone;
}

function regionalValues(user: User | null): RegionalValues {
  return {
    locale: isLocale(user?.locale) ? user.locale : DEFAULT_LOCALE,
    timezone: user?.timezone || deviceTimezone(),
    timeFormat: user?.timeFormat ?? "12-hour",
  };
}

function TimezoneDescription({
  timezone,
  timeFormat,
  onUseDevice,
}: {
  timezone: string;
  timeFormat: TimeFormatType;
  onUseDevice: () => void;
}) {
  const t = useT();
  const rt = useRichT();
  const now = useNow("minute");
  const device = deviceTimezone();
  const offerDevice = device !== timezone && KNOWN_ZONES.has(device);

  return (
    <p className="text-2xs text-foreground/70 flex flex-wrap items-center gap-x-1.5 text-left">
      <span>
        {rt(
          "It's <b>{0}</b> for you.",
          { b: (children) => <b className="font-mono font-medium">{children}</b> },
          formatUnixTime(now, { timezone, timeFormat }),
        )}
      </span>
      {offerDevice ? (
        <Button type="button" variant="link" size="bare" className="text-2xs" onClick={onUseDevice}>
          {t("Use this device's")}
        </Button>
      ) : null}
    </p>
  );
}

function RegionalRows({ user }: { user: User | null }) {
  const t = useT();
  const rt = useRichT();
  const form = useForm<RegionalValues>({ defaultValues: regionalValues(user) });
  const [timezone, timeFormat] = useWatch({
    control: form.control,
    name: ["timezone", "timeFormat"],
  });

  const { commit } = useAutosaveForm({
    form,
    resourceName: "User Settings",
    save: async (next, previous) => {
      if (next.locale !== previous.locale) {
        await applyLocale(next.locale);
      }
      try {
        const updated = await apiService.userService.updateMySettings(next);
        useAuthStore.getState().setUser(updated);
      } catch (error) {
        if (next.locale !== previous.locale) {
          await applyLocale(previous.locale);
        }
        throw error;
      }
    },
  });

  return (
    <>
      <SettingsRow id="language">
        <SelectField
          control={form.control}
          name="locale"
          layout="inline"
          label={t("Language")}
          description={t("Menus, labels and the emails Trenova sends you.")}
          options={LOCALES.map((locale) => ({ value: locale, label: LOCALE_NAMES[locale] }))}
        />
      </SettingsRow>
      <SettingsRow id="timezone">
        <TimezoneField
          control={form.control}
          name="timezone"
          layout="inline"
          label={t("Time zone")}
          description={
            <TimezoneDescription
              timezone={timezone}
              timeFormat={timeFormat}
              onUseDevice={() => {
                form.setValue("timezone", deviceTimezone(), { shouldDirty: true });
                commit();
              }}
            />
          }
        />
      </SettingsRow>
      <SettingsRow id="time-format">
        <SegmentedField
          control={form.control}
          name="timeFormat"
          layout="inline"
          label={t("Time format")}
          description={rt(
            "Pickups show as <b>{0}</b>",
            { b: (children) => <b className="font-mono font-medium">{children}</b> },
            formatUnixTime(SAMPLE_PICKUP, { timezone: "UTC", timeFormat }),
          )}
          options={[
            { value: "12-hour", label: t("12-hour") },
            { value: "24-hour", label: t("24-hour") },
          ]}
        />
      </SettingsRow>
    </>
  );
}

function useDisplayRows() {
  const t = useT();
  const { theme, setTheme } = useTheme();
  const sidebarCollapsed = useNavigationStore((state) => state.sidebarCollapsed);
  const reduceMotion = useMotionPreferenceStore((state) => state.reduceMotion);
  const form = useForm<DisplayValues>({
    defaultValues: {
      theme: theme as ThemeChoice,
      sidebar: sidebarCollapsed ? "folded" : "open",
      reduceMotion,
    },
  });
  const themeValue = useWatch({ control: form.control, name: "theme" });

  const { commit } = useAutosaveForm({
    form,
    resourceName: "Display",
    save: async (next) => {
      setTheme(next.theme);
      useNavigationStore.getState().setSidebarCollapsed(next.sidebar === "folded");
      useMotionPreferenceStore.getState().setReduceMotion(next.reduceMotion);
    },
  });

  return {
    theme: (
      <SettingsRow id="theme">
        <FieldWrapper
          layout="inline"
          label={t("Theme")}
          description={t("Light, dark, or follow your system.")}
        >
          <ThemePicker
            value={themeValue}
            onChange={(next) => {
              form.setValue("theme", next, { shouldDirty: true });
              commit();
            }}
          />
        </FieldWrapper>
      </SettingsRow>
    ),
    rest: (
      <>
        <SettingsRow id="sidebar">
          <SegmentedField
            control={form.control}
            name="sidebar"
            layout="inline"
            label={t("Sidebar")}
            description={t("The module's pages beside the rail, or the rail alone.")}
            options={[
              { value: "open", label: t("Rail and pages") },
              { value: "folded", label: t("Rail only") },
            ]}
          />
        </SettingsRow>
        <SettingsRow id="motion">
          <SwitchField
            control={form.control}
            name="reduceMotion"
            label={t("Reduce motion")}
            description={t("Fewer slides, springs and other movement.")}
            className="p-0"
          />
        </SettingsRow>
      </>
    ),
  };
}

function AssistantRow({ user }: { user: User | null }) {
  const t = useT();
  const form = useForm<AssistantValues>({
    defaultValues: { showAssistant: user?.assistantHidden !== true },
  });

  useAutosaveForm({
    form,
    resourceName: "Assistant",
    save: async (next) => {
      const updated = await apiService.userService.updateMyAssistantVisibility({
        assistantHidden: !next.showAssistant,
      });
      if (updated.assistantHidden) {
        useAssistantStore.getState().closeWidget();
      }
      useAuthStore.getState().setUser(updated);
    },
  });

  return (
    <SettingsRow id="assistant">
      <SwitchField
        control={form.control}
        name="showAssistant"
        label={t("Show the assistant")}
        description={t(
          "The assistant on every page and its shortcut. Turned off, it stays off on every device you sign in from; the Desk is still there.",
        )}
        className="p-0"
      />
    </SettingsRow>
  );
}

export function PreferencesSection() {
  const user = useAuthStore((state) => state.user);
  const display = useDisplayRows();
  const { allowed: canUseAssistant } = usePermission(Resource.Assistant, Operation.Read);

  return (
    <SettingsCard>
      {display.theme}
      <RegionalRows user={user} />
      {display.rest}
      {canUseAssistant ? <AssistantRow user={user} /> : null}
    </SettingsCard>
  );
}
