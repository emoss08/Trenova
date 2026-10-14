import { FieldWrapper } from "@/components/fields/field-components";
import { SegmentedField } from "@/components/fields/segmented-field";
import { SelectField } from "@/components/fields/select-field";
import { SwitchField } from "@/components/fields/switch-field";
import { ToggleChipsField } from "@/components/fields/toggle-chips-field";
import {
  buildNotificationCatalog,
  type NotificationCatalogCategory,
} from "@/components/notification-center/notification-registry";
import { user as userQueries } from "@/lib/queries/user";
import {
  formatClockValue,
  halfHourValues,
  PAUSE_CHOICES,
  pauseUntil,
  WEEKDAYS,
  type PauseChoice,
} from "@/lib/user-settings";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { Switch } from "@trenova/shared/components/ui/switch";
import { Bell01Icon, ChevronDownIcon, Lock01Icon } from "@trenova/shared/components/icons";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixInUserTimezone, resolveUserTimezone } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import { notificationPreferencesService } from "@trenova/shared/services/notification-preferences";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import type {
  DeliveryMode,
  NotificationPreferences,
} from "@trenova/shared/types/notification-preferences";
import type { TimeFormatType } from "@trenova/shared/types/user";
import { useMemo, useRef, useState, useSyncExternalStore } from "react";
import { useForm, useWatch, type UseFormReturn } from "react-hook-form";
import { toast } from "sonner";
import { SETTINGS_PREVIEW_TOASTER_ID, useSettingsFeedback } from "./settings-feedback";
import { SettingsCard, SettingsRow } from "./settings-layout";
import { useAutosaveForm } from "./use-autosave-form";

type NotifyValues = {
  desktopMode: DeliveryMode;
  emailMode: DeliveryMode;
  emailDigest: boolean;
  sound: boolean;
  quietHoursEnabled: boolean;
  quietHoursStart: string;
  quietHoursEnd: string;
  quietHoursDays: string[];
  quietHoursAllowUrgent: boolean;
  pause: PauseChoice;
  enabled: boolean[];
};

type CatalogEntry = { key: string; required: boolean };

function flatCatalog(catalog: NotificationCatalogCategory[]): CatalogEntry[] {
  return catalog.flatMap((category) =>
    category.types.map((type) => ({ key: type.key, required: type.requiredReason !== null })),
  );
}

function toValues(
  prefs: NotificationPreferences,
  entries: CatalogEntry[],
  now: number,
): NotifyValues {
  const disabled = new Set(prefs.disabledEventTypes);
  return {
    desktopMode: prefs.desktopMode,
    emailMode: prefs.emailMode,
    emailDigest: prefs.emailDigest,
    sound: prefs.sound,
    quietHoursEnabled: prefs.quietHoursEnabled,
    quietHoursStart: prefs.quietHoursStart,
    quietHoursEnd: prefs.quietHoursEnd,
    quietHoursDays: prefs.quietHoursDays.map(String),
    quietHoursAllowUrgent: prefs.quietHoursAllowUrgent,
    pause: prefs.pausedUntil && prefs.pausedUntil > now ? "current" : "off",
    enabled: entries.map((entry) => entry.required || !disabled.has(entry.key)),
  };
}

type DesktopPermission = NotificationPermission | "unsupported";

function readPermission(): DesktopPermission {
  return typeof window !== "undefined" && "Notification" in window
    ? window.Notification.permission
    : "unsupported";
}

const permissionListeners = new Set<() => void>();

function subscribePermission(listener: () => void) {
  permissionListeners.add(listener);
  return () => {
    permissionListeners.delete(listener);
  };
}

async function requestPermission(): Promise<DesktopPermission> {
  if (readPermission() === "unsupported") return "unsupported";
  const result = await window.Notification.requestPermission();
  permissionListeners.forEach((listener) => listener());
  return result;
}

function useDesktopPermission(): DesktopPermission {
  return useSyncExternalStore(subscribePermission, readPermission, () => "unsupported");
}

