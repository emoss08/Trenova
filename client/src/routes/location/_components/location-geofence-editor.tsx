import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { useMapId } from "@/hooks/use-map-id";
import { locationGeofenceTypeChoices } from "@/lib/choices";
import { cn } from "@/lib/utils";
import { DEFAULT_ZOOM, US_CENTER } from "@/lib/constants";
import { queries } from "@/lib/queries";
import type { Location, LocationGeofenceType, LocationGeofenceVertex } from "@/types/location";
import { useQuery } from "@tanstack/react-query";
import {
  AdvancedMarker,
  APIProvider,
  Circle,
  Map,
  Polygon,
  Rectangle,
  useMap,
} from "@vis.gl/react-google-maps";
import { useCallback, useEffect, useMemo } from "react";
import { useFormContext, useFormState, useWatch, type Path, type PathValue } from "react-hook-form";

const DEFAULT_GEOFENCE_RADIUS_METERS = 250;
const SHAPE_COLORS = {
  stroke: "#2563eb",
  fill: "#3b82f6",
  center: "#1d4ed8",
} as const;

function useLocationGeofence() {
  const { control, setValue } = useFormContext<Location>();
  const { errors } = useFormState<Location>({
    control,
    name: ["geofenceType", "geofenceRadiusMeters", "geofenceVertices", "latitude", "longitude"],
  });
  const [geofenceType, geofenceRadiusMeters, geofenceVertices, latitude, longitude] = useWatch({
    control,
    name: ["geofenceType", "geofenceRadiusMeters", "geofenceVertices", "latitude", "longitude"],
  });

  const center = useMemo(
    () => resolveCenter(latitude ?? null, longitude ?? null, geofenceVertices ?? []),
    [geofenceVertices, latitude, longitude],
  );
  const rectangleBounds = useMemo(
    () => (geofenceType === "rectangle" ? toBounds(geofenceVertices ?? []) : null),
    [geofenceType, geofenceVertices],
  );
  const polygonPaths = useMemo(
    () => (geofenceType === "draw" ? (geofenceVertices ?? []).map(toLatLng) : []),
    [geofenceType, geofenceVertices],
  );
  const errorMessages = useMemo(
    () =>
      [
        readFieldError(errors.geofenceType),
        readFieldError(errors.geofenceRadiusMeters),
        readFieldError(errors.geofenceVertices),
        readFieldError(errors.latitude),
        readFieldError(errors.longitude),
      ].filter((message): message is string => Boolean(message)),
    [errors],
  );

  const setFieldValue = useCallback(
    <TField extends Path<Location>>(field: TField, value: PathValue<Location, TField>) => {
      setValue(field, value, {
        shouldDirty: true,
        shouldValidate: true,
      });
    },
    [setValue],
  );

  const updateCenter = useCallback(
    (nextCenter: google.maps.LatLngLiteral) => {
      setFieldValue("latitude", nextCenter.lat);
      setFieldValue("longitude", nextCenter.lng);
    },
    [setFieldValue],
  );

  const updateVertices = useCallback(
    (nextVertices: LocationGeofenceVertex[]) => {
      const normalized = normalizeVertices(nextVertices);
      setFieldValue("geofenceVertices", normalized);

      const nextCenter = toBoundsCenter(normalized);
      if (nextCenter) {
        updateCenter(nextCenter);
      }
    },
    [setFieldValue, updateCenter],
  );

  const handleTypeChange = useCallback(
    (value: string) => {
      const nextType = value as LocationGeofenceType;
      setFieldValue("geofenceType", nextType);

      switch (nextType) {
        case "auto":
          setFieldValue("geofenceRadiusMeters", DEFAULT_GEOFENCE_RADIUS_METERS);
          setFieldValue("geofenceVertices", []);
          break;
        case "circle":
          setFieldValue(
            "geofenceRadiusMeters",
            geofenceRadiusMeters ?? DEFAULT_GEOFENCE_RADIUS_METERS,
          );
          setFieldValue("geofenceVertices", []);
          break;
        case "rectangle":
          setFieldValue("geofenceRadiusMeters", null);
          updateVertices(
            toBoundsVertices(geofenceVertices ?? []) ?? buildRectangleVertices(center),
          );
          break;
        case "draw":
          setFieldValue("geofenceRadiusMeters", null);
          updateVertices(
            (geofenceVertices?.length ?? 0) >= 3
              ? (geofenceVertices ?? [])
              : buildPolygonVertices(center),
          );
          break;
      }
    },
    [center, geofenceRadiusMeters, geofenceVertices, setFieldValue, updateVertices],
  );

  useEffect(() => {
    if (geofenceType === "rectangle" && (geofenceVertices?.length ?? 0) === 0) {
      updateVertices(buildRectangleVertices(center));
    }

    if (geofenceType === "draw" && (geofenceVertices?.length ?? 0) === 0) {
      updateVertices(buildPolygonVertices(center));
    }
  }, [center, geofenceType, geofenceVertices, updateVertices]);

  return {
    geofenceType,
    geofenceRadiusMeters,
    latitude,
    longitude,
    center,
    rectangleBounds,
    polygonPaths,
    errorMessages,
    setFieldValue,
    updateCenter,
    updateVertices,
    handleTypeChange,
  };
}

