import { useMapId } from "@/hooks/use-map-id";
import { queries } from "@/lib/queries";
import { useQuery } from "@tanstack/react-query";
import { AdvancedMarker, APIProvider, Map } from "@vis.gl/react-google-maps";

interface LazyMapProps {
  apiKey: string;
  position: { lat: number; lng: number };
}

export default function LazyMap({ apiKey, position }: LazyMapProps) {
  const mapId = useMapId();

  return (
    <APIProvider apiKey={apiKey}>
      <Map defaultCenter={position} defaultZoom={17} mapId={mapId}>
        <AdvancedMarker position={position} />
      </Map>
    </APIProvider>
  );
}

export function LazyMapWithKey({ position }: Pick<LazyMapProps, "position">) {
  const { data: googleMapsData } = useQuery({
    ...queries.integration.runtimeConfig("GoogleMaps"),
  });

  if (!googleMapsData?.apiKey) {
    console.error("Google Maps API key not found");
    return null;
  }

  return <LazyMap apiKey={googleMapsData?.apiKey ?? ""} position={position} />;
}