function DeliveryCard({ form, email }: { form: UseFormReturn<NotifyValues>; email: string }) {
  const t = useT();
  const feedback = useSettingsFeedback();
  const permission = useDesktopPermission();
  const [desktopMode, emailMode] = useWatch({
    control: form.control,
    name: ["desktopMode", "emailMode"],
  });
  const modes = [
    { value: "off" as const, label: t("Off") },
    { value: "urgent" as const, label: t("Urgent only") },
    { value: "all" as const, label: t("Everything") },
  ];

  const sendTest = () => {
    const title = t("Test notification");
    const body = t("This is how a notification from Trenova reaches you.");
    toast.custom(
      () => (
        <div className="bg-raised ui-lift-float flex w-90 max-w-[calc(100vw-40px)] gap-3 rounded-lg border px-3.5 py-3">
          <span className="bg-muted grid size-8 shrink-0 place-items-center rounded-md">
            <Bell01Icon className="text-muted-foreground size-3.5" />
          </span>
          <span className="flex min-w-0 flex-col gap-0.5">
            <span className="text-foreground-subtle text-2xs">{t("Trenova · now")}</span>
            <span className="text-xs font-medium">{title}</span>
            <span className="text-muted-foreground text-xs">{body}</span>
          </span>
        </div>
      ),
      { toasterId: SETTINGS_PREVIEW_TOASTER_ID, duration: 4600 },
    );
    if (permission === "granted") {
      new window.Notification(title, { body, tag: "trenova-test" });
    }
  };

  const desktopDescription =
    desktopMode === "off" ? (
      t("Nothing pops up outside Trenova.")
    ) : permission === "granted" ? (
      <span className="text-2xs text-foreground/70">
        {t("Allowed in this browser.")}{" "}
        <Button type="button" variant="link" size="bare" className="text-2xs" onClick={sendTest}>
          {t("Send a test")}
        </Button>
      </span>
    ) : permission === "denied" ? (
      t("This browser blocks Trenova's notifications. Allow them in the site settings.")
    ) : permission === "unsupported" ? (
      t("This browser can't show desktop notifications.")
    ) : (
      <span className="text-2xs text-foreground/70">
        {t("This browser hasn't allowed Trenova yet.")}{" "}
        <Button
          type="button"
          variant="link"
          size="bare"
          className="text-2xs"
          onClick={() =>
            void requestPermission().then((result) => {
              if (result === "granted") feedback.message(t("Desktop notifications allowed"));
            })
          }
        >
          {t("Allow notifications")}
        </Button>
      </span>
    );

  return (
    <SettingsCard
      title={t("Delivery")}
      description={t("Everything lands in the bell. Choose what also reaches you outside Trenova.")}
    >
      <SettingsRow id="desktop">
        <SegmentedField
          control={form.control}
          name="desktopMode"
          layout="inline"
          label={t("Desktop")}
          description={desktopDescription}
          options={modes}
        />
      </SettingsRow>
      <SettingsRow id="email">
        <SegmentedField
          control={form.control}
          name="emailMode"
          layout="inline"
          label={t("Email")}
          description={emailMode === "off" ? t("No notification emails.") : t("Sent to {0}", email)}
          options={modes}
        />
      </SettingsRow>
      {emailMode === "all" ? (
        <SettingsRow sub id="digest">
          <SwitchField
            control={form.control}
            name="emailDigest"
            label={t("Bundle into a daily digest")}
            description={t(
              "Non-urgent emails arrive together at 8:00 AM. Urgent ones still come right away.",
            )}
            className="p-0"
          />
        </SettingsRow>
      ) : null}
      <SettingsRow id="sound">
        <SwitchField
          control={form.control}
          name="sound"
          label={t("Sound")}
          description={t("A soft chime with desktop notifications.")}
          className="p-0"
        />
      </SettingsRow>
    </SettingsCard>
  );
}

function QuietHoursCard({
  form,
  timeFormat,
}: {
  form: UseFormReturn<NotifyValues>;
  timeFormat: TimeFormatType;
}) {
  const t = useT();
  const enabled = useWatch({ control: form.control, name: "quietHoursEnabled" });
  const hours = useMemo(
    () => halfHourValues().map((value) => ({ value, label: formatClockValue(value, timeFormat) })),
    [timeFormat],
  );
  const days = [t("Sun"), t("Mon"), t("Tue"), t("Wed"), t("Thu"), t("Fri"), t("Sat")];

  return (
    <SettingsCard>
      <SettingsRow id="quiet">
        <SwitchField
          control={form.control}
          name="quietHoursEnabled"
          label={t("Quiet hours")}
          description={t(
            "Hold desktop notifications and emails during these hours. They wait in the bell.",
          )}
          className="p-0"
        />
      </SettingsRow>
      {enabled ? (
        <>
          <SettingsRow sub>
            <FieldWrapper layout="inline" label={t("Hours")}>
              <div className="grid grid-cols-[minmax(0,1fr)_auto_minmax(0,1fr)] items-center gap-2">
                <SelectField
                  control={form.control}
                  name="quietHoursStart"
                  options={hours}
                  placeholder={t("From")}
                />
                <span className="text-foreground-subtle text-xs">{t("to")}</span>
                <SelectField
                  control={form.control}
                  name="quietHoursEnd"
                  options={hours}
                  placeholder={t("To")}
                />
              </div>
            </FieldWrapper>
          </SettingsRow>
          <SettingsRow sub>
            <FieldWrapper layout="inline" label={t("Days")}>
              <ToggleChipsField
                control={form.control}
                name="quietHoursDays"
                label={t("Quiet days")}
                checkedTone="neutral"
                options={WEEKDAYS.map((day, index) => ({ value: day, label: days[index] }))}
                className="pt-1"
              />
            </FieldWrapper>
          </SettingsRow>
          <SettingsRow sub>
            <SwitchField
              control={form.control}
              name="quietHoursAllowUrgent"
              label={t("Let urgent ones through")}
              description={t("Waterfalls running dry, blocked carriers, failed EDI and the like.")}
              className="p-0"
            />
          </SettingsRow>
        </>
      ) : null}
    </SettingsCard>
  );
}

