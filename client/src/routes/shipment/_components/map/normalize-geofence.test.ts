import { describe, expect, it } from "vitest";
import { normalizeGeofence, type GeofenceInput } from "./normalize-geofence";

const baseInput = (overrides: Partial<GeofenceInput> = {}): GeofenceInput => ({
  id: "loc_1",
  locationName: "Test",
  geofenceType: "circle",
  latitude: 35.0,
  longitude: -97.0,
  geofenceRadiusMeters: 500,
  geofenceVertices: [],
  ...overrides,
});

describe("normalizeGeofence", () => {
  it("builds a circle from lat/lng + radius", () => {
    const result = normalizeGeofence(baseInput());
    expect(result).toMatchObject({
      kind: "circle",
      center: { lat: 35.0, lng: -97.0 },
      radiusMeters: 500,
      sourceType: "circle",
    });
  });

  it("builds a polygon from rectangle vertices", () => {
    const result = normalizeGeofence(
      baseInput({
        geofenceType: "rectangle",
        geofenceVertices: [
          { latitude: 0, longitude: 0 },
          { latitude: 0, longitude: 1 },
          { latitude: 1, longitude: 1 },
          { latitude: 1, longitude: 0 },
        ],
      }),
    );
    expect(result?.kind).toBe("polygon");
    if (result?.kind === "polygon") {
      expect(result.path).toHaveLength(4);
    }
  });

  it("builds a polygon from draw vertices", () => {
    const result = normalizeGeofence(
      baseInput({
        geofenceType: "draw",
        geofenceVertices: [
          { latitude: 0, longitude: 0 },
          { latitude: 0, longitude: 1 },
          { latitude: 1, longitude: 1 },
        ],
      }),
    );
    expect(result?.kind).toBe("polygon");
  });

  it("auto resolves to polygon when vertices present", () => {
    const result = normalizeGeofence(
      baseInput({
        geofenceType: "auto",
        geofenceVertices: [
          { latitude: 0, longitude: 0 },
          { latitude: 0, longitude: 1 },
          { latitude: 1, longitude: 1 },
        ],
      }),
    );
    expect(result?.kind).toBe("polygon");
  });

  it("auto resolves to circle when no vertices and lat/lng/radius present", () => {
    const result = normalizeGeofence(
      baseInput({ geofenceType: "auto", geofenceVertices: [] }),
    );
    expect(result?.kind).toBe("circle");
  });

  it("returns null when circle has no radius", () => {
    expect(
      normalizeGeofence(
        baseInput({ geofenceType: "circle", geofenceRadiusMeters: null }),
      ),
    ).toBeNull();
  });

  it("returns null when polygon has fewer than 3 valid vertices", () => {
    expect(
      normalizeGeofence(
        baseInput({
          geofenceType: "draw",
          geofenceVertices: [
            { latitude: 0, longitude: 0 },
            { latitude: 1, longitude: 1 },
          ],
        }),
      ),
    ).toBeNull();
  });

  it("rejects NaN coordinates", () => {
    expect(
      normalizeGeofence(
        baseInput({ geofenceType: "circle", latitude: Number.NaN }),
      ),
    ).toBeNull();
  });

  it("filters out malformed vertices in polygon mode", () => {
    const result = normalizeGeofence(
      baseInput({
        geofenceType: "draw",
        geofenceVertices: [
          { latitude: 0, longitude: 0 },
          { latitude: Number.NaN, longitude: 0 },
          { latitude: 1, longitude: 1 },
          { latitude: 0, longitude: 1 },
        ],
      }),
    );
    expect(result?.kind).toBe("polygon");
    if (result?.kind === "polygon") {
      expect(result.path).toHaveLength(3);
    }
  });

  it("returns null when only 2 valid vertices remain after filtering", () => {
    expect(
      normalizeGeofence(
        baseInput({
          geofenceType: "draw",
          geofenceVertices: [
            { latitude: 0, longitude: 0 },
            { latitude: Number.NaN, longitude: 0 },
            { latitude: 1, longitude: 1 },
          ],
        }),
      ),
    ).toBeNull();
  });

  it("returns null when circle radius is zero or negative", () => {
    expect(
      normalizeGeofence(baseInput({ geofenceType: "circle", geofenceRadiusMeters: 0 })),
    ).toBeNull();
    expect(
      normalizeGeofence(baseInput({ geofenceType: "circle", geofenceRadiusMeters: -100 })),
    ).toBeNull();
  });
});
