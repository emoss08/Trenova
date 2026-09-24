import { useT } from "@trenova/shared/i18n/use-t";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@trenova/shared/components/ui/dialog";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useAppDialogOpen, useAppDialogsStore } from "@/stores/app-dialogs-store";
import { useHotkey } from "@tanstack/react-hotkeys";
import { lazy, Suspense, useCallback, useState } from "react";

// The settings dialog carries the timezone/time-format choice tables, the
// avatar cropper and the react-hook-form field set — none of which belong in
// the chunk that renders the sidebar on every page.
const UserSettingsDialog = lazy(() =>
  import("@/components/navigation/user-settings-dialog").then((module) => ({
    default: module.UserSettingsDialog,
  })),
);

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
      <DialogContent size="md">
        <DialogHeader>
          <DialogTitle>{t("Settings")}</DialogTitle>
          <DialogDescription>{t("Manage your preferences and security.")}</DialogDescription>
        </DialogHeader>
        <div className="bg-sidebar flex items-center gap-4 rounded-md border p-4">
          <Skeleton className="size-14 shrink-0 rounded-md" />
          <div className="flex min-w-0 flex-1 flex-col gap-1.5">
            <Skeleton className="h-3.5 w-40 rounded-md" />
            <Skeleton className="h-3 w-28 rounded-md" />
            <Skeleton className="h-3 w-48 rounded-md" />
          </div>
        </div>
        <div className="space-y-5">
          {Array.from({ length: 3 }, (_, section) => (
            <div key={section} className="space-y-3">
              <div className="flex items-center gap-3">
                <Skeleton className="size-8 shrink-0 rounded-lg" />
                <div className="flex flex-col gap-1.5">
                  <Skeleton className="h-3.5 w-32 rounded-md" />
                  <Skeleton className="h-3 w-56 rounded-md" />
                </div>
              </div>
              <Skeleton className="h-9 w-full rounded-md" />
            </div>
          ))}
        </div>
      </DialogContent>
    </Dialog>
  );
}

/**
 * Mounts the settings dialog wherever it is asked for from: the account menu
 * or the command palette. The chunk loads on the first open and stays, so a
 * second open does not flash the skeleton.
 */
export function UserSettingsHost() {
  const open = useAppDialogOpen("settings");
  const setDialogOpen = useAppDialogsStore((state) => state.setDialogOpen);
  const openDialog = useAppDialogsStore((state) => state.openDialog);
  const [mounted, setMounted] = useState(false);

  useHotkey("Mod+Shift+S", () => openDialog("settings"), {
    ignoreInputs: true,
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
