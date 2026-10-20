import { PanelResizeHandle } from "@/components/elements/panel-resize-handle";
import {
  DESK_RAIL_DEFAULT_WIDTH,
  DESK_RAIL_MAX_WIDTH,
  DESK_RAIL_MIN_WIDTH,
  useDeskSettingsStore,
} from "@/stores/desk-settings-store";
import { useT } from "@trenova/shared/i18n/use-t";
import type { CSSProperties, ReactNode } from "react";

/**
 * The rail's own box, as wide as the person dragged it. Only this reads the
 * width, so a drag redraws the box and not the conversations inside it. A
 * folded rail is drawn at no width and takes no focus; dragging its edge well
 * past the narrowest and letting go folds it. On a phone the rail is a sheet
 * of its own width and the edge is not drawn.
 */
export function DeskRailFrame({
  folded,
  onFold,
  children,
}: {
  folded: boolean;
  onFold: () => void;
  children: ReactNode;
}) {
  const t = useT();
  const width = useDeskSettingsStore((state) => state.railWidth);
  const setWidth = useDeskSettingsStore((state) => state.setRailWidth);

  return (
    <aside
      className="dk-sb dk-sb-rs"
      aria-label={t("Desk")}
      data-folded={folded || undefined}
      inert={folded || undefined}
      style={{ "--dk-rail-w": `${width}px` } as CSSProperties}
    >
      <div className="dk-sb-in">{children}</div>
      {!folded && (
        <PanelResizeHandle
          className="dk-sb-rz max-[720px]:hidden"
          width={width}
          min={DESK_RAIL_MIN_WIDTH}
          max={DESK_RAIL_MAX_WIDTH}
          defaultWidth={DESK_RAIL_DEFAULT_WIDTH}
          onResize={setWidth}
          onFold={onFold}
          label={t("Resize sidebar")}
        />
      )}
    </aside>
  );
}
