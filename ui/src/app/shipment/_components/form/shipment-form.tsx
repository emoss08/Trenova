/* eslint-disable react/display-name */
import type { ShipmentSchema } from "@/lib/schemas/shipment-schema";
import { lazy, memo, Suspense } from "react";
import { ShipmentNotFoundOverlay } from "../sidebar/shipment-not-found-overlay";
import { ShipmentDetailsSkeleton } from "./shipment-details-skeleton";
import { ShipmentFormContent } from "./shipment-form-body";
import { ShipmentFormHeader } from "./shipment-form-header";

// Lazy loaded components
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
const ShipmentSections = memo(() => {
  return (
    <>
      <ShipmentServiceDetails />
      <ShipmentBillingDetails />
      <ShipmentGeneralInformation />
      <ShipmentCommodityDetails />
      <ShipmentMovesDetails />
    </>
  );
});
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
    <ShipmentFormBodyOuter>
      <ShipmentFormHeader selectedShipment={selectedShipment} />
      <ShipmentFormContent selectedShipment={selectedShipment}>
        {children}
      </ShipmentFormContent>
    </ShipmentFormBodyOuter>
  );
}

function ShipmentFormBodyOuter({ children }: { children: React.ReactNode }) {
  return <div className="size-full">{children}</div>;
}
