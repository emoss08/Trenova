import { useT } from "@trenova/shared/i18n/use-t";
import { AdvancedMarker } from "@vis.gl/react-google-maps";
import { useMapFocusStore } from "./map-focus-store";
import { groupLanes, laneArcPath, laneWeight, type MapLane } from "./map-lanes";
import type { MapShipment } from "./map-shipment-model";
import { MapPolyline } from "@/components/map/map-polyline";

/** Lanes this busy carry their count at the middle of the arc. */
const LABELED_LANE_MIN = 3;

/**
 * The network view: one arc per origin and destination, sized by how many
 * loads run it and drawn in the danger tone when any of them is late.
 * Clicking an arc narrows the rail to that lane.
 */
export function ShipmentLaneOverlay({
  models,
  dimmed,
}: {
  models: readonly MapShipment[];
  dimmed: boolean;
}) {
  const lanes = groupLanes(models);

  return (
    <>
      {lanes.map((lane) => (
        <LaneArc key={lane.key} lane={lane} dimmed={dimmed} />
      ))}
    </>
  );
}

function LaneArc({ lane, dimmed }: { lane: MapLane; dimmed: boolean }) {
  const t = useT();
  const setScope = useMapFocusStore((s) => s.setScope);
  const path = laneArcPath(lane.origin, lane.destination);
  const count = lane.shipmentIds.length;
  const late = lane.lateCount > 0;

  return (
    <>
      <MapPolyline
        path={path}
        colorVar={late ? "--danger" : "--info"}
        weight={laneWeight(count)}
        opacity={dimmed ? 0.1 : late ? 0.55 : 0.35}
        geodesic={false}
        zIndex={4}
        onClick={() =>
          setScope({
            ids: lane.shipmentIds,
            label: t("{0} to {1}", lane.originLabel, lane.destinationLabel),
          })
        }
      />
      {count >= LABELED_LANE_MIN && !dimmed ? (
        <AdvancedMarker
          position={path[Math.floor(path.length / 2)]}
          zIndex={5}
          anchorLeft="-50%"
          anchorTop="-50%"
          clickable={false}
        >
          <span className="bg-card text-foreground rounded-badge pointer-events-none px-1 font-mono text-2xs font-medium tabular-nums">
            {count}
          </span>
        </AdvancedMarker>
      ) : null}
    </>
  );
}