function PauseCard({
  form,
  pausedUntil,
  timezone,
  timeFormat,
}: {
  form: UseFormReturn<NotifyValues>;
  pausedUntil: number | null;
  timezone: string;
  timeFormat: TimeFormatType;
}) {
  const t = useT();
  const pause = useWatch({ control: form.control, name: "pause" });
  const labels: Record<(typeof PAUSE_CHOICES)[number], string> = {
    off: t("Not paused"),
    "1h": t("For 1 hour"),
    tomorrow: t("Until 8:00 AM tomorrow"),
    monday: t("Until Monday"),
  };
  const options = [
    ...PAUSE_CHOICES.map((choice) => ({ value: choice, label: labels[choice] })),
    ...(pause === "current" && pausedUntil
      ? [
          {
            value: "current",
            label: t(
              "Until {0}",
              formatUnixInUserTimezone(pausedUntil, {
                timezone,
                timeFormat,
                weekday: "short",
                hour: "numeric",
                minute: "2-digit",
              }),
            ),
          },
        ]
      : []),
  ];

  return (
    <SettingsCard>
      <SettingsRow id="pause">
        <SelectField
          control={form.control}
          name="pause"
          layout="inline"
          label={t("Pause notifications")}
          description={t("Stop desktop notifications and emails for a while.")}
          options={options}
        />
      </SettingsRow>
    </SettingsCard>
  );
}

function CategoryRow({
  category,
  offset,
  form,
  open,
  onToggle,
  onChanged,
}: {
  category: NotificationCatalogCategory;
  offset: number;
  form: UseFormReturn<NotifyValues>;
  open: boolean;
  onToggle: () => void;
  onChanged: () => void;
}) {
  const t = useT();
  const enabled = useWatch({ control: form.control, name: "enabled" });
  const Icon = category.icon;
  const indices = category.types.map((_, index) => offset + index);
  const on = indices.filter((index) => enabled[index]).length;
  const total = category.types.length;
  const categoryLabel = t(category.label);

  const setAll = (value: boolean) => {
    category.types.forEach((type, index) => {
      if (type.requiredReason === null) {
        form.setValue(`enabled.${offset + index}`, value, { shouldDirty: true });
      }
    });
    onChanged();
  };

  return (
    <div className="flex flex-col">
      <div className="flex items-center gap-2.5 px-4.5 py-3">
        <button
          type="button"
          onClick={total > 1 ? onToggle : undefined}
          aria-expanded={total > 1 ? open : undefined}
          className="ui-focus-ring flex min-w-0 flex-1 items-center gap-3 rounded-md text-left"
        >
          <span
            className={cn(
              "grid size-7.5 shrink-0 place-items-center rounded-md",
              category.tileClass,
            )}
          >
            <Icon className={cn("size-3.5", category.iconClass)} />
          </span>
          <span className="flex min-w-0 flex-col gap-px">
            <span className="text-sm font-medium">{categoryLabel}</span>
            <span className="text-foreground-subtle truncate text-xs">
              {t(category.description)}
            </span>
          </span>
        </button>
        {total > 1 ? (
          <Button type="button" variant="quiet" size="xs" onClick={onToggle} aria-expanded={open}>
            {on === total ? t("All {0}", total) : t("{0} of {1}", on, total)}
            <ChevronDownIcon
              className={cn("size-3 transition-transform duration-200", open && "rotate-180")}
            />
          </Button>
        ) : null}
        <Switch
          checked={on > 0}
          aria-label={t("Notifications about {0}", categoryLabel)}
          onCheckedChange={() => setAll(!(on > 0))}
        />
      </div>
      {open && total > 1 ? (
        <div className="animate-in fade-in-0 slide-in-from-top-1 flex flex-col pr-2 pb-2.5 pl-2 duration-200 min-[620px]:pl-15">
          {category.types.map((type, index) => (
            <SwitchField
              key={type.key}
              control={form.control}
              name={`enabled.${offset + index}`}
              label={t(type.label)}
              disabled={type.requiredReason !== null}
              description={type.requiredReason ?? undefined}
              className="hover:bg-surface-hover px-2.5 py-2 [&_label]:font-normal"
              badges={
                <>
                  {type.urgent ? <Badge variant="danger">{t("Urgent")}</Badge> : null}
                  {type.requiredReason !== null ? (
                    <Badge variant="neutral">
                      <Lock01Icon />
                      {t("Required")}
                    </Badge>
                  ) : null}
                </>
              }
            />
          ))}
        </div>
      ) : null}
    </div>
  );
}

