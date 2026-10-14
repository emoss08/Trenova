import { PageLayout } from "@/components/navigation/sidebar-layout";
import { useSignOut } from "@/hooks/use-sign-out";
import type { RoutePrefetch } from "@/lib/route-prefetch";
import type { SettingsSection } from "@/lib/settings-path";
import { user, userOrganization } from "@/lib/queries/user";
import { useMotionPreferenceStore } from "@/stores/motion-preference-store";
import { useNavigationStore } from "@/stores/navigation-store";
import { Button } from "@trenova/shared/components/ui/button";
import { Kbd, KbdGroup } from "@trenova/shared/components/ui/kbd";
import { Toaster } from "@trenova/shared/components/ui/toaster";
import { BellOff01Icon, LogOut01Icon } from "@trenova/shared/components/icons";
import { Badge } from "@trenova/shared/components/ui/badge";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { useQuery } from "@tanstack/react-query";
import { useNow } from "@trenova/shared/hooks/use-now";
import { useCallback, useRef, useState } from "react";
import { NotificationsSection } from "./_components/notifications-section";
import { PreferencesSection } from "./_components/preferences-section";
import { ProfileSection } from "./_components/profile-section";
import { SecuritySection } from "./_components/security-section";
import {
  SETTINGS_PREVIEW_TOASTER_ID,
  SETTINGS_TOASTER_ID,
  SettingsFeedbackProvider,
  useSettingsFeedback,
} from "./_components/settings-feedback";
import { SettingsHero } from "./_components/settings-hero";
import { SETTINGS_SECTION_META } from "./_components/settings-index";
import {
  SettingsSection as Section,
  settingsRowId,
  settingsSectionId,
} from "./_components/settings-layout";
import { SettingsSearch } from "./_components/settings-search";
import { useProfilePicture } from "./_components/use-profile-picture";
import { useRequestedSection } from "./_components/use-requested-section";

const FLASH_MS = 1600;
const SPY_OFFSET_PX = 160;
const NOTIFICATION_PREFERENCES_STALE_MS = 30_000;

export const prefetch: RoutePrefetch = () => [
  { ...user.myNotificationPreferences(), staleTime: NOTIFICATION_PREFERENCES_STALE_MS },
  user.mySessions(),
  user.mySignIns(),
  userOrganization.all(),
];

function useIsPaused(): boolean {
  const now = useNow("minute");
  const { data } = useQuery({
    ...user.myNotificationPreferences(),
    staleTime: NOTIFICATION_PREFERENCES_STALE_MS,
  });
  return Boolean(data?.pausedUntil && data.pausedUntil > now);
}

function SavedStatus() {
  const t = useT();
  const { pulsing } = useSettingsFeedback();

  return (
    <span
      role="status"
      className={cn(
        "text-foreground-subtle flex h-7 items-center gap-1.5 rounded-full px-2.5 text-xs transition-colors duration-200",
        pulsing && "bg-success-subtle text-success-foreground",
      )}
    >
      <span aria-hidden className="bg-success size-1.5 rounded-full" />
      {pulsing ? t("Saved") : t("All changes saved")}
    </span>
  );
}

function flash(element: HTMLElement) {
  if (useMotionPreferenceStore.getState().reduceMotion) {
    return;
  }
  element.animate(
    [
      { backgroundColor: "var(--brand-subtle)" },
      { backgroundColor: "var(--brand-subtle)", offset: 0.3 },
      { backgroundColor: "transparent" },
    ],
    { duration: FLASH_MS, easing: "cubic-bezier(0.16, 1, 0.3, 1)" },
  );
}

function PausedBadge() {
  const t = useT();
  return (
    <Badge variant="warning">
      <BellOff01Icon />
      {t("Paused")}
    </Badge>
  );
}

