import { useDebounce } from "@/hooks/use-debounce";
import { type ShipmentFilterSchema } from "@/lib/schemas/shipment-filter-schema";
import { ShipmentProvider } from "@/lib/shipment/shipment-context";
import { type Shipment as ShipmentResponse } from "@/types/shipment";
import { parseAsInteger, parseAsString, useQueryState } from "nuqs";
import {
  lazy,
  Suspense,
  useCallback,
  useEffect,
  useMemo,
  useTransition,
} from "react";
import { useFormContext } from "react-hook-form";
import { useShipmentDetails, useShipments } from "../../queries/shipment";
import { ShipmentDetailsSkeleton } from "./details/shipment-details-skeleton";
import { ShipmentList } from "./shipment-list";
import { ShipmentPagination } from "./shipment-sidebar-pagination";

// Components
const ShipmentDetails = lazy(() => import("./details/shipment-details"));

const DEFAULT_PAGE_SIZE = 10;
const PAGE_SIZE_OPTIONS = [10, 25, 50] as const;
const SEARCH_DEBOUNCE_TIME = 500;

const searchParams = {
  page: parseAsInteger.withDefault(1),
  pageSize: parseAsInteger.withDefault(DEFAULT_PAGE_SIZE),
  selectedShipmentId: parseAsString.withDefault(""),
  mode: parseAsString.withDefault("list"),
};

export function ShipmentSidebar() {
  const [isTransitioning, startTransition] = useTransition();
  const form = useFormContext<ShipmentFilterSchema>();

  /* Query States */
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

  const [selectedShipmentId, setSelectedShipmentId] = useQueryState(
    "selectedShipmentId",
    searchParams.selectedShipmentId.withOptions({
      startTransition,
      shallow: false,
    }),
  );

  // get the search value from the form values
  const queryValue = form.watch("search");
  const debouncedQueryValue = useDebounce(queryValue, SEARCH_DEBOUNCE_TIME);

  const shipmentsQuery = useShipments({
    pageIndex: (page ?? 1) - 1,
    pageSize: pageSize ?? DEFAULT_PAGE_SIZE,
    expandShipmentDetails: true,
    query: debouncedQueryValue,
  });

  const shipmentDetails = useShipmentDetails({
    shipmentId: selectedShipmentId ?? "",
  });

  const displayData = useMemo(
    () =>
      shipmentsQuery.isLoading
        ? (Array.from({ length: pageSize }, () => undefined) as (
            | ShipmentResponse
            | undefined
          )[])
        : shipmentsQuery.data?.results,
    [shipmentsQuery.data?.results, shipmentsQuery.isLoading, pageSize],
  );

  // Reset to the first page when search value changes
  useEffect(() => {
    if (page !== 1) {
      startTransition(() => {
        setPage(1);
      });
    }
  }, [debouncedQueryValue, page, setPage, startTransition]);

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

  const handleShipmentSelection = useCallback(
    (shipmentId: string) => {
      startTransition(() => {
        setSelectedShipmentId(shipmentId);
      });
    },
    [setSelectedShipmentId, startTransition],
  );

  const handleBack = () => {
    handleShipmentSelection("");
  };

  return (
    <div className="flex flex-col h-full bg-sidebar rounded-md border border-sidebar-border">
      {selectedShipmentId ? (
        <ShipmentProvider
          initialShipment={shipmentDetails.data}
          isLoading={shipmentDetails.isLoading}
        >
          <Suspense fallback={<ShipmentDetailsSkeleton />}>
            <ShipmentDetails
              selectedShipment={shipmentDetails.data}
              isLoading={shipmentDetails.isLoading}
              onBack={handleBack}
            />
          </Suspense>
        </ShipmentProvider>
      ) : (
        <ShipmentList
          displayData={displayData ?? []}
          isLoading={shipmentsQuery.isLoading || isTransitioning}
          selectedShipmentId={selectedShipmentId}
          onShipmentSelect={handleShipmentSelection}
          inputValue={debouncedQueryValue}
        />
      )}

      {!selectedShipmentId && (
        <ShipmentPagination
          totalCount={shipmentsQuery.data?.count || 0}
          page={page}
          pageSize={pageSize}
          onPageChange={handlePageChange}
          onPageSizeChange={handlePageSizeChange}
          pageSizeOptions={PAGE_SIZE_OPTIONS}
          isLoading={shipmentsQuery.isLoading || isTransitioning}
        />
      )}
    </div>
  );
}
