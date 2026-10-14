import { useT } from "@trenova/shared/i18n/use-t";
import { ModuleRail } from "@/components/navigation/module-rail";
import { PageBarLeading, PageBarTrailing } from "@/components/navigation/rail-page-bar";
import { RailPanel, RailPanelBody } from "@/components/navigation/rail-panel";
import { MODULE_GO_KEYS } from "@/components/navigation/rail-shortcuts";
import { PageBarSlotsContext, type PageBarSlots } from "@/components/navigation/shell-frame";
import { useSidebarFold } from "@/components/navigation/sidebar-fold";
import { useGoToChords } from "@/components/navigation/use-go-to-chords";
import { useSidebarNavigation } from "@/hooks/use-sidebar-navigation";
import { edition } from "@/lib/edition";
import { useNavigationStore } from "@/stores/navigation-store";
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
 * On a phone the rail and the panel come out together as one sheet; the
 * page bar's fold opens it. Whether it is out comes from the shell, which
 * decided the sheet was needed from the same reading of the viewport, so
 * the sheet never opens on the render that first mounts it.
 */
function RailSheet({ shown }: { shown: boolean }) {
  const t = useT();
  const setCompactSidebarOpen = useNavigationStore((state) => state.setCompactSidebarOpen);
  const width = useNavigationStore((state) => state.railPanelWidth);

  return (
    <Sheet open={shown} onOpenChange={setCompactSidebarOpen}>
      <SheetContent
        side="left"
        showCloseButton={false}
        className="bg-sidebar w-auto flex-row gap-0 rounded-none border-y-0 border-l-0 p-0"
      >
        <SheetTitle className="sr-only">{t("Sidebar")}</SheetTitle>
        <ModuleRail panelFolded={false} />
        <div style={{ width }} className="flex h-full min-h-0 flex-col">
          <RailPanelBody />
        </div>
      </SheetContent>
    </Sheet>
  );
}

/**
 * The shell of every page: the product's map down the left edge, the module
 * you are in beside it, and the page with one bar of its own. There is no
 * global header; what applies everywhere lives on the rail, and where you
 * are is the page's title. The panel folds with ⌘B and the rail stays.
 */
export function RailShell({ children }: { children: ReactNode }) {
  const { compact, shown } = useSidebarFold();
  useGoToChords(useModuleChordTargets());

  return (
    <div className="flex h-screen flex-col overflow-hidden">
      <AppBanner />
      <div className="flex min-h-0 flex-1">
        {compact ? (
          <RailSheet shown={shown} />
        ) : (
          <>
            <ModuleRail panelFolded={!shown} className="max-md:hidden" />
            <RailPanel shown={shown} />
          </>
        )}
        <main className="min-w-0 flex-1 overflow-y-auto">
          <PageBarSlotsContext.Provider value={PAGE_BAR_SLOTS}>
            {children}
          </PageBarSlotsContext.Provider>
        </main>
      </div>
    </div>
  );
}
