"use no memo";

import { MetaTags } from "@/components/meta-tags";
import { SuspenseLoader } from "@/components/ui/component-loader";
import { API_URL } from "@/constants/env";
import { useDebounce } from "@/hooks/use-debounce";
import { ShipmentFilterSchema } from "@/lib/schemas/shipment-filter-schema";
import { LimitOffsetResponse } from "@/types/server";
import { type Shipment as ShipmentResponse } from "@/types/shipment";
import { useQuery } from "@tanstack/react-query";
import { APIProvider, Map } from "@vis.gl/react-google-maps";
import { parseAsInteger, useQueryState } from "nuqs";
import { useCallback, useEffect, useTransition } from "react";
import { FormProvider, useForm } from "react-hook-form";
import ShipmentSidebar from "./_components/sidebar/shipment-sidebar";

const DEFAULT_PAGE_SIZE = 10;
const PAGE_SIZE_OPTIONS = [10, 25, 50] as const;
const SEARCH_DEBOUNCE_TIME = 300;

const searchParams = {
  page: parseAsInteger.withDefault(1),
  pageSize: parseAsInteger.withDefault(DEFAULT_PAGE_SIZE),
};

type ShipmentQueryParams = {
  pageIndex: number;
  pageSize: number;
  expandShipmentDetails: boolean;
  query?: string;
};

function fetchShipments(queryParams: ShipmentQueryParams) {
  const fetchURL = new URL(`${API_URL}/shipments/`);
  fetchURL.searchParams.set("limit", queryParams.pageSize.toString());
  fetchURL.searchParams.set(
    "offset",
    (queryParams.pageIndex * queryParams.pageSize).toString(),
  );
  fetchURL.searchParams.set(
    "expandShipmentDetails",
    queryParams.expandShipmentDetails.toString(),
  );

  // Append the search params
  if (queryParams.query) {
    fetchURL.searchParams.set("query", queryParams.query);
  }

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
  const [isTransitioning, startTransition] = useTransition();

  const [page, setPage] = useQueryState(
    "page",
    searchParams.page.withOptions({
      startTransition,
      shallow: false,
    }),
  );

  const [pageSize, setPageSize] = useQueryState(
    "pageSize",
    searchParams.pageSize.withOptions({
      startTransition,
      shallow: false,
    }),
  );

  const form = useForm<ShipmentFilterSchema>({
    defaultValues: {
      search: undefined,
      status: undefined,
    },
  });

  // get the search value from the form values
  const queryValue = form.watch("search");
  const debouncedQueryValue = useDebounce(queryValue, SEARCH_DEBOUNCE_TIME);

  // Reset to the first page when search value changes
  useEffect(() => {
    if (page !== 1) {
      startTransition(() => {
        setPage(1);
      });
    }
  }, [debouncedQueryValue, page, setPage, startTransition]);

  const { data, isLoading } = fetchShipments({
    pageIndex: (page ?? 1) - 1,
    pageSize: pageSize ?? DEFAULT_PAGE_SIZE,
    expandShipmentDetails: true,
    query: debouncedQueryValue,
  });

  const handlePageChange = useCallback(
    (page: number) => {
      startTransition(() => {
        setPage(page);
      });
    },
    [setPage, startTransition],
  );

  const handlePageSizeChange = useCallback(
    (pageSize: number) => {
      startTransition(() => {
        setPage(1);
        setPageSize(pageSize);
      });
    },
    [setPage, setPageSize, startTransition],
  );

  return (
    <>
      <MetaTags title="Shipments" description="Shipments" />
      <SuspenseLoader>
        <FormProvider {...form}>
          <div className="flex gap-4 h-[calc(100vh-theme(spacing.16))]">
            <div className="w-[420px] flex-shrink-0">
              <SuspenseLoader>
                <ShipmentSidebar
                  shipments={data?.results || []}
                  totalCount={data?.count || 0}
                  page={page ?? 1}
                  pageSize={pageSize ?? DEFAULT_PAGE_SIZE}
                  onPageChange={handlePageChange}
                  onPageSizeChange={handlePageSizeChange}
                  pageSizeOptions={PAGE_SIZE_OPTIONS}
                  isLoading={isLoading || isTransitioning}
                />
              </SuspenseLoader>
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
