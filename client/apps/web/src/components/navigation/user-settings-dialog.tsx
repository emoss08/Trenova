import { stripAppTitle } from "@/components/assistant/page-context";
import { useSignOut } from "@/hooks/use-sign-out";
import { useAppDialogsStore, type SettingsSection } from "@/stores/app-dialogs-store";
import { useMotionPreferenceStore } from "@/stores/motion-preference-store";
import { Button } from "@trenova/shared/components/ui/button";
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogTitle,
} from "@trenova/shared/components/ui/dialog";
import { Kbd, KbdGroup } from "@trenova/shared/components/ui/kbd";
import { Toaster } from "@trenova/shared/components/ui/toaster";
import {
  BellOff01Icon,
  LogOut01Icon,
  Settings01Icon,
  XCloseIcon,
} from "@trenova/shared/components/icons";
import { Badge } from "@trenova/shared/components/ui/badge";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { useQuery } from "@tanstack/react-query";
import { useNow } from "@trenova/shared/hooks/use-now";
import { useCallback, useEffect, useRef, useState } from "react";
import { queries } from "@/lib/queries";
import { NotificationsSection } from "./user-settings/notifications-section";
import { PreferencesSection } from "./user-settings/preferences-section";
import { ProfileSection } from "./user-settings/profile-section";
import { SecuritySection } from "./user-settings/security-section";
import {
  SETTINGS_PREVIEW_TOASTER_ID,
  SETTINGS_TOASTER_ID,
  SettingsFeedbackProvider,
  useSettingsFeedback,
} from "./user-settings/settings-feedback";
import { SettingsHero } from "./user-settings/settings-hero";
import { SETTINGS_SECTION_META } from "./user-settings/settings-index";
import {
  SettingsSection as Section,
  settingsRowId,
  settingsSectionId,
} from "./user-settings/settings-layout";
import { SettingsSearch } from "./user-settings/settings-search";
import { useProfilePicture } from "./user-settings/use-profile-picture";

type UserSettingsDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
};

const FLASH_MS = 1600;
const SPY_OFFSET_PX = 160;

function useIsPaused(): boolean {
  const now = useNow("minute");
  const { data } = useQuery({ ...queries.user.myNotificationPreferences(), staleTime: 30_000 });
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

function SettingsShell({ pageTitle }: { pageTitle: string }) {
  const t = useT();
  const signOut = useSignOut();
  const mainRef = useRef<HTMLDivElement>(null);
  const [current, setCurrent] = useState<SettingsSection>("profile");
  const requestedSection = useAppDialogsStore((state) => state.settingsSection);
  const clearSettingsSection = useAppDialogsStore((state) => state.clearSettingsSection);
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

  useEffect(() => {
    if (!requestedSection) return;
    const frame = window.requestAnimationFrame(() => {
      jump(requestedSection);
      clearSettingsSection();
    });
    return () => window.cancelAnimationFrame(frame);
  }, [clearSettingsSection, jump, requestedSection]);

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
    <div className="bg-background text-foreground grid h-dvh grid-rows-[52px_minmax(0,1fr)]">
      <header className="border-border-subtle flex items-center gap-2.5 border-b px-4">
        <span
          aria-hidden
          className="bg-ink text-ink-foreground grid size-6 place-items-center rounded-md"
        >
          <Settings01Icon className="size-3.5" />
        </span>
        <div className="flex min-w-0 items-baseline gap-1.5 text-sm">
          <DialogTitle className="text-sm font-medium">{t("Settings")}</DialogTitle>
          {pageTitle ? (
            <span className="text-foreground-subtle truncate text-xs">
              {t("over {0}", pageTitle)}
            </span>
          ) : null}
        </div>
        <span className="flex-1" />
        <SavedStatus />
        <DialogClose
          render={
            <Button
              variant="ghost"
              size="icon-sm"
              aria-label={t("Close settings")}
              title={t("Close · Esc")}
            />
          }
        >
          <XCloseIcon className="size-4" />
        </DialogClose>
      </header>

      <div className="grid min-h-0 grid-cols-1 min-[900px]:grid-cols-[248px_minmax(0,1fr)]">
        <nav
          aria-label={t("Settings sections")}
          className="border-border-subtle hidden min-h-0 flex-col gap-0.5 overflow-y-auto border-r px-3 py-4 min-[900px]:flex"
        >
          <SettingsSearch onJump={(result) => jump(result.section, result.row)} />
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
                onClick={() => jump(meta.id)}
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

        <div
          ref={mainRef}
          onScroll={onScroll}
          tabIndex={-1}
          className="min-h-0 overflow-y-auto scroll-smooth outline-none"
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
      </div>
      {picture.element}
    </div>
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

/**
 * Settings, as a whole screen over the page the person was on. The page stays mounted
 * underneath; closing returns to it exactly as it was.
 */
export function UserSettingsDialog({ open, onOpenChange }: UserSettingsDialogProps) {
  const t = useT();
  const pageTitle = open ? stripAppTitle(document.title) : "";

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent
        size="fullscreen"
        showCloseButton={false}
        aria-label={t("Settings")}
        className="block"
        initialFocus={() =>
          document.querySelector<HTMLElement>("[data-testid=user-settings-scroller]")
        }
      >
        <SettingsFeedbackProvider>
          <SettingsShell pageTitle={pageTitle} />
        </SettingsFeedbackProvider>
        <Toaster id={SETTINGS_TOASTER_ID} position="bottom-center" />
        <Toaster id={SETTINGS_PREVIEW_TOASTER_ID} position="top-right" offset={64} />
      </DialogContent>
    </Dialog>
  );
}
