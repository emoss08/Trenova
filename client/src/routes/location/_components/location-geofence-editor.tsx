import { NumberField } from "@/components/fields/number-field";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { useMapId } from "@/hooks/use-map-id";
import { locationGeofenceTypeChoices } from "@/lib/choices";
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
import { MapPinIcon, RefreshCcwIcon } from "lucide-react";
import { useCallback, useEffect, useMemo } from "react";
import { useFormContext, useFormState, useWatch, type Path, type PathValue } from "react-hook-form";

const DEFAULT_GEOFENCE_RADIUS_METERS = 250;
const SHAPE_COLORS = {
  stroke: "#0f766e",
  fill: "#14b8a61f",
  center: "#0f766e",
} as const;

export function LocationGeofenceEditor() {
  const mapId = useMapId();
  const { control, setValue } = useFormContext<Location>();
  const { errors } = useFormState<Location>({
    control,
    name: ["geofenceType", "geofenceRadiusMeters", "geofenceVertices", "latitude", "longitude"],
  });
  const [geofenceType, geofenceRadiusMeters, geofenceVertices, latitude, longitude] = useWatch({
    control,
    name: ["geofenceType", "geofenceRadiusMeters", "geofenceVertices", "latitude", "longitude"],
  });

  const googleMapsQuery = useQuery({
    ...queries.integration.runtimeConfig("GoogleMaps"),
    staleTime: 5 * 60 * 1000,
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
  const activeChoice = useMemo(
    () => locationGeofenceTypeChoices.find((choice) => choice.value === geofenceType),
    [geofenceType],
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

  const resetManualShape = useCallback(
    (nextType: LocationGeofenceType) => {
      const size = geofenceRadiusMeters ?? DEFAULT_GEOFENCE_RADIUS_METERS;

      if (nextType === "rectangle") {
        updateVertices(buildRectangleVertices(center, size));
        return;
      }

      updateVertices(buildPolygonVertices(center, size));
    },
    [center, geofenceRadiusMeters, updateVertices],
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

  return (
    <Card className="border-dashed bg-muted/20">
      <CardHeader className="gap-2 border-b">
        <div className="flex items-start justify-between gap-3">
          <div className="space-y-1">
            <CardTitle className="flex items-center gap-2">
              <MapPinIcon className="size-4 text-muted-foreground" />
              Geofence
            </CardTitle>
            <CardDescription>
              Build the location boundary directly on the map. Use auto for a standard radius, or
              switch to a manual shape when the site needs tighter control.
            </CardDescription>
          </div>
          {(geofenceType === "rectangle" || geofenceType === "draw") && (
            <Button
              type="button"
              variant="outline"
              size="sm"
              onClick={() => resetManualShape(geofenceType)}
            >
              <RefreshCcwIcon className="size-3.5" />
              Reset Shape
            </Button>
          )}
        </div>
      </CardHeader>
      <CardContent className="space-y-4">
        <div className="space-y-2">
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

        {geofenceType === "circle" && (
          <NumberField
            control={control}
            name="geofenceRadiusMeters"
            label="Radius"
            description="Circular geofence radius in meters."
            min={1}
            step={25}
            sideText="m"
            decimalScale={0}
          />
        )}

        {errorMessages.length > 0 && (
          <div className="rounded-md border border-red-500/30 bg-red-500/10 px-3 py-2 text-xs text-red-600">
            {errorMessages[0]}
          </div>
        )}

        <div className="overflow-hidden rounded-lg border bg-background">
          {googleMapsQuery.isLoading ? (
            <div className="flex h-[320px] items-center justify-center text-sm text-muted-foreground">
              Loading map editor...
            </div>
          ) : !googleMapsQuery.data?.apiKey ? (
            <div className="flex h-[320px] items-center justify-center px-6 text-center text-sm text-muted-foreground">
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
                className="h-[320px] w-full"
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
                    />
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
        </div>

        <p className="text-xs text-muted-foreground">
          {geofenceType === "auto" || geofenceType === "circle"
            ? "Click the map or drag the pin to reposition the location. Circle mode also lets you resize the boundary."
            : geofenceType === "rectangle"
              ? "Drag the rectangle or pull its handles to match the facility footprint."
              : "Drag polygon vertices and edge handles to sketch the exact yard or campus boundary."}
        </p>
      </CardContent>
    </Card>
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
