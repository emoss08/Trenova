import type { ShipmentSchema } from "@/lib/schemas/shipment-schema";
import { Suspense } from "react";
import { ShipmentNotFoundOverlay } from "../sidebar/shipment-not-found-overlay";
import { ShipmentDetailsSkeleton } from "./shipment-details-skeleton";
import { ShipmentFormContent } from "./shipment-form-body";
import { ShipmentFormHeader } from "./shipment-form-header";
import { ShipmentGeneralInfoForm } from "./shipment-general-info-form";

export function ShipmentCreateForm() {
  return (
    <Suspense fallback={<ShipmentDetailsSkeleton />}>
      <ShipmentFormBody>
        <ShipmentGeneralInfoForm className="max-h-[calc(100vh-7rem)]" />
      </ShipmentFormBody>
    </Suspense>
  );
}

export function ShipmentFormBody({
  selectedShipment,
  isError,
  children,
}: {
  selectedShipment?: ShipmentSchema | null;
  isError?: boolean;
  children: React.ReactNode;
}) {
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