export function LocationGeofenceControls({ className }: { className?: string }) {
  const { geofenceType, handleTypeChange } = useLocationGeofence();
  const activeChoice = useMemo(
    () => locationGeofenceTypeChoices.find((choice) => choice.value === geofenceType),
    [geofenceType],
  );

  return (
    <div className={cn("space-y-2", className)}>
      <Tabs value={geofenceType} onValueChange={handleTypeChange} className="gap-3">
        <TabsList className="grid w-full grid-cols-2 md:grid-cols-4">
          {locationGeofenceTypeChoices.map((choice) => (
            <TabsTrigger key={choice.value} value={choice.value}>
              {choice.label}
            </TabsTrigger>
          ))}
        </TabsList>
      </Tabs>
      <p className="text-xs text-muted-foreground">
        {describeMode(activeChoice?.value ?? "auto")}
      </p>
    </div>
  );
}

export function LocationGeofenceMap({ className }: { className?: string }) {
  const mapId = useMapId();
  const {
    geofenceType,
    geofenceRadiusMeters,
    latitude,
    longitude,
    center,
    rectangleBounds,
    polygonPaths,
    errorMessages,
    setFieldValue,
    updateCenter,
    updateVertices,
  } = useLocationGeofence();

  const googleMapsQuery = useQuery({
    ...queries.integration.runtimeConfig("GoogleMaps"),
    staleTime: 5 * 60 * 1000,
  });

  return (
    <div className={cn("relative h-full w-full overflow-hidden bg-background", className)}>
      {googleMapsQuery.isLoading ? (
        <div className="flex h-full items-center justify-center text-sm text-muted-foreground">
          Loading map editor...
        </div>
      ) : !googleMapsQuery.data?.apiKey ? (
        <div className="flex h-full items-center justify-center px-6 text-center text-sm text-muted-foreground">
          Google Maps is not configured for this environment, so the geofence editor cannot be
          displayed.
        </div>
      ) : (
        <APIProvider apiKey={googleMapsQuery.data.apiKey}>
          <Map
            mapId={mapId}
            defaultCenter={center}
            defaultZoom={latitude != null && longitude != null ? 15 : DEFAULT_ZOOM}
            gestureHandling="greedy"
            disableDefaultUI
            onClick={({ detail }) => {
              if (!detail.latLng) {
                return;
              }

              if (geofenceType === "auto" || geofenceType === "circle") {
                updateCenter(detail.latLng);
              }
            }}
            className="h-full w-full"
          >
            <MapCenterSync center={center} />
            {(geofenceType === "auto" || geofenceType === "circle") && (
              <>
                <AdvancedMarker
                  position={center}
                  draggable
                  onDragEnd={(event) => {
                    if (event.latLng) {
                      updateCenter({
                        lat: event.latLng.lat(),
                        lng: event.latLng.lng(),
                      });
                    }
                  }}
                >
                  <div
                    className="size-3 rounded-full border-2 border-white shadow-md ring-1 ring-black/20"
                    style={{ backgroundColor: SHAPE_COLORS.center }}
                  />
                </AdvancedMarker>
                <Circle
                  center={center}
                  radius={geofenceRadiusMeters ?? DEFAULT_GEOFENCE_RADIUS_METERS}
                  editable={geofenceType === "circle"}
                  draggable={geofenceType === "circle"}
                  onCenterChanged={(nextCenter) => {
                    if (nextCenter) {
                      updateCenter({ lat: nextCenter.lat(), lng: nextCenter.lng() });
                    }
                  }}
                  onRadiusChanged={(nextRadius) => {
                    if (geofenceType === "circle") {
                      setFieldValue(
                        "geofenceRadiusMeters",
                        Math.max(1, Math.round(nextRadius)),
                      );
                    }
                  }}
                  strokeColor={SHAPE_COLORS.stroke}
                  strokeOpacity={0.95}
                  strokeWeight={2}
                  fillColor={SHAPE_COLORS.fill}
                  fillOpacity={0.24}
                />
              </>
            )}
            {geofenceType === "rectangle" && rectangleBounds && (
              <Rectangle
                bounds={rectangleBounds}
                editable
                draggable
                onBoundsChanged={(bounds) => {
                  if (!bounds) {
                    return;
                  }

                  updateVertices(boundsToVertices(bounds));
                }}
                strokeColor={SHAPE_COLORS.stroke}
                strokeOpacity={0.95}
                strokeWeight={2}
                fillColor={SHAPE_COLORS.fill}
                fillOpacity={0.18}
              />
            )}
            {geofenceType === "draw" && polygonPaths.length >= 3 && (
              <Polygon
                paths={polygonPaths}
                editable
                draggable
                onPathsChanged={(paths) => {
                  const nextPath = paths[0] ?? [];
                  updateVertices(
                    nextPath.map((point) => ({
                      latitude: point.lat(),
                      longitude: point.lng(),
                    })),
                  );
                }}
                strokeColor={SHAPE_COLORS.stroke}
                strokeOpacity={0.95}
                strokeWeight={2}
                fillColor={SHAPE_COLORS.fill}
                fillOpacity={0.18}
              />
            )}
          </Map>
        </APIProvider>
      )}

      {errorMessages.length > 0 && (
        <div className="pointer-events-none absolute top-3 left-3 max-w-sm rounded-md border border-red-500/30 bg-red-500/10 px-3 py-2 text-xs text-red-600 shadow-sm backdrop-blur">
          {errorMessages[0]}
        </div>
      )}
    </div>
  );
}

