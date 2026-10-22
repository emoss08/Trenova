import { usePermission } from "@/hooks/use-permission";
import { lazyComponent } from "@/lib/lazy-component";
import { useAppDialogOpen, useAppDialogsStore } from "@/stores/app-dialogs-store";
import { useHotkey } from "@tanstack/react-hotkeys";
import { ShieldTickIcon } from "@trenova/shared/components/icons";
import { Button } from "@trenova/shared/components/ui/button";
import { Sheet, SheetContent } from "@trenova/shared/components/ui/sheet";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import { useT } from "@trenova/shared/i18n/use-t";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { Suspense } from "react";
import { useLocation } from "react-router";
import { PageAccessPanelSkeleton } from "./page-access-panel-skeleton";

/** The keybind that opens the panel, as `useHotkey` takes it. */
export const PAGE_ACCESS_HOTKEY = "Mod+Alt+P";

// The panel, its queries and the role picker are read when it first opens, or
// when its button is pointed at, so no page's first visit carries them.
const Panel = lazyComponent(
  () => import("./page-access-panel"),
  (module) => module.PageAccessPanel,
);

const preloadPanel = () => void Panel.preload().catch(() => undefined);

/**
 * Whether the page access panel is offered: to administrators, who manage
 * roles, and on every screen of a development build.
 */
export function usePageAccessPanel() {
  const { allowed } = usePermission(Resource.Role, Operation.Read);
  return { shown: allowed || import.meta.env.DEV, canPreview: allowed };
}

/**
 * The page access panel's sheet and keybind, mounted once in the shell so it
 * opens on every screen, rooms included. Its body is read on first open.
 */
export function PageAccessMount() {
  const { pathname } = useLocation();
  const { shown, canPreview } = usePageAccessPanel();
  const open = useAppDialogOpen("page-access");
  const setDialogOpen = useAppDialogsStore((state) => state.setDialogOpen);
  const toggleDialog = useAppDialogsStore((state) => state.toggleDialog);

  useHotkey(
    PAGE_ACCESS_HOTKEY,
    () => {
      preloadPanel();
      toggleDialog("page-access");
    },
    { enabled: shown, ignoreInputs: true, preventDefault: true },
  );

  if (!shown) {
    return null;
  }

  return (
    <Sheet open={open} onOpenChange={(next) => setDialogOpen("page-access", next)}>
      <SheetContent side="right" className="w-full gap-0 p-0 sm:max-w-lg">
        {open ? (
          <Suspense fallback={<PageAccessPanelSkeleton />}>
            <Panel.Component location={pathname} canPreview={canPreview} />
          </Suspense>
        ) : null}
      </SheetContent>
    </Sheet>
  );
}

/** The page bar's button that opens the page access panel for the screen on view. */
export function PageAccessTrigger() {
  const t = useT();
  const { shown } = usePageAccessPanel();
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
            onPointerEnter={preloadPanel}
            onFocus={preloadPanel}
            onClick={() => openDialog("page-access")}
          />
        }
      >
        <ShieldTickIcon className="size-3.5" />
      </TooltipTrigger>
      <TooltipContent>{t("Page access")}</TooltipContent>
    </Tooltip>
  );
}
