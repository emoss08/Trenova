import { cn } from "@trenova/shared/lib/utils";
import { AdvancedMarker } from "@vis.gl/react-google-maps";
import { STAGE_META } from "@/lib/shipment-board/stage";
import { useShipmentBoardStore } from "../board/store";
import { useMapFocusStore } from "./map-focus-store";
import type { MapShipment, MapStage } from "./map-shipment-model";
import type { Latlng } from "./shipment-route-coordinates";

/* A late load is the largest mark so it reads first; a load that has not
   started is a hollow ring, so a yard full of them stays quiet. */
const MARK_SIZE: Record<MapStage, number> = {
  Late: 14,
  NeedsCoverage: 12,
  Moving: 12,
  Scheduled: 10,
  Delivered: 10,
};
const HOLLOW_STAGES = new Set<MapStage>(["Scheduled", "Delivered"]);

/* A marker click must not also reach the map, whose click clears the selection. */
function stopAtMarker(event: unknown) {
  if (event && typeof event === "object" && "stop" in event && typeof event.stop === "function") {
    event.stop();
  }
}

function inkOn(colorVar: string): string {
  return colorVar === "--warning" ? "var(--warning-on-solid)" : "var(--foreground-on-solid)";
}

/**
 * One shipment's mark. It reads its own hover from the board store, so a
 * pointer moving across the map or the rail redraws only the marks it leaves
 * and enters; dimming the rest is a style on the map (`data-map-dim`).
 */
export function ShipmentMark({
  model,
  title,
  pinned,
}: {
  model: MapShipment;
  title: string;
  /** Selected, or touched by the selected alert. */
  pinned: boolean;
}) {
  const hovered = useShipmentBoardStore((s) => s.highlightId === model.id);
  const setHighlightId = useShipmentBoardStore((s) => s.setHighlightId);
  const toggleShipment = useMapFocusStore((s) => s.toggleShipment);
  const focused = pinned || hovered;
  const color = `var(${STAGE_META[model.stage].colorVar})`;
  const size = MARK_SIZE[model.stage] + (focused ? 4 : 0);
  const hollow = HOLLOW_STAGES.has(model.stage);

  return (
    <AdvancedMarker
      position={model.position}
      zIndex={focused ? 100 : model.stage === "Late" ? 60 : 30}
      anchorLeft="-50%"
      anchorTop="-50%"
      title={title}
      onClick={(event) => {
        stopAtMarker(event);
        toggleShipment(model.id);
      }}
    >
      <div
        data-map-mark
        data-focused={focused}
        className={cn(
          "relative rounded-full transition-[opacity,width,height] duration-150",
          model.stage === "Late" && "cc-pulse-pin",
          focused && "is-highlighted",
        )}
        onMouseEnter={() => setHighlightId(model.id)}
        onMouseLeave={() => {
          if (useShipmentBoardStore.getState().highlightId === model.id) setHighlightId(null);
        }}
        style={{ width: size, height: size, color }}
      >
        <span
          aria-hidden
          className="block size-full rounded-full"
          style={{
            background: hollow ? "var(--card)" : color,
            border: hollow ? `2px solid ${color}` : "2px solid var(--card)",
            outline: focused ? "2px solid var(--foreground)" : undefined,
            outlineOffset: 2,
          }}
        />
      </div>
    </AdvancedMarker>
  );
}

/**
 * Shipments too close to tell apart at this zoom, drawn as one count. The ring
 * shows what share of them is moving.
 */
export function ShipmentClusterMark({
  latlng,
  count,
  movingShare,
  title,
  onSelect,
}: {
  latlng: Latlng;
  count: number;
  movingShare: number;
  title: string;
  onSelect: () => void;
}) {
  const size = 22 + Math.min(Math.sqrt(count) * 4, 18);
  const turn = Math.max(0, Math.min(1, movingShare));

  return (
    <AdvancedMarker
      position={latlng}
      zIndex={20}
      anchorLeft="-50%"
      anchorTop="-50%"
      title={title}
      onClick={(event) => {
        stopAtMarker(event);
        onSelect();
      }}
    >
      <div
        data-map-cluster
        className="grid place-items-center rounded-full transition-opacity duration-150"
        style={{
          width: size,
          height: size,
          padding: 3,
          background: `conic-gradient(var(--info) ${turn}turn, var(--border-strong) 0)`,
        }}
      >
        <span className="bg-card text-foreground grid size-full place-items-center rounded-full font-mono text-xs font-medium tabular-nums">
          {count}
        </span>
      </div>
    </AdvancedMarker>
  );
}

/** A numbered stop on a route in focus; filled once the stop is done. */
export function RouteStopPin({
  latlng,
  number,
  done,
  colorVar,
  label,
}: {
  latlng: Latlng;
  number: number;
  done: boolean;
  colorVar: string;
  label?: string | null;
}) {
  const color = `var(${colorVar})`;

  return (
    <AdvancedMarker position={latlng} zIndex={85} anchorLeft="-8px" anchorTop="-8px">
      <div className="pointer-events-none flex items-center gap-1.5">
        <span
          aria-hidden
          className="grid size-4 place-items-center rounded-full font-mono text-2xs font-semibold tabular-nums"
          style={{
            background: done ? color : "var(--card)",
            color: done ? inkOn(colorVar) : color,
            border: `2px solid ${done ? "var(--card)" : color}`,
          }}
        >
          {number}
        </span>
        {label ? (
          <span className="bg-card/90 text-foreground rounded-badge px-1 text-xs font-medium whitespace-nowrap">
            {label}
          </span>
        ) : null}
      </div>
    </AdvancedMarker>
  );
}
