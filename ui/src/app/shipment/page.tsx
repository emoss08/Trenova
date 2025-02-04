import { MetaTags } from "@/components/meta-tags";
import { SuspenseLoader } from "@/components/ui/component-loader";
import { Skeleton } from "@/components/ui/skeleton";
import { ShipmentFilterSchema } from "@/lib/schemas/shipment-filter-schema";
import { APIProvider, Map } from "@vis.gl/react-google-maps";
import { Suspense } from "react";
import { FormProvider, useForm } from "react-hook-form";
import { ShipmentSidebar } from "./_components/sidebar/shipment-sidebar";

export function Shipment() {
  const center = { lat: 39.8283, lng: -98.5795 }; // Center of continental US
  const form = useForm<ShipmentFilterSchema>({
    defaultValues: {
      search: undefined,
      status: undefined,
    },
  });

  return (
    <>
      <MetaTags title="Shipments" description="Shipments" />
      <SuspenseLoader>
        <FormProvider {...form}>
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
        </FormProvider>
      </SuspenseLoader>
    </>
  );
}
