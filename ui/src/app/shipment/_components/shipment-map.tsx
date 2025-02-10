import { Skeleton } from "@/components/ui/skeleton";
import { APIProvider, Map } from "@vis.gl/react-google-maps";
import { Suspense } from "react";
import { ShipmentSidebar } from "./sidebar/shipment-sidebar";

export default function ShipmentMap() {
  const center = { lat: 39.8283, lng: -98.5795 };

  return (
    <div className="flex gap-4 h-[calc(100vh-theme(spacing.16))]">
      <div className="w-[420px] flex-shrink-0">
        <Suspense fallback={<Skeleton className="size-full" />}>
          <ShipmentSidebar />
        </Suspense>
      </div>
      <div className="flex-grow rounded-md border overflow-hidden">
        <APIProvider apiKey={import.meta.env.VITE_GOOGLE_MAPS_API_KEY}>
          <Map
            defaultCenter={center}
            defaultZoom={5}
            gestureHandling="greedy"
            mapId="SHIPMENT_MAP"
            streetViewControl={false}
            className="w-full h-full"
          />
        </APIProvider>
      </div>
    </div>
  );
}