function TypesCard({
  form,
  catalog,
  onChanged,
}: {
  form: UseFormReturn<NotifyValues>;
  catalog: NotificationCatalogCategory[];
  onChanged: () => void;
}) {
  const t = useT();
  const [open, setOpen] = useState<string | null>(null);
  const offsets = useMemo(
    () =>
      catalog.reduce<number[]>((starts, _category, index) => {
        starts.push(index === 0 ? 0 : starts[index - 1] + catalog[index - 1].types.length);
        return starts;
      }, []),
    [catalog],
  );

  return (
    <SettingsCard
      id="types"
      title={t("What you hear about")}
      description={t("Turn off anything you don't need. A few are required by your organization.")}
    >
      {catalog.map((category, index) => {
        return (
          <CategoryRow
            key={category.name}
            category={category}
            offset={offsets[index]}
            form={form}
            open={open === category.name}
            onToggle={() =>
              setOpen((current) => (current === category.name ? null : category.name))
            }
            onChanged={onChanged}
          />
        );
      })}
    </SettingsCard>
  );
}

function NotificationsForm({ preferences }: { preferences: NotificationPreferences }) {
  const user = useAuthStore((state) => state.user);
  const queryClient = useQueryClient();
  const catalog = useMemo(() => buildNotificationCatalog(), []);
  const entries = useMemo(() => flatCatalog(catalog), [catalog]);
  const [pausedUntil, setPausedUntil] = useState<number | null>(preferences.pausedUntil);
  const pausedUntilRef = useRef<number | null>(preferences.pausedUntil);
  const timezone = resolveUserTimezone(user?.timezone);
  const timeFormat = user?.timeFormat ?? "12-hour";
  const [defaultValues] = useState(() =>
    toValues(preferences, entries, Math.floor(Date.now() / 1000)),
  );
  const form = useForm<NotifyValues>({ defaultValues });

  const { commit } = useAutosaveForm({
    form,
    resourceName: "Notification preferences",
    save: async (next) => {
      const pausedUntil =
        next.pause === "current"
          ? pausedUntilRef.current
          : pauseUntil(next.pause, new Date(), timezone);
      const saved = await notificationPreferencesService.update({
        desktopMode: next.desktopMode,
        emailMode: next.emailMode,
        emailDigest: next.emailDigest,
        sound: next.sound,
        quietHoursEnabled: next.quietHoursEnabled,
        quietHoursStart: next.quietHoursStart,
        quietHoursEnd: next.quietHoursEnd,
        quietHoursDays: next.quietHoursDays.map(Number).sort((a, b) => a - b),
        quietHoursAllowUrgent: next.quietHoursAllowUrgent,
        pausedUntil,
        disabledEventTypes: entries
          .filter((entry, index) => !entry.required && !next.enabled[index])
          .map((entry) => entry.key),
      });
      pausedUntilRef.current = saved.pausedUntil;
      setPausedUntil(saved.pausedUntil);
      queryClient.setQueryData(userQueries.myNotificationPreferences().queryKey, saved);
    },
  });

  return (
    <>
      <DeliveryCard form={form} email={user?.emailAddress ?? ""} />
      <QuietHoursCard form={form} timeFormat={timeFormat} />
      <PauseCard
        form={form}
        pausedUntil={pausedUntil}
        timezone={timezone}
        timeFormat={timeFormat}
      />
      <TypesCard form={form} catalog={catalog} onChanged={commit} />
    </>
  );
}

function NotificationsSkeleton() {
  return (
    <div className="flex flex-col gap-3.5" aria-hidden>
      {[3, 1, 1, 6].map((rows, card) => (
        <div key={card} className="bg-card flex flex-col gap-3 rounded-lg border px-4.5 py-3.5">
          {Array.from({ length: rows }, (_, row) => (
            <Skeleton key={row} className="h-7 w-full rounded-md" />
          ))}
        </div>
      ))}
    </div>
  );
}

export function NotificationsSection() {
  const t = useT();
  const { data, isError, refetch } = useQuery(userQueries.myNotificationPreferences());

  if (isError) {
    return (
      <div className="bg-card flex items-center justify-between gap-3 rounded-lg border px-4.5 py-3.5 text-sm">
        <span>{t("Notification settings could not be loaded.")}</span>
        <Button type="button" variant="outline" size="sm" onClick={() => void refetch()}>
          {t("Try again")}
        </Button>
      </div>
    );
  }

  if (!data) {
    return <NotificationsSkeleton />;
  }

  return <NotificationsForm preferences={data} />;
}
