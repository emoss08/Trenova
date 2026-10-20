import { useT } from "@trenova/shared/i18n/use-t";
import { ModulePageList, SearchTrigger } from "@/components/navigation/sidebar-chrome";
import { moduleDisplayLabel, moduleSidebarView } from "@/components/navigation/sidebar-model";
import { HomePanel } from "@/components/navigation/home-panel";
import { useSidebarNavigation } from "@/hooks/use-sidebar-navigation";
import {
  RAIL_PANEL_DEFAULT_WIDTH,
  RAIL_PANEL_MAX_WIDTH,
  RAIL_PANEL_MIN_WIDTH,
  useNavigationStore,
} from "@/stores/navigation-store";
import { PanelResizeHandle } from "@/components/elements/panel-resize-handle";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { cn } from "@trenova/shared/lib/utils";

function PanelHeading({ children }: { children: string }) {
  return (
    <h2 className="text-foreground truncate px-4 pt-3 pb-1 text-xs font-semibold select-none">
      {children}
    </h2>
  );
}

export function RailPanelBody() {
  const t = useT();
  const { home, views, activeModule, activePath, pageAttention, hiddenSections } =
    useSidebarNavigation();
  const activeView = moduleSidebarView(activeModule, views);

  return (
    <>
      <div className="px-2 pt-2">
        <SearchTrigger />
      </div>
      <ScrollArea className="min-h-0 flex-1" maskHeight={16} maskVariant="sidebar">
        {activeView ? (
          <div key={activeView.module.id} className="animate-rise">
            <PanelHeading>{t(moduleDisplayLabel(activeView.module))}</PanelHeading>
            <ModulePageList
              view={activeView}
              activePath={activePath}
              attentionByPath={pageAttention}
              className="px-2"
            />
          </div>
        ) : home ? (
          <div key="home" className="animate-rise">
            <HomePanel hiddenSections={hiddenSections} />
          </div>
        ) : null}
        <div className="h-2" aria-hidden />
      </ScrollArea>
    </>
  );
}

/**
 * What the rail's current tile holds: the pages of the module you are in,
 * or on Home the work that needs you. Search leads it because the panel is
 * where names live. Its width is yours to drag; ⌘B folds it to nothing and
 * the rail's tiles then open the same lists on hover.
 */
export function RailPanel({ shown }: { shown: boolean }) {
  const t = useT();
  const width = useNavigationStore((state) => state.railPanelWidth);
  const setWidth = useNavigationStore((state) => state.setRailPanelWidth);

  return (
    <aside
      aria-label={t("Sidebar")}
      inert={!shown || undefined}
      style={{ width: shown ? width : 0 }}
      className={cn(
        "bg-sidebar border-border relative flex h-full shrink-0 flex-col border-r transition-[width] duration-200 ease-settle",
        !shown && "overflow-hidden border-r-0",
      )}
    >
      <div style={{ width }} className="flex h-full min-h-0 flex-col">
        <RailPanelBody />
      </div>
      {shown && (
        <PanelResizeHandle
          width={width}
          min={RAIL_PANEL_MIN_WIDTH}
          max={RAIL_PANEL_MAX_WIDTH}
          defaultWidth={RAIL_PANEL_DEFAULT_WIDTH}
          onResize={setWidth}
          label={t("Resize sidebar")}
        />
      )}
    </aside>
  );
}
