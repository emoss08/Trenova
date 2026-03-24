"use no memo";
import { lazy, Suspense } from "react";
import { ShipmentFormSkeleton } from "../shipment-form-skeleton";

const ServiceDetails = lazy(() => import("./shipment-service-details"));
const BillingDetails = lazy(() => import("./shipment-billing-details"));
const AdditionalChargesSection = lazy(
  () => import("./additional-charges/shipment-additional-charges"),
);
const ShipmentGeneralInformation = lazy(() => import("./shipment-general-information"));
const CommoditiesSection = lazy(() => import("./shipment-commodities"));
const ShipmentMoveDetails = lazy(() => import("./move/shipment-move-details"));

export function ShipmentForm() {
  return (
    <div className="flex flex-col gap-6">
      <Suspense fallback={<ShipmentFormSkeleton />}>
        <ServiceDetails />
        <BillingDetails />
        <AdditionalChargesSection />
        <ShipmentGeneralInformation />
        <CommoditiesSection />
        <ShipmentMoveDetails />
      </Suspense>
    </div>
  );
}
