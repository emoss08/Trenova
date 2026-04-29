import type { Location } from "@/types/location";
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
        <MapPin className="size-3 fill-blue-600 text-blue-600" />
        <span className="absolute top-1/2 left-full ml-1 -translate-y-1/2 text-[10px] font-semibold whitespace-nowrap text-black [-webkit-text-stroke:2px_white] [paint-order:stroke] dark:text-white dark:[-webkit-text-stroke:2px_black]">
          {location.name}
        </span>
      </div>
    </AdvancedMarker>
  );
}
