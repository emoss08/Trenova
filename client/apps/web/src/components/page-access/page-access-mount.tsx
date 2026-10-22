import { usePermission } from "@/hooks/use-permission";
import { lazyComponent } from "@/lib/lazy-component";
import { useAppDialogOpen, useAppDialogsStore } from "@/stores/app-dialogs-store";
import { useHotkey } from "@tanstack/react-hotkeys";
import { ShieldTickIcon } from "@trenova/shared/components/icons";
import { Button } from "@trenova/shared/components/ui/button";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import { useT } from "@trenova/shared/i18n/use-t";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { Suspense, useState } from "react";

/** The keybind that opens the panel, as `useHotkey` takes it. */
export const PAGE_ACCESS_HOTKEY = "Mod+Alt+P";

// Everything the panel draws (the sheet, its queries, the role picker and the
// report) is read when it first opens, or when its button is pointed at, so
// the shell carries only the keybind and the button.
const PageAccessSheet = lazyComponent(
  () => import("./page-access-sheet"),
  (module) => module.PageAccessSheet,
);

/** Reads the panel's code ahead of its first opening. */
export function preloadPageAccess(): void {
  void PageAccessSheet.preload().catch(() => undefined);
}

/**
 * Whether the page access panel is offered: to administrators, who manage
 * roles, and on every screen of a development build.
 */
export function usePageAccessShown(): boolean {
  const { allowed } = usePermission(Resource.Role, Operation.Read);
  return allowed || import.meta.env.DEV;
}

/** The sheet's place while its code loads, drawn where the sheet opens. */
function PageAccessSheetSkeleton() {
  const t = useT();

  return (
    <div
      aria-busy="true"
      aria-label={t("Loading page access")}
      className="fixed inset-y-4 right-4 z-50 flex w-3/4 flex-col gap-4 rounded-lg border border-border bg-overlay p-4 sm:max-w-lg"
    >
      <Skeleton aria-hidden className="h-5 w-32" />
      <Skeleton aria-hidden className="h-8 w-full rounded-md" />
      <Skeleton aria-hidden className="h-40 w-full rounded-md" />
    </div>
  );
}

/**
 * The page access keybind, mounted once in the shell so it works on every
 * screen, rooms included. The sheet is mounted on its first opening and stays
 * mounted, so it closes with its own transition.
 */
export function PageAccessMount() {
  const shown = usePageAccessShown();
  const open = useAppDialogOpen("page-access");
  const toggleDialog = useAppDialogsStore((state) => state.toggleDialog);
  const [mounted, setMounted] = useState(false);

  useHotkey(
    PAGE_ACCESS_HOTKEY,
    () => {
      preloadPageAccess();
      toggleDialog("page-access");
    },
    { enabled: shown, ignoreInputs: true, preventDefault: true },
  );

  // Set during render so the sheet mounts in the commit that opens it.
  if (open && shown && !mounted) {
    setMounted(true);
  }

  if (!shown || !mounted) {
    return null;
  }

  return (
    <Suspense fallback={open ? <PageAccessSheetSkeleton /> : null}>
      <PageAccessSheet.Component />
    </Suspense>
  );
}

/** The page bar's button that opens the page access panel for the screen on view. */
export function PageAccessTrigger() {
  const t = useT();
  const shown = usePageAccessShown();
  const openDialog = useAppDialogsStore((state) => state.openDialog);

  if (!shown) {
    return null;
  }

  return (
    <Tooltip>
      <TooltipTrigger
        render={
          <Button
            type="button"
            variant="ghost"
            size="icon-xs"
            aria-label={t("Page access")}
            onPointerEnter={preloadPageAccess}
            onFocus={preloadPageAccess}
            onClick={() => {
              preloadPageAccess();
              openDialog("page-access");
            }}
          />
        }
      >
        <ShieldTickIcon className="size-3.5" />
      </TooltipTrigger>
      <TooltipContent>{t("Page access")}</TooltipContent>
    </Tooltip>
  );
}
