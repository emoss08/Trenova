import { useT } from "@trenova/shared/i18n/use-t";
import { ModuleRail } from "@/components/navigation/module-rail";
import { PageBarLeading, PageBarTrailing } from "@/components/navigation/rail-page-bar";
import { RailPanel, RailPanelBody } from "@/components/navigation/rail-panel";
import { MODULE_GO_KEYS } from "@/components/navigation/rail-shortcuts";
import { PageBarSlotsContext, type PageBarSlots } from "@/components/navigation/shell-frame";
import { SearchTrigger } from "@/components/navigation/sidebar-chrome";
import { useSidebarFold } from "@/components/navigation/sidebar-fold";
import { useGoToChords } from "@/components/navigation/use-go-to-chords";
import { UserMenu } from "@/components/navigation/user-menu";
import { NotificationSheet } from "@/components/notification-center/notification-sheet";
import { useSidebarNavigation } from "@/hooks/use-sidebar-navigation";
import { edition } from "@/lib/edition";
import { useNavigationStore } from "@/stores/navigation-store";
import { LayoutLeftIcon } from "@trenova/shared/components/icons";
import { Button } from "@trenova/shared/components/ui/button";
import { Sheet, SheetContent, SheetTitle } from "@trenova/shared/components/ui/sheet";
import { useMemo, type ReactNode } from "react";

const AppBanner = edition.slots.AppBanner;

const PAGE_BAR_SLOTS: PageBarSlots = {
  leading: (title) => <PageBarLeading title={title} />,
  trailing: <PageBarTrailing />,
};

function useModuleChordTargets() {
  const { views, home, settings } = useSidebarNavigation();
  return useMemo(() => {
    const targets = new Map<string, string>();
    for (const view of views.values()) {
      const key = MODULE_GO_KEYS[view.module.id];
      if (key) targets.set(key, view.landingPath);
    }
    if (home) targets.set(MODULE_GO_KEYS.home ?? "H", home.basePath);
    if (settings && MODULE_GO_KEYS.admin) targets.set(MODULE_GO_KEYS.admin, settings.basePath);
    return targets;
  }, [views, home, settings]);
}

/**
 * On a phone the rail comes out as a sheet, with the panel beside it on a
 * page; the page bar's fold opens it. A room's own rail is the room's to
 * draw on a phone, so its sheet holds the module rail alone. Whether the
 * sheet is out comes from the shell, which decided the sheet was needed from
 * the same reading of the viewport, so it never opens on the render that
 * first mounts it.
 */
function RailSheet({ shown, room }: { shown: boolean; room: boolean }) {
  const t = useT();
  const setCompactSidebarOpen = useNavigationStore((state) => state.setCompactSidebarOpen);
  const width = useNavigationStore((state) => state.railPanelWidth);

  return (
    <Sheet open={shown} onOpenChange={setCompactSidebarOpen}>
      <SheetContent
        side="left"
        showCloseButton={false}
        className="bg-sidebar flex-row gap-0 rounded-none border-y-0 border-l-0 p-0 data-[side=left]:w-auto data-[side=left]:max-w-none"
      >
        <SheetTitle className="sr-only">{t("Sidebar")}</SheetTitle>
        <ModuleRail panelFolded={false} search={room} />
        {!room && (
          <div style={{ width }} className="flex h-full min-h-0 flex-col">
            <RailPanelBody />
          </div>
        )}
      </SheetContent>
    </Sheet>
  );
}

/**
 * A room on a phone has no rail on screen and no page bar, so one strip
 * holds the way to the rest of the product: the sheet's toggle, search,
 * notifications and you.
 */
function RoomCompactBar() {
  const t = useT();
  const { toggle } = useSidebarFold();

  return (
    <div className="bg-sidebar border-border flex h-10 shrink-0 items-center gap-1 border-b px-2">
      <Button
        type="button"
        variant="ghost"
        size="icon-xs"
        aria-label={t("Show sidebar")}
        onClick={toggle}
      >
        <LayoutLeftIcon className="size-3.5" />
      </Button>
      <div className="min-w-0 flex-1 px-1">
        <SearchTrigger />
      </div>
      <NotificationSheet />
      <UserMenu compact />
    </div>
  );
}

/**
 * The shell of every page: the product's map down the left edge, the module
 * you are in beside it, and the page with one bar of its own. There is no
 * global header; what applies everywhere lives on the rail, and where you
 * are is the page's title. The panel folds with ⌘B and the rail stays.
 *
 * A room (Triage, Settings) draws its own rail where the panel would be and
 * folds it with the same ⌘B and the rail's fold tile; the module rail stays
 * beside it, so the rest of the product is one click away from inside the
 * room. A room whose rail stands (the Formula Studio, where the rate trace is
 * the room's working column) never folds it: the panel stays folded in its
 * favour, so the tiles open their module's pages on hover, and the rail
 * draws no fold tile.
 */
export function RailShell({
  children,
  room = false,
  railStands = false,
}: {
  children: ReactNode;
  room?: boolean;
  railStands?: boolean;
}) {
  const { compact, shown } = useSidebarFold();
  useGoToChords(useModuleChordTargets());

  return (
    <div className="flex h-screen flex-col overflow-hidden">
      <AppBanner />
      {compact && room && <RoomCompactBar />}
      <div className="flex min-h-0 flex-1">
        {compact ? (
          <RailSheet shown={shown} room={room} />
        ) : (
          <>
            <ModuleRail
              panelFolded={!shown || railStands}
              search={room || !shown}
              fold={room && !railStands}
            />
            {!room && <RailPanel shown={shown} />}
          </>
        )}
        <main className="min-w-0 flex-1 overflow-y-auto">
          {room ? (
            children
          ) : (
            <PageBarSlotsContext.Provider value={PAGE_BAR_SLOTS}>
              {children}
            </PageBarSlotsContext.Provider>
          )}
        </main>
      </div>
    </div>
  );
}
