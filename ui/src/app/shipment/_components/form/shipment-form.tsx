import { ScrollArea } from "@/components/ui/scroll-area";
import type { ShipmentSchema } from "@/lib/schemas/shipment-schema";
import { lazy, memo, Suspense } from "react";
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
  selectedShipment?: ShipmentSchema | null;
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
  children,
}: Omit<ShipmentDetailsProps, "isLoading"> & { children: React.ReactNode }) {
  // Handle error state
  if (isError) {
    return (
      <div className="flex size-full items-center justify-center">
        <ShipmentNotFoundOverlay />
      </div>
    );
  }

  return (
    <div className="size-full">
      <ShipmentFormHeader selectedShipment={selectedShipment} />
      <ShipmentScrollAreaOuter>
        <ShipmentDetailsHeader selectedShipment={selectedShipment} />
        <ShipmentScrollArea>
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

function ShipmentScrollArea({ children }: { children: React.ReactNode }) {
  return (
    <ScrollArea className="flex flex-col overflow-y-auto max-h-[calc(100vh-8.5rem)]">
      {children}
    </ScrollArea>
  );
}
