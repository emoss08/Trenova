import { STAGE_META } from "@/lib/shipment-board/stage";
import { useShipmentBoardStore } from "../board/store";
import { isLegDriven, placeLabel, type MapShipment } from "./map-shipment-model";
import { MapPolyline } from "@/components/map/map-polyline";
import { RouteStopPin } from "./shipment-map-marks";

/**
 * How a route in focus is drawn: the selected shipment boldest with its stops
 * named, a hovered one as a preview, and the loads a selected alert touches as
 * thin context lines.
 */
export type RouteEmphasis = "selected" | "preview" | "context";

const STYLE: Record<RouteEmphasis, { weight: number; opacity: number; zIndex: number }> = {
  selected: { weight: 4, opacity: 0.95, zIndex: 90 },
  preview: { weight: 3.5, opacity: 0.85, zIndex: 80 },
  context: { weight: 2.5, opacity: 0.7, zIndex: 70 },
};

export type RouteRequest = { model: MapShipment; emphasis: RouteEmphasis };

export function ShipmentFocusRoutes({ routes }: { routes: readonly RouteRequest[] }) {
  return (
    <>
      {routes.map(({ model, emphasis }) => (
        <ShipmentRoute key={model.id} model={model} emphasis={emphasis} />
      ))}
    </>
  );
}

/**
 * The preview route of whatever shipment the pointer rests on, on the map or
 * in the rail. It is the only part of the map that hover redraws a route for.
 */
export function HoveredRoute({
  modelsById,
  selectedId,
}: {
  modelsById: ReadonlyMap<string, MapShipment>;
  selectedId: string | null;
}) {
  const hoveredId = useShipmentBoardStore((s) => s.highlightId);
  const model = hoveredId && hoveredId !== selectedId ? modelsById.get(hoveredId) : undefined;
  return model ? <ShipmentRoute model={model} emphasis="preview" /> : null;
}

/** Driven legs solid, legs still to drive dotted, and a numbered pin at each stop. */
function ShipmentRoute({ model, emphasis }: RouteRequest) {
  const colorVar = STAGE_META[model.stage].colorVar;
  const style = STYLE[emphasis];

  return (
    <>
      {model.stops.slice(0, -1).map((from, index) => (
        <MapPolyline
          key={`leg-${from.stop.id ?? index}`}
          path={[from.latlng, model.stops[index + 1].latlng]}
          colorVar={colorVar}
          weight={style.weight}
          opacity={style.opacity}
          dashed={!isLegDriven(model, index)}
          zIndex={style.zIndex}
        />
      ))}
      {model.stops.map(({ stop, latlng }, index) =>
        index === model.currentIndex ? null : (
          <RouteStopPin
            key={`stop-${stop.id ?? index}`}
            latlng={latlng}
            number={index + 1}
            done={stop.status === "Completed"}
            colorVar={colorVar}
            label={emphasis === "context" ? null : placeLabel(stop.location)}
          />
        ),
      )}
    </>
  );
}
