import { ScrollArea } from "@/components/ui/scroll-area";
import { type Shipment } from "@/types/shipment";
import { lazy, memo, Suspense } from "react";
import { ShipmentNotFoundOverlay } from "../sidebar/shipment-not-found-overlay";
import { ShipmentDetailsSkeleton } from "./shipment-details-skeleton";
import { ShipmentFormHeader } from "./shipment-form-header";

// Lazy loaded components
const ShipmentDetailsHeader = lazy(() => import("./shipment-details-header"));
const ShipmentBillingDetails = lazy(() => import("./shipment-billing-details"));
const ShipmentGeneralInformation = lazy(
  () => import("./shipment-general-information"),
);
const ShipmentCommodityDetails = lazy(
  () => import("./commodity/commodity-details"),
);
const ShipmentMovesDetails = lazy(() => import("./move/move-details"));
const ShipmentServiceDetails = lazy(() => import("./shipment-service-details"));

type ShipmentDetailsProps = {
  selectedShipment?: Shipment | null;
  isLoading?: boolean;
  isError?: boolean;
  onBack: () => void;
  dimensions: {
    contentHeight: number;
    viewportHeight: number;
  };
};

// Wrap the component with memo to prevent unnecessary re-renders
export const ShipmentForm = memo(function ShipmentForm({
  isLoading,
  ...props
}: ShipmentDetailsProps) {
  // Handle data loading state separately from component loading state
  if (isLoading) {
    return <ShipmentDetailsSkeleton />;
  }

  // Only use Suspense for component loading, not data loading
  return (
    <Suspense fallback={<ShipmentDetailsSkeleton />}>
      <ShipmentScrollArea {...props}>
        <ShipmentServiceDetails />
        <ShipmentBillingDetails />
        <ShipmentGeneralInformation />
        <ShipmentCommodityDetails />
        <ShipmentMovesDetails />
      </ShipmentScrollArea>
    </Suspense>
  );
});

// Memoize the ShipmentScrollArea to prevent unnecessary re-renders
export const ShipmentScrollArea = memo(function ShipmentScrollArea({
  selectedShipment,
  isError,
  onBack,
  dimensions,
  children,
}: Omit<ShipmentDetailsProps, "isLoading"> & { children: React.ReactNode }) {
  // Handle error state
  if (isError) {
    return (
      <div className="flex size-full items-center justify-center">
        <ShipmentNotFoundOverlay onBack={onBack} />
      </div>
    );
  }

  // Calculate the optimal height for the scroll area
  const calculateScrollAreaHeight = () => {
    const { contentHeight, viewportHeight } = dimensions;

    // Constants for height calculations
    const headerHeight = 120; // Height of the header section
    const minHeight = 400; // Minimum height for the scroll area

    // Use viewport height as base for calculation
    const baseHeight = Math.min(contentHeight, viewportHeight);
    const calculatedHeight = baseHeight - headerHeight;

    // Ensure we don't go below minimum height
    return Math.max(calculatedHeight, minHeight);
  };

  const scrollAreaHeight = `${calculateScrollAreaHeight()}px`;

  return (
    <div className="size-full">
      <div className="pt-4">
        <ShipmentFormHeader
          onBack={onBack}
          selectedShipment={selectedShipment}
        />
        <div className="flex flex-col gap-2 mt-4">
          <ShipmentDetailsHeader />
          <ScrollArea
            className="flex flex-col overflow-y-auto px-4"
            style={{
              height: scrollAreaHeight,
              minHeight: "400px",
            }}
          >
            <div className="flex flex-col gap-4 pb-16">{children}</div>
            <div className="pointer-events-none rounded-b-lg absolute bottom-0 z-50 left-0 right-0 h-8 bg-gradient-to-t from-background to-transparent" />
          </ScrollArea>
        </div>
      </div>
    </div>
  );
});
