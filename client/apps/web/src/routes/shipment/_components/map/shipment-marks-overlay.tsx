import { useT } from "@trenova/shared/i18n/use-t";
import { DEFAULT_ZOOM } from "@trenova/shared/lib/constants";
import { useMap } from "@vis.gl/react-google-maps";
import { useSyncExternalStore } from "react";
import { clusterPoints, type MapCluster } from "./map-clusters";
import { useMapFocusStore } from "./map-focus-store";
import { placeLabel, type MapShipment } from "./map-shipment-model";
import { ShipmentClusterMark, ShipmentMark } from "./shipment-map-marks";

/** Marks closer than this on screen fold into one count. */
const CLUSTER_RADIUS_PX = 26;

/**
 * The map's whole zoom level. Clusters change only at whole levels, so the
 * fractional steps of a pinch or an animated zoom do not re-cluster.
 */
function useWholeZoom(): number {
  const map = useMap();
  return useSyncExternalStore(
    (notify) => {
      const listener = map?.addListener("zoom_changed", notify);
      return () => listener?.remove();
    },
    () => Math.floor(map?.getZoom() ?? DEFAULT_ZOOM),
  );
}

/**
 * One mark per shipment, at the stop it is at or heading to. Marks that would
 * overlap fold into a count, except a late load and a pinned one (selected, or
 * touched by the selected alert), which always stand alone. Hover is not an
 * input here: each mark reads its own, so moving the pointer re-clusters nothing.
 */
export function ShipmentMarksOverlay({
  models,
  pinnedIds,
}: {
  models: readonly MapShipment[];
  pinnedIds: ReadonlySet<string>;
}) {
  const t = useT();
  const map = useMap();
  const zoom = useWholeZoom();
  const setScope = useMapFocusStore((s) => s.setScope);

  const alone = models.filter((m) => m.stage === "Late" || pinnedIds.has(m.id));
  const foldable = models.filter((m) => m.stage !== "Late" && !pinnedIds.has(m.id));
  const clusters = clusterPoints(
    foldable.map((m) => ({ key: m.id, latlng: m.position, item: m })),
    zoom,
    CLUSTER_RADIUS_PX,
  );

  /* A cluster zooms in to where it first splits. One that never splits is a
     single yard, so the rail narrows to its loads instead. */
  const openCluster = (cluster: MapCluster<MapShipment>) => {
    if (cluster.expansionZoom != null && map) {
      map.setCenter(cluster.latlng);
      map.setZoom(cluster.expansionZoom);
      return;
    }
    const members = cluster.items;
    const first = members[0];
    const place = placeLabel(first.stops[first.currentIndex].stop.location);
    setScope({
      ids: members.map((m) => m.id),
      label: place
        ? t("{0, plural, one {# shipment} other {# shipments}} near {1}", members.length, place)
        : t("{0, plural, one {# shipment} other {# shipments}}", members.length),
    });
  };

  const renderMark = (model: MapShipment) => (
    <ShipmentMark
      key={model.id}
      model={model}
      title={[model.shipment.proNumber, model.shipment.customer?.name].filter(Boolean).join(" · ")}
      pinned={pinnedIds.has(model.id)}
    />
  );

  return (
    <>
      {clusters.map((cluster) =>
        cluster.items.length === 1 ? (
          renderMark(cluster.items[0])
        ) : (
          <ShipmentClusterMark
            key={cluster.key}
            latlng={cluster.latlng}
            count={cluster.items.length}
            movingShare={
              cluster.items.filter((m) => m.stage === "Moving").length / cluster.items.length
            }
            title={t("{0, plural, one {# shipment} other {# shipments}}", cluster.items.length)}
            onSelect={() => openCluster(cluster)}
          />
        ),
      )}
      {alone.map(renderMark)}
    </>
  );
}