function SettingsRail({
  current,
  paused,
  onJump,
}: {
  current: SettingsSection;
  paused: boolean;
  onJump: (section: SettingsSection, row?: string) => void;
}) {
  const t = useT();
  const signOut = useSignOut();
  const folded = useNavigationStore((state) => state.sidebarCollapsed);

  return (
    <aside
      aria-hidden={folded || undefined}
      inert={folded || undefined}
      className={cn(
        "border-border bg-canvas hidden shrink-0 flex-col overflow-hidden border-r transition-[width] duration-300 ease-settle md:flex",
        folded ? "w-0 border-r-0" : "w-62",
      )}
    >
      <div className="flex min-h-0 w-62 flex-1 flex-col">
        <div className="flex items-center gap-2 pt-3 pr-2 pb-1 pl-4.25">
          <p aria-hidden className="flex-1 text-base font-semibold tracking-tight">
            {t("Settings")}
          </p>
          <SavedStatus />
        </div>
        <nav
          aria-label={t("Settings sections")}
          className="flex min-h-0 flex-1 flex-col gap-0.5 overflow-y-auto px-3 pt-2 pb-4"
        >
          <SettingsSearch onJump={(result) => onJump(result.section, result.row)} />
          {SETTINGS_SECTION_META.map((meta) => {
            const Icon = meta.icon;
            const active = current === meta.id;
            return (
              <Button
                key={meta.id}
                type="button"
                variant="bare"
                size="bare"
                aria-current={active ? "true" : undefined}
                onClick={() => onJump(meta.id)}
                className={cn(
                  "text-muted-foreground hover:bg-surface-hover hover:text-foreground flex h-8.5 w-full items-center justify-start gap-2.5 rounded-md px-2.5 text-sm transition-colors",
                  active && "bg-surface-hover text-foreground font-medium",
                )}
              >
                <Icon className="size-3.75" />
                {t(meta.label)}
                {meta.id === "notifications" ? (
                  paused ? (
                    <span
                      className="bg-warning ml-auto size-1.75 rounded-full"
                      title={t("Paused")}
                      aria-label={t("Paused")}
                    />
                  ) : (
                    <Badge variant="brand" className="ml-auto h-4 px-1.5 text-2xs">
                      {t("New")}
                    </Badge>
                  )
                ) : null}
              </Button>
            );
          })}
          <div className="text-foreground-subtle mt-auto flex flex-col gap-2 px-2.5 pt-3 pb-1 text-xs">
            <span className="flex items-center gap-1.5">
              <KbdGroup>
                <Kbd>⌘</Kbd>
                <Kbd>⇧</Kbd>
                <Kbd>S</Kbd>
              </KbdGroup>
              {t("opens Settings anywhere")}
            </span>
            <Button
              type="button"
              variant="quiet"
              size="sm"
              className="hover:text-danger-foreground -ml-2 w-fit"
              onClick={() => void signOut()}
            >
              <LogOut01Icon className="size-3.5" />
              {t("Log out")}
            </Button>
          </div>
        </nav>
      </div>
    </aside>
  );
}

function SettingsWorkspace() {
  const t = useT();
  const mainRef = useRef<HTMLDivElement>(null);
  const [current, setCurrent] = useState<SettingsSection>("profile");
  const paused = useIsPaused();
  const picture = useProfilePicture();

  const jump = useCallback((section: SettingsSection, row?: string) => {
    const main = mainRef.current;
    if (!main) return;
    const target =
      (row ? document.getElementById(settingsRowId(row)) : null) ??
      document.getElementById(settingsSectionId(section));
    if (!target) return;

    const top =
      target.getBoundingClientRect().top - main.getBoundingClientRect().top + main.scrollTop;
    main.scrollTo({ top: Math.max(0, top - (row ? 90 : 12)) });
    if (row) flash(target);
  }, []);

  useRequestedSection(jump);

  const onScroll = () => {
    const main = mainRef.current;
    if (!main) return;
    let next: SettingsSection = SETTINGS_SECTION_META[0].id;
    for (const meta of SETTINGS_SECTION_META) {
      const element = document.getElementById(settingsSectionId(meta.id));
      if (element && element.offsetTop - main.scrollTop < SPY_OFFSET_PX) {
        next = meta.id;
      }
    }
    if (main.scrollTop + main.clientHeight >= main.scrollHeight - 4) {
      next = SETTINGS_SECTION_META[SETTINGS_SECTION_META.length - 1].id;
    }
    setCurrent(next);
  };

  return (
    <div className="flex min-h-0 flex-1">
      <SettingsRail current={current} paused={paused} onJump={jump} />
      <div
        ref={mainRef}
        onScroll={onScroll}
        tabIndex={-1}
        className="relative min-h-0 min-w-0 flex-1 overflow-y-auto scroll-smooth outline-none"
        data-testid="user-settings-scroller"
      >
        <div className="mx-auto max-w-200 px-4.5 pt-6 pb-25 min-[900px]:px-8 min-[900px]:pt-9 min-[900px]:pb-30">
          <SettingsHero picture={picture} onStatusClick={() => jump("profile", "availability")} />
          {SETTINGS_SECTION_META.map((meta) => (
            <Section
              key={meta.id}
              id={meta.id}
              title={t(meta.label)}
              description={t(meta.description)}
              badge={meta.id === "notifications" && paused ? <PausedBadge /> : undefined}
            >
              {meta.id === "profile" ? <ProfileSection picture={picture} /> : null}
              {meta.id === "preferences" ? <PreferencesSection /> : null}
              {meta.id === "notifications" ? <NotificationsSection /> : null}
              {meta.id === "security" ? <SecuritySection /> : null}
            </Section>
          ))}
        </div>
      </div>
      {picture.element}
    </div>
  );
}

/**
 * The person's own Settings, as a room: the module rail stays, and the sections take the
 * panel's place and its fold. Each change saves as it is made.
 */
export function SettingsPage() {
  const t = useT();

  return (
    <PageLayout
      fill
      className="gap-y-0 p-0"
      pageHeaderProps={{
        title: t("Settings"),
        description: t(
          "Your profile, preferences, notifications and security, each saved as you change it",
        ),
      }}
    >
      <SettingsFeedbackProvider>
        <SettingsWorkspace />
      </SettingsFeedbackProvider>
      <Toaster id={SETTINGS_TOASTER_ID} position="bottom-center" />
      <Toaster id={SETTINGS_PREVIEW_TOASTER_ID} position="top-right" offset={64} />
    </PageLayout>
  );
}
