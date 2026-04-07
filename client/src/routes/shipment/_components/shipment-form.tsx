"use no memo";
import type { Shipment } from "@/types/shipment";
import { LockIcon } from "lucide-react";
import { parseAsBoolean, useQueryState } from "nuqs";
import { lazy, Suspense } from "react";
import { useFormContext, useWatch } from "react-hook-form";
import { ShipmentFormSkeleton } from "../shipment-form-skeleton";

const ServiceDetails = lazy(() => import("./shipment-service-details"));
const BillingDetails = lazy(() => import("./shipment-billing-details"));
const AdditionalChargesSection = lazy(
  () => import("./additional-charges/shipment-additional-charges"),
);
const ShipmentGeneralInformation = lazy(() => import("./shipment-general-information"));
const CommoditiesSection = lazy(() => import("./shipment-commodities"));
const ShipmentMoveDetails = lazy(() => import("./move/shipment-move-details"));
const LoadPlannerDialog = lazy(() => import("./trailer-loading/trailer-loading-drawer"));

const BILLING_UNLOCKED_STATUSES = new Set(["", null, undefined, "SentBackToOps"]);

export function ShipmentForm() {
  const [loadPlannerOpen, setLoadPlannerOpen] = useQueryState(
    "loadPlanner",
    parseAsBoolean.withDefault(false),
  );

  const { control } = useFormContext<Shipment>();
  const billingTransferStatus = useWatch({ control, name: "billingTransferStatus" });
  const isLockedForBilling = !BILLING_UNLOCKED_STATUSES.has(billingTransferStatus);

  return (
    <div className="flex flex-col gap-6">
      <Suspense fallback={<ShipmentFormSkeleton />}>
        <div className="relative">
          <div className="flex flex-col gap-6">
            <ServiceDetails />
            <BillingDetails />
            <AdditionalChargesSection />
            <ShipmentGeneralInformation />
            <CommoditiesSection />
            <ShipmentMoveDetails />
          </div>
          {isLockedForBilling && (
            <div className="absolute inset-0 z-10 rounded-lg bg-background/60">
              <div className="sticky top-1/3 flex flex-col items-center gap-3 py-12">
                <div className="flex size-12 items-center justify-center rounded-full bg-muted">
                  <LockIcon className="size-5 text-muted-foreground" />
                </div>
                <div className="text-center max-w-sm">
                  <p className="text-sm font-medium">Under Billing Review</p>
                  <p className="text-xs text-muted-foreground mt-1">
                    This shipment is currently being reviewed by the billing team and cannot be
                    modified. If changes are needed, contact your billing department to have it
                    returned to operations.
                  </p>
                </div>
              </div>
            </div>
          )}
        </div>
      </Suspense>
      <Suspense fallback={null}>
        <LoadPlannerDialog open={loadPlannerOpen} onOpenChange={setLoadPlannerOpen} />
      </Suspense>
    </div>
  );
}
