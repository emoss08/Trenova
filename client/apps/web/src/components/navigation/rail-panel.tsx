import { useT } from "@trenova/shared/i18n/use-t";
import { ModulePageList, SearchTrigger } from "@/components/navigation/sidebar-chrome";
import { moduleDisplayLabel, moduleSidebarView } from "@/components/navigation/sidebar-model";
import { HomePanel } from "@/components/navigation/home-panel";
import { useSidebarNavigation } from "@/hooks/use-sidebar-navigation";
import {
  clampRailPanelWidth,
  RAIL_PANEL_DEFAULT_WIDTH,
  RAIL_PANEL_MAX_WIDTH,
  RAIL_PANEL_MIN_WIDTH,
  useNavigationStore,
} from "@/stores/navigation-store";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { cn } from "@trenova/shared/lib/utils";
import { useCallback, useRef, useState, type PointerEvent as ReactPointerEvent } from "react";

/**
 * The panel's edge, which drags. It is a line the pointer can take hold of,
 * with a little more to grab than it shows, and a double click puts the
 * width back.
 */
function ResizeHandle({ width }: { width: number }) {
  const t = useT();
  const setWidth = useNavigationStore((state) => state.setRailPanelWidth);
  const [dragging, setDragging] = useState(false);
  const origin = useRef<{ x: number; width: number } | null>(null);

  const onPointerDown = useCallback(
    (event: ReactPointerEvent<HTMLDivElement>) => {
      if (event.button !== 0) return;
      event.preventDefault();
      origin.current = { x: event.clientX, width };
      event.currentTarget.setPointerCapture(event.pointerId);
      setDragging(true);
    },
    [width],
  );

  const onPointerMove = useCallback(
    (event: ReactPointerEvent<HTMLDivElement>) => {
      if (!origin.current) return;
      setWidth(clampRailPanelWidth(origin.current.width + event.clientX - origin.current.x));
    },
    [setWidth],
  );

  const onPointerUp = useCallback((event: ReactPointerEvent<HTMLDivElement>) => {
    origin.current = null;
    event.currentTarget.releasePointerCapture(event.pointerId);
    setDragging(false);
  }, []);

  return (
    <div
      role="separator"
      aria-orientation="vertical"
      aria-label={t("Resize sidebar")}
      aria-valuemin={RAIL_PANEL_MIN_WIDTH}
      aria-valuemax={RAIL_PANEL_MAX_WIDTH}
      aria-valuenow={width}
      tabIndex={0}
      onPointerDown={onPointerDown}
      onPointerMove={onPointerMove}
      onPointerUp={onPointerUp}
      onPointerCancel={onPointerUp}
      onDoubleClick={() => setWidth(RAIL_PANEL_DEFAULT_WIDTH)}
      onKeyDown={(event) => {
        if (event.key === "ArrowLeft") setWidth(width - 16);
        if (event.key === "ArrowRight") setWidth(width + 16);
      }}
      className={cn(
        "ui-focus-ring group absolute inset-y-0 -right-1 z-10 w-2 cursor-col-resize outline-none select-none",
      )}
    >
      <span
        aria-hidden
        className={cn(
          "absolute inset-y-0 left-[3px] w-0.5 transition-colors",
          dragging ? "bg-border-strong" : "bg-transparent group-hover:bg-border-strong",
        )}
      />
    </div>
  );
}

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
      {shown && <ResizeHandle width={width} />}
    </aside>
  );
}
