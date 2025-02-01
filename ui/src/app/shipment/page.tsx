"use no memo";

import { InputField } from "@/components/fields/input-field";
import { SelectField } from "@/components/fields/select-field";
import { MetaTags } from "@/components/meta-tags";
import { Button } from "@/components/ui/button";
import { SuspenseLoader } from "@/components/ui/component-loader";
import { Icon } from "@/components/ui/icons";
import { API_URL } from "@/constants/env";
import { statusChoices } from "@/lib/choices";
import { ShipmentFilterSchema } from "@/lib/schemas/shipment-filter-schema";
import { LimitOffsetResponse } from "@/types/server";
import { type Shipment as ShipmentResponse } from "@/types/shipment";
import { faFilter, faSearch } from "@fortawesome/pro-regular-svg-icons";
import { useQuery } from "@tanstack/react-query";
import { APIProvider, Map } from "@vis.gl/react-google-maps";
import { useForm } from "react-hook-form";
import { FilterOptions } from "./_components/sidebar/filter-options";

type ShipmentQueryParams = {
  pageIndex: number;
  pageSize: number;
  includeMoveDetails: boolean;
  includeCommodityDetails: boolean;
  includeStopDetails: boolean;
  includeCustomerDetails: boolean;
};

function fetchShipments(queryParams: ShipmentQueryParams) {
  const fetchURL = new URL(`${API_URL}/shipments/`);
  fetchURL.searchParams.set("pageIndex", queryParams.pageIndex.toString());
  fetchURL.searchParams.set("pageSize", queryParams.pageSize.toString());
  fetchURL.searchParams.set(
    "includeMoveDetails",
    queryParams.includeMoveDetails.toString(),
  );
  fetchURL.searchParams.set(
    "includeCommodityDetails",
    queryParams.includeCommodityDetails.toString(),
  );
  fetchURL.searchParams.set(
    "includeStopDetails",
    queryParams.includeStopDetails.toString(),
  );
  fetchURL.searchParams.set(
    "includeCustomerDetails",
    queryParams.includeCustomerDetails.toString(),
  );

  return useQuery<LimitOffsetResponse<ShipmentResponse>>({
    queryKey: ["shipments", fetchURL.href, queryParams],
    queryFn: async () => {
      const response = await fetch(fetchURL.href, {
        credentials: "include",
      });
      return response.json();
    },
  });
}

export function Shipment() {
  const center = { lat: 39.8283, lng: -98.5795 }; // Center of continental US

  const { data: shipments } = fetchShipments({
    pageIndex: 0,
    pageSize: 10,
    includeMoveDetails: true,
    includeCommodityDetails: false,
    includeStopDetails: true,
    includeCustomerDetails: true,
  });

  return (
    <>
      <MetaTags title="Shipments" description="Shipments" />
      <SuspenseLoader>
        <div className="flex gap-4 size-full">
          <div className="w-full max-w-[420px] flex-shrink-0">
            <ShipmentSidebar />
          </div>
          <div className="flex-grow rounded-md border overflow-hidden mb-2">
            <APIProvider apiKey={import.meta.env.VITE_GOOGLE_MAPS_API_KEY}>
              <Map
                defaultCenter={center}
                defaultZoom={4}
                gestureHandling="greedy"
                mapId="SHIPMENT_MAP"
                streetViewControl={false}
                className="w-full h-full min-h-[600px]"
              />
            </APIProvider>
          </div>
        </div>
      </SuspenseLoader>
    </>
  );
}

function ShipmentSidebar() {
  const form = useForm<ShipmentFilterSchema>();
  const { control } = form;

  return (
    <div className="flex flex-col gap-2 bg-sidebar rounded-md p-2 border border-sidebar-border w-full">
      <FilterOptions />
      <div className="flex flex-row gap-2 justify-start mb-1">
        <InputField
          control={control}
          name="search"
          placeholder="Search"
          className="h-7 w-[250px]"
          icon={
            <Icon icon={faSearch} className="size-3.5 text-muted-foreground" />
          }
        />
        <SelectField
          control={control}
          name="status"
          placeholder="Status"
          className="h-7 w-30"
          isClearable
          options={statusChoices}
        />
        <Button
          variant="outline"
          size="icon"
          className="border-muted-foreground/20 bg-muted border"
        >
          <Icon icon={faFilter} className="size-3.5" />
        </Button>
      </div>
    </div>
  );
}
