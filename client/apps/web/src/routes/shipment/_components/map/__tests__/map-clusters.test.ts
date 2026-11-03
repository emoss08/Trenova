import { describe, expect, it } from "vitest";
import { CLUSTER_MAX_ZOOM, clusterPoints } from "../map-clusters";

const RADIUS_PX = 26;

const near = [
  { key: "a", latlng: { lat: 32.78, lng: -96.8 }, item: "a" },
  { key: "b", latlng: { lat: 32.8, lng: -96.75 }, item: "b" },
];
const far = { key: "c", latlng: { lat: 41.88, lng: -87.63 }, item: "c" };

describe("clusterPoints", () => {
  it("folds points that would overlap at a national zoom into one count", () => {
    const clusters = clusterPoints([...near, far], 4, RADIUS_PX);

    const folded = clusters.find((c) => c.items.length > 1);
    expect(folded?.items.sort()).toEqual(["a", "b"]);
    expect(folded?.expansionZoom).toBeGreaterThan(4);
    expect(folded?.expansionZoom).toBeLessThanOrEqual(CLUSTER_MAX_ZOOM);
    expect(clusters.find((c) => c.key === "c")).toEqual({
      key: "c",
      latlng: far.latlng,
      items: ["c"],
      expansionZoom: null,
    });
  });

  it("draws the same points apart once zoomed in far enough", () => {
    const clusters = clusterPoints(near, 14, RADIUS_PX);

    expect(clusters.map((c) => c.items)).toEqual(expect.arrayContaining([["a"], ["b"]]));
    expect(clusters).toHaveLength(2);
  });

  it("marks loads at one place as a cluster no zoom will split", () => {
    const yard = { lat: 34.06, lng: -117.65 };
    const clusters = clusterPoints(
      ["x", "y", "z"].map((key) => ({ key, latlng: yard, item: key })),
      18,
      RADIUS_PX,
    );

    expect(clusters).toHaveLength(1);
    expect(clusters[0].items.sort()).toEqual(["x", "y", "z"]);
    expect(clusters[0].expansionZoom).toBeNull();
  });

  it("returns nothing for no points", () => {
    expect(clusterPoints([], 4, RADIUS_PX)).toEqual([]);
  });
});
