import { MetaTags } from "@/components/meta-tags";
import { SuspenseLoader } from "@/components/ui/component-loader";
import { AdvancedMarker, APIProvider, Map } from "@vis.gl/react-google-maps";

export function Shipment() {
  const position = { lat: 53.54992, lng: 10.00678 };

  return (
    <>
      <MetaTags title="Shipments" description="Shipments" />
      <SuspenseLoader>
        <div className="grid grid-cols-12 gap-4 size-full">
          <div className="col-span-3">
            <div className="h-full w-full bg-muted" />
          </div>
          <div className="col-span-9">
            <APIProvider apiKey={import.meta.env.VITE_GOOGLE_MAPS_API_KEY}>
              <Map
                defaultCenter={position}
                defaultZoom={10}
                mapId="SHIPMENT_MAP"
              >
                <AdvancedMarker position={position} />
              </Map>
            </APIProvider>
          </div>
        </div>
      </SuspenseLoader>
    </>
  );
}