function MapCenterSync({ center }: { center: google.maps.LatLngLiteral }) {
  const map = useMap();

  useEffect(() => {
    if (!map) {
      return;
    }

    map.panTo(center);
  }, [center, map]);

  return null;
}

function resolveCenter(
  latitude: number | null,
  longitude: number | null,
  vertices: LocationGeofenceVertex[],
): google.maps.LatLngLiteral {
  if (latitude != null && longitude != null) {
    return { lat: latitude, lng: longitude };
  }

  return toBoundsCenter(vertices) ?? US_CENTER;
}

function buildRectangleVertices(
  center: google.maps.LatLngLiteral,
  sizeMeters = DEFAULT_GEOFENCE_RADIUS_METERS,
): LocationGeofenceVertex[] {
  const latDelta = metersToLatitude(sizeMeters);
  const lngDelta = metersToLongitude(sizeMeters, center.lat);

  return [
    { latitude: center.lat - latDelta, longitude: center.lng - lngDelta },
    { latitude: center.lat - latDelta, longitude: center.lng + lngDelta },
    { latitude: center.lat + latDelta, longitude: center.lng + lngDelta },
    { latitude: center.lat + latDelta, longitude: center.lng - lngDelta },
  ];
}

function buildPolygonVertices(
  center: google.maps.LatLngLiteral,
  sizeMeters = DEFAULT_GEOFENCE_RADIUS_METERS,
): LocationGeofenceVertex[] {
  return buildRectangleVertices(center, sizeMeters);
}

