import type { Location } from "@trenova/shared/types/location";
import { AdvancedMarker } from "@vis.gl/react-google-maps";
import { MapPin } from "lucide-react";

export type AddressMarkerLocation = Pick<Location, "id" | "name" | "latitude" | "longitude">;

export function LocationAddressMarker({ location }: { location: AddressMarkerLocation }) {
  if (location.latitude == null || location.longitude == null) return null;
  if (!Number.isFinite(location.latitude) || !Number.isFinite(location.longitude)) return null;

  return (
    <AdvancedMarker
      position={{ lat: location.latitude, lng: location.longitude }}
      zIndex={20}
      title={location.name}
    >
      <div className="relative">
        <MapPin className="size-3 fill-info text-info-foreground" />
        <span className="absolute top-1/2 left-full ml-1 -translate-y-1/2 text-2xs font-semibold whitespace-nowrap text-foreground [-webkit-text-stroke:2px_var(--canvas)] [paint-order:stroke]">
          {location.name}
        </span>
      </div>
    </AdvancedMarker>
  );
}
