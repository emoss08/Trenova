import { useMapId } from "@/hooks/use-map-id";
import { AdvancedMarker, APIProvider, Map } from "@vis.gl/react-google-maps";

const ZOOM = 9;

/**
 * Where the tractor last reported itself, as a small still map with its pin:
 * it shows the place, and a dispatcher who wants to pan opens the map on the
 * board. Its own chunk, read only when Google Maps is set up and there is a
 * position to show.
 */
export function TractorMap({
  apiKey,
  latitude,
  longitude,
  label,
}: {
  apiKey: string;
  latitude: number;
  longitude: number;
  label: string;
}) {
  const mapId = useMapId();
  const position = { lat: latitude, lng: longitude };
  return (
    <APIProvider apiKey={apiKey}>
      <Map
        className="size-full"
        defaultCenter={position}
        defaultZoom={ZOOM}
        mapId={mapId}
        disableDefaultUI
        gestureHandling="none"
        keyboardShortcuts={false}
        clickableIcons={false}
        aria-label={label}
      >
        <AdvancedMarker position={position} title={label} />
      </Map>
    </APIProvider>
  );
}
