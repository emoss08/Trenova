import { ScrollArea } from "@/components/ui/scroll-area";
import { useResponsiveDimensions } from "@/hooks/use-responsive-dimensions";
import { type Shipment } from "@/types/shipment";
import { lazy, memo, Suspense, useMemo } from "react";
import { ShipmentNotFoundOverlay } from "../sidebar/shipment-not-found-overlay";
import { ShipmentDetailsSkeleton } from "./shipment-details-skeleton";
import { ShipmentFormHeader } from "./shipment-form-header";

// Lazy loaded components
const ShipmentDetailsHeader = lazy(() => import("./shipment-details-header"));
const ShipmentBillingDetails = lazy(
  () => import("./billing-details/shipment-billing-details"),
);
const ShipmentGeneralInformation = lazy(
  () => import("./shipment-general-information"),
);
const ShipmentCommodityDetails = lazy(
  () => import("./commodity/commodity-details"),
);
const ShipmentMovesDetails = lazy(() => import("./move/move-details"));
const ShipmentServiceDetails = lazy(
  () => import("./service-details/shipment-service-details"),
);

type ShipmentDetailsProps = {
  onBack: () => void;
  open: boolean;
  sheetRef: React.RefObject<HTMLDivElement | null>;
  selectedShipment?: Shipment | null;
  isLoading?: boolean;
  isError?: boolean;
};

export function ShipmentForm({ isLoading, ...props }: ShipmentDetailsProps) {
  if (isLoading) {
    return <ShipmentDetailsSkeleton />;
  }

  return (
    <Suspense fallback={<ShipmentDetailsSkeleton />}>
      <ShipmentFormBody {...props}>
        <ShipmentSections />
      </ShipmentFormBody>
    </Suspense>
  );
}

// Separate component for the sections to prevent re-renders of the scroll area container
const ShipmentSectionsComponent = () => {
  return (
    <>
      <ShipmentServiceDetails />
      <ShipmentBillingDetails />
      <ShipmentGeneralInformation />
      <ShipmentCommodityDetails />
      <ShipmentMovesDetails />
    </>
  );
};

ShipmentSectionsComponent.displayName = "ShipmentSections";
const ShipmentSections = memo(ShipmentSectionsComponent);

export function ShipmentFormBody({
  selectedShipment,
  isError,
  onBack,
  children,
  open,
  sheetRef,
}: Omit<ShipmentDetailsProps, "isLoading"> & { children: React.ReactNode }) {
  // Handle error state
  if (isError) {
    return (
      <div className="flex size-full items-center justify-center">
        <ShipmentNotFoundOverlay onBack={onBack} />
      </div>
    );
  }

  return (
    <div className="size-full">
      <ShipmentFormHeader onBack={onBack} selectedShipment={selectedShipment} />
      <ShipmentScrollAreaOuter>
        <ShipmentDetailsHeader />
        <ShipmentScrollArea sheetRef={sheetRef} open={open}>
          <div className="flex flex-col gap-4 p-4 pb-16">{children}</div>
          <div className="pointer-events-none rounded-b-lg absolute bottom-0 z-50 left-0 right-0 h-8 bg-gradient-to-t from-background to-transparent" />
        </ShipmentScrollArea>
      </ShipmentScrollAreaOuter>
    </div>
  );
}

function ShipmentScrollAreaOuter({ children }: { children: React.ReactNode }) {
  return <div className="flex flex-col">{children}</div>;
}

function ShipmentScrollArea({
  sheetRef,
  open,
  children,
}: {
  sheetRef: React.RefObject<HTMLDivElement | null>;
  open: boolean;
  children: React.ReactNode;
}) {
  const dimensions = useResponsiveDimensions(sheetRef, open);

  const scrollAreaHeight = useMemo(() => {
    const { contentHeight, viewportHeight } = dimensions;

    // Constants for height calculations
    const headerHeight = 120; // Height of the header section
    const minHeight = 400; // Minimum height for the scroll area

    // Use viewport height as base for calculation
    const baseHeight = Math.min(contentHeight, viewportHeight);
    const calculatedHeight = baseHeight - headerHeight;

    // Ensure we don't go below minimum height
    return `${Math.max(calculatedHeight, minHeight)}px`;
  }, [dimensions]);

  return (
    <ScrollArea
      className="flex flex-col overflow-y-auto"
      style={{
        height: scrollAreaHeight,
        minHeight: "400px",
      }}
    >
      {children}
    </ScrollArea>
  );
}
