import { useDebounce } from "@/hooks/use-debounce";
import { ShipmentProvider } from "@/hooks/use-shipment";
import { useShipmentParams } from "@/hooks/use-shipment-params";
import { type ShipmentFilterSchema } from "@/lib/schemas/shipment-filter-schema";
import { lazy, Suspense, useCallback, useEffect, useMemo } from "react";
import { useFormContext } from "react-hook-form";
import { useShipmentDetails, useShipments } from "../../queries/shipment";
import { ShipmentDetailsSkeleton } from "./details/shipment-details-skeleton";
import { ShipmentList } from "./shipment-list";
import { ShipmentPagination } from "./shipment-sidebar-pagination";

// Components
const ShipmentDetails = lazy(() => import("./details/shipment-details"));

const PAGE_SIZE_OPTIONS = [10, 25, 50] as const;
const SEARCH_DEBOUNCE_TIME = 500;

export function ShipmentSidebar() {
  const {
    page,
    pageSize,
    selectedShipmentId,
    updateParams,
    isTransitioning,
    DEFAULT_PAGE_SIZE,
  } = useShipmentParams();
  const form = useFormContext<ShipmentFilterSchema>();

  const handlePageChange = useCallback(
    (newPage: number) => {
      updateParams({ page: newPage });
    },
    [updateParams],
  );

  const handlePageSizeChange = useCallback(
    (newPageSize: number) => {
      updateParams({ pageSize: newPageSize });
    },
    [updateParams],
  );

  const handleShipmentSelection = useCallback(
    (shipmentId: string) => {
      updateParams({ selectedShipmentId: shipmentId });
    },
    [updateParams],
  );
  // get the search value from the form values
  const queryValue = form.watch("search");
  const debouncedQueryValue = useDebounce(queryValue, SEARCH_DEBOUNCE_TIME);

  const shipmentsQuery = useShipments({
    pageIndex: (page ?? 1) - 1,
    pageSize: pageSize ?? DEFAULT_PAGE_SIZE,
    expandShipmentDetails: true,
    query: debouncedQueryValue,
    enabled: !selectedShipmentId, // Only run query when no shipment is selected
  });

  const shipmentDetails = useShipmentDetails({
    shipmentId: selectedShipmentId ?? "",
    enabled: Boolean(selectedShipmentId),
  });

  // Modified displayData logic to better handle loading states
  const displayData = useMemo(() => {
    // Don't show loading state when transitioning to detail view
    if (shipmentsQuery.isLoading && !selectedShipmentId) {
      return Array.from(
        { length: pageSize ?? DEFAULT_PAGE_SIZE },
        () => undefined,
      );
    }
    return shipmentsQuery.data?.results ?? [];
  }, [
    shipmentsQuery.data?.results,
    shipmentsQuery.isLoading,
    pageSize,
    selectedShipmentId,
    DEFAULT_PAGE_SIZE,
  ]);

  useEffect(() => {
    // Only reset page when search value changes
    if (!selectedShipmentId) {
      updateParams({ page: 1 });
    }
  }, [debouncedQueryValue, updateParams, selectedShipmentId]);

  const handleBack = useCallback(() => {
    handleShipmentSelection("");
  }, [handleShipmentSelection]);

  const isDetailsLoading = selectedShipmentId
    ? shipmentDetails.isLoading
    : false;

  return (
    <div className="flex flex-col h-full bg-sidebar rounded-md border border-sidebar-border">
      {selectedShipmentId ? (
        <ShipmentProvider
          initialShipment={shipmentDetails.data}
          isLoading={isDetailsLoading}
        >
          <Suspense fallback={<ShipmentDetailsSkeleton />}>
            <ShipmentDetails
              selectedShipment={shipmentDetails.data}
              isLoading={isDetailsLoading}
              onBack={handleBack}
            />
          </Suspense>
        </ShipmentProvider>
      ) : (
        <ShipmentList
          displayData={displayData}
          isLoading={shipmentsQuery.isLoading}
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