function normalizeVertices(vertices: LocationGeofenceVertex[]): LocationGeofenceVertex[] {
  if (vertices.length <= 1) {
    return vertices;
  }

  const normalized = [...vertices];
  const first = normalized[0];
  const last = normalized.at(-1);

  if (last && first.latitude === last.latitude && first.longitude === last.longitude) {
    normalized.pop();
  }

  return normalized;
}

function toLatLng(vertex: LocationGeofenceVertex): google.maps.LatLngLiteral {
  return { lat: vertex.latitude, lng: vertex.longitude };
}

function toBounds(vertices: LocationGeofenceVertex[]): google.maps.LatLngBoundsLiteral | null {
  const normalized = normalizeVertices(vertices);
  if (normalized.length === 0) {
    return null;
  }

  let north = normalized[0].latitude;
  let south = normalized[0].latitude;
  let east = normalized[0].longitude;
  let west = normalized[0].longitude;

  for (const vertex of normalized.slice(1)) {
    north = Math.max(north, vertex.latitude);
    south = Math.min(south, vertex.latitude);
    east = Math.max(east, vertex.longitude);
    west = Math.min(west, vertex.longitude);
  }

  return { north, south, east, west };
}

function toBoundsCenter(vertices: LocationGeofenceVertex[]): google.maps.LatLngLiteral | null {
  const bounds = toBounds(vertices);
  if (!bounds) {
    return null;
  }

  return {
    lat: (bounds.north + bounds.south) / 2,
    lng: (bounds.east + bounds.west) / 2,
  };
}

function toBoundsVertices(vertices: LocationGeofenceVertex[]): LocationGeofenceVertex[] | null {
  const bounds = toBounds(vertices);
  if (!bounds) {
    return null;
  }

  return [
    { latitude: bounds.south, longitude: bounds.west },
    { latitude: bounds.south, longitude: bounds.east },
    { latitude: bounds.north, longitude: bounds.east },
    { latitude: bounds.north, longitude: bounds.west },
  ];
}

function boundsToVertices(bounds: google.maps.LatLngBounds): LocationGeofenceVertex[] {
  const northEast = bounds.getNorthEast();
  const southWest = bounds.getSouthWest();

  return [
    { latitude: southWest.lat(), longitude: southWest.lng() },
    { latitude: southWest.lat(), longitude: northEast.lng() },
    { latitude: northEast.lat(), longitude: northEast.lng() },
    { latitude: northEast.lat(), longitude: southWest.lng() },
  ];
}

function metersToLatitude(meters: number) {
  return meters / 111_320;
}

function metersToLongitude(meters: number, latitude: number) {
  const cosLatitude = Math.cos((latitude * Math.PI) / 180);
  return meters / (111_320 * Math.max(Math.abs(cosLatitude), 0.0001));
}

function readFieldError(error: unknown) {
  if (!error || typeof error !== "object") {
    return null;
  }

  if ("message" in error && typeof error.message === "string") {
    return error.message;
  }

  return null;
}

function describeMode(mode: LocationGeofenceType) {
  switch (mode) {
    case "auto":
      return "Auto uses the location point with the default 250 meter operating radius.";
    case "circle":
      return "Circle lets you control the radius while keeping the geofence centered on the map pin.";
    case "rectangle":
      return "Rectangle is useful for facilities with a simple box-shaped footprint.";
    case "draw":
      return "Draw unlocks a freeform polygon for yards, campuses, and irregular site boundaries.";
  }
}
