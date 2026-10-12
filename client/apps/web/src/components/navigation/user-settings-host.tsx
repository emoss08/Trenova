import { useT } from "@trenova/shared/i18n/use-t";
import { Dialog, DialogContent, DialogTitle } from "@trenova/shared/components/ui/dialog";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import {
  SETTINGS_SECTIONS,
  useAppDialogOpen,
  useAppDialogsStore,
  type SettingsSection,
} from "@/stores/app-dialogs-store";
import { useHotkey } from "@tanstack/react-hotkeys";
import { lazy, Suspense, useCallback, useEffect, useState } from "react";
import { useLocation, useNavigate } from "react-router";

const SETTINGS_PARAM = "settings";

// The settings dialog carries the timezone/time-format choice tables, the
// avatar cropper and the react-hook-form field set — none of which belong in
// the chunk that renders the sidebar on every page.
const UserSettingsDialog = lazy(() =>
  import("@/components/navigation/user-settings-dialog").then((module) => ({
    default: module.UserSettingsDialog,
  })),
);

function isSettingsSection(value: string | null): value is SettingsSection {
  return value !== null && (SETTINGS_SECTIONS as readonly string[]).includes(value);
}

function UserSettingsDialogSkeleton({
  open,
  onOpenChange,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const t = useT();

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent size="fullscreen" showCloseButton={false} className="block">
        <div className="bg-background grid h-dvh grid-rows-[52px_minmax(0,1fr)]">
          <div className="border-border-subtle flex items-center gap-2.5 border-b px-4">
            <Skeleton className="size-6 rounded-md" />
            <DialogTitle className="text-sm font-medium">{t("Settings")}</DialogTitle>
          </div>
          <div className="grid min-h-0 grid-cols-1 min-[900px]:grid-cols-[248px_minmax(0,1fr)]">
            <div className="border-border-subtle hidden flex-col gap-2 border-r px-3 py-4 min-[900px]:flex">
              <Skeleton className="mb-2 h-8 w-full rounded-md" />
              {Array.from({ length: 4 }, (_, index) => (
                <Skeleton key={index} className="h-8 w-full rounded-md" />
              ))}
            </div>
            <div className="mx-auto w-full max-w-200 px-4.5 pt-9 min-[900px]:px-8">
              <div className="flex items-center gap-5 pb-9">
                <Skeleton className="size-16 rounded-full" />
                <div className="flex flex-1 flex-col gap-2">
                  <Skeleton className="h-7 w-72 rounded-md" />
                  <Skeleton className="h-4 w-56 rounded-md" />
                </div>
              </div>
              <Skeleton className="mb-2 h-5 w-32 rounded-md" />
              <Skeleton className="mb-4 h-4 w-64 rounded-md" />
              <Skeleton className="h-64 w-full rounded-lg" />
            </div>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
}

/**
 * Opens a section of Settings named in the address, which is how an email's "Change what
 * reaches you" link lands on it, and then takes the parameter back out so a reload or the
 * back button does not open Settings again.
 */
function useSettingsDeepLink() {
  const location = useLocation();
  const navigate = useNavigate();
  const openDialog = useAppDialogsStore((state) => state.openDialog);

  useEffect(() => {
    const params = new URLSearchParams(location.search);
    const section = params.get(SETTINGS_PARAM);
    if (section === null) return;

    openDialog("settings", isSettingsSection(section) ? { section } : undefined);
    params.delete(SETTINGS_PARAM);
    const search = params.toString();
    void navigate(
      { pathname: location.pathname, search: search ? `?${search}` : "", hash: location.hash },
      { replace: true },
    );
  }, [location.hash, location.pathname, location.search, navigate, openDialog]);
}

/**
 * Mounts the settings dialog wherever it is asked for from: the account menu, the command
 * palette, ⌘⇧S or a link. The chunk loads on the first open and stays, so a second open
 * does not flash the skeleton.
 */
export function UserSettingsHost() {
  const open = useAppDialogOpen("settings");
  const setDialogOpen = useAppDialogsStore((state) => state.setDialogOpen);
  const toggleDialog = useAppDialogsStore((state) => state.toggleDialog);
  const [mounted, setMounted] = useState(false);

  useSettingsDeepLink();
  useHotkey("Mod+Shift+S", () => toggleDialog("settings"), {
    ignoreInputs: false,
    preventDefault: true,
  });
  const handleOpenChange = useCallback(
    (next: boolean) => setDialogOpen("settings", next),
    [setDialogOpen],
  );

  if (open && !mounted) {
    setMounted(true);
  }

  if (!mounted) {
    return null;
  }

  return (
    <Suspense fallback={<UserSettingsDialogSkeleton open={open} onOpenChange={handleOpenChange} />}>
      <UserSettingsDialog open={open} onOpenChange={handleOpenChange} />
    </Suspense>
  );
}
