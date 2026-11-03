import Supercluster from "supercluster";
import type { Latlng } from "./shipment-route-coordinates";

/* supercluster measures its radius against `extent`. Google's tiles are 256
   pixels, so an extent of 256 makes the radius a radius in screen pixels at
   Google's zoom levels. */
const TILE_EXTENT = 256;
/** Past this zoom nothing is folded; a cluster still whole here is one place. */
export const CLUSTER_MAX_ZOOM = 16;
const WORLD: [number, number, number, number] = [-180, -85, 180, 85];

export type ClusterPoint<T> = { key: string; latlng: Latlng; item: T };

export type MapCluster<T> = {
  key: string;
  latlng: Latlng;
  items: T[];
  /**
   * The zoom at which this cluster first splits, or null when its members sit
   * in one place and no zoom will separate them. Null for a lone point.
   */
  expansionZoom: number | null;
};

type PointProps = { index: number };

/**
 * Folds points that would draw within `radiusPx` of each other at `zoom` into
 * clusters. A lone point comes back as a cluster of one under its own key.
 */
export function clusterPoints<T>(
  points: readonly ClusterPoint<T>[],
  zoom: number,
  radiusPx: number,
): MapCluster<T>[] {
  if (points.length === 0) return [];

  const index = new Supercluster<PointProps>({
    radius: radiusPx,
    extent: TILE_EXTENT,
    maxZoom: CLUSTER_MAX_ZOOM,
  });
  index.load(
    points.map((point, i) => ({
      type: "Feature",
      geometry: { type: "Point", coordinates: [point.latlng.lng, point.latlng.lat] },
      properties: { index: i },
    })),
  );

  // Above its max zoom supercluster hands back every point on its own, which
  // would stack loads at one yard invisibly on top of each other.
  const queryZoom = Math.min(Math.floor(zoom), CLUSTER_MAX_ZOOM);
  return index.getClusters(WORLD, queryZoom).map((feature) => {
    const [lng, lat] = feature.geometry.coordinates;
    const props = feature.properties;
    if (!("cluster" in props) || !props.cluster) {
      const point = points[(props as PointProps).index];
      return { key: point.key, latlng: point.latlng, items: [point.item], expansionZoom: null };
    }
    const members = index
      .getLeaves(props.cluster_id, Infinity)
      .map((leaf) => points[leaf.properties.index]);
    const expansionZoom = index.getClusterExpansionZoom(props.cluster_id);
    return {
      key: members.map((m) => m.key).join("|"),
      latlng: { lat, lng },
      items: members.map((m) => m.item),
      expansionZoom: expansionZoom > CLUSTER_MAX_ZOOM ? null : expansionZoom,
    };
  });
}
