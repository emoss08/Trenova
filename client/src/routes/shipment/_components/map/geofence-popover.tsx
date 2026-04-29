import { Separator } from "@/components/ui/separator";
import type { Location } from "@/types/location";
import { AdvancedMarker } from "@vis.gl/react-google-maps";
import { HashIcon, MapPinIcon, XIcon } from "lucide-react";
import type { NormalizedGeofence } from "./geofence-types";

function formatCoord(value: number) {
  return value.toFixed(5);
}

function geofenceAnchor(geofence: NormalizedGeofence): google.maps.LatLngLiteral {
  if (geofence.kind === "circle") return geofence.center;

  let latSum = 0;
  let lngSum = 0;
  for (const point of geofence.path) {
    latSum += point.lat;
    lngSum += point.lng;
  }
  return { lat: latSum / geofence.path.length, lng: lngSum / geofence.path.length };
}

function buildAddressLine(location: Location) {
  const stateAbbr = location.state?.abbreviation ?? "";
  const tail = [location.city, [stateAbbr, location.postalCode].filter(Boolean).join(" ")]
    .filter(Boolean)
    .join(", ");
  return tail;
}

export function GeofencePopover({
  geofence,
  location,
  onClose,
}: {
  geofence: NormalizedGeofence;
  location: Location | null;
  onClose: () => void;
}) {
  const addressTail = location ? buildAddressLine(location) : "";
  const center = geofenceAnchor(geofence);

  return (
    <AdvancedMarker position={center} zIndex={150} onClick={(e) => e.stop()}>
      <div className="relative mb-4">
        <div className="relative z-10 flex w-72 flex-col gap-2.5 rounded-lg border bg-popover p-3 text-xs text-popover-foreground shadow-lg ring-1 ring-foreground/10">
          <div className="flex items-start justify-between gap-2">
            <div className="flex min-w-0 flex-col gap-0.5">
              <span className="truncate text-sm font-semibold text-foreground">
                {location?.name ?? geofence.locationName}
              </span>
              {location?.code && (
                <span className="flex items-center gap-1 text-2xs text-muted-foreground">
                  <HashIcon className="size-3" />
                  {location.code}
                </span>
              )}
            </div>
            <button
              type="button"
              onClick={onClose}
              className="rounded text-muted-foreground hover:text-foreground"
              aria-label="Close geofence info"
            >
              <XIcon className="size-3.5" />
            </button>
          </div>

          {location?.description && (
            <p className="text-2xs leading-relaxed text-muted-foreground">{location.description}</p>
          )}

          {location && (location.addressLine1 || addressTail) && (
            <>
              <Separator />
              <div className="flex items-start gap-2">
                <MapPinIcon className="mt-0.5 size-3.5 shrink-0 text-muted-foreground" />
                <div className="flex min-w-0 flex-col text-2xs leading-relaxed text-foreground">
                  {location.addressLine1 && <span>{location.addressLine1}</span>}
                  {location.addressLine2 && (
                    <span className="text-muted-foreground">{location.addressLine2}</span>
                  )}
                  {addressTail && <span>{addressTail}</span>}
                </div>
              </div>
            </>
          )}

          {(location?.latitude != null || location?.longitude != null) && (
            <>
              <Separator />
              <div className="grid grid-cols-2 gap-2 text-2xs tabular-nums">
                <div className="flex flex-col">
                  <span className="text-muted-foreground">Latitude</span>
                  <span className="text-foreground">
                    {location.latitude != null ? formatCoord(location.latitude) : "—"}
                  </span>
                </div>
                <div className="flex flex-col">
                  <span className="text-muted-foreground">Longitude</span>
                  <span className="text-foreground">
                    {location.longitude != null ? formatCoord(location.longitude) : "—"}
                  </span>
                </div>
              </div>
            </>
          )}
        </div>
      </div>
    </AdvancedMarker>
  );
}
