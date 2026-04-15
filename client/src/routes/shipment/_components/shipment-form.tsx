"use no memo";
import type { Shipment } from "@/types/shipment";
import { FileTextIcon, LockIcon } from "lucide-react";
import { parseAsBoolean, useQueryState } from "nuqs";
import { type ReactNode, lazy, Suspense } from "react";
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

const BILLING_REVIEW_STATUSES = new Set(["ReadyForReview", "InReview", "OnHold", "Exception"]);

export function ShipmentForm() {
  const [loadPlannerOpen, setLoadPlannerOpen] = useQueryState(
    "loadPlanner",
    parseAsBoolean.withDefault(false),
  );

  const { control } = useFormContext<Shipment>();
  const billingTransferStatus = useWatch({ control, name: "billingTransferStatus" });

  const isInBillingReview = BILLING_REVIEW_STATUSES.has(billingTransferStatus as string);
  const isInvoiced = billingTransferStatus === "Approved";
  const isCanceled = billingTransferStatus === "Canceled";
  const isFullyLocked = isInBillingReview || isCanceled;

  return (
    <div className="flex flex-col gap-6">
      <Suspense fallback={<ShipmentFormSkeleton />}>
        <div className="relative">
          <div className="flex flex-col gap-6">
            {isInvoiced && <InvoicedBanner />}
            <ServiceDetails />
            <SectionLock locked={isInvoiced}>
              <BillingDetails />
              <AdditionalChargesSection />
            </SectionLock>
            <ShipmentGeneralInformation />
            <CommoditiesSection />
            <ShipmentMoveDetails />
          </div>
          {isFullyLocked && (
            <div className="absolute inset-0 z-10 rounded-lg bg-background/60">
              <div className="sticky top-1/3 flex flex-col items-center gap-3 py-12">
                <div className="flex size-12 items-center justify-center rounded-full bg-muted">
                  <LockIcon className="size-5 text-muted-foreground" />
                </div>
                <div className="max-w-sm text-center">
                  <p className="text-sm font-medium">
                    {isCanceled ? "Billing Canceled" : "Under Billing Review"}
                  </p>
                  <p className="mt-1 text-xs text-muted-foreground">
                    {isCanceled
                      ? "Billing for this shipment has been canceled. No further modifications can be made."
                      : "This shipment is currently being reviewed by the billing team and cannot be modified. If changes are needed, contact your billing department to have it returned to operations."}
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

function InvoicedBanner() {
  return (
    <div className="flex items-center gap-3 rounded-lg border border-blue-200 bg-blue-50 px-4 py-3 dark:border-blue-900 dark:bg-blue-950/50">
      <div className="flex size-8 shrink-0 items-center justify-center rounded-full bg-blue-100 dark:bg-blue-900">
        <FileTextIcon className="size-4 text-blue-600 dark:text-blue-400" />
      </div>
      <div>
        <p className="text-sm font-medium text-blue-900 dark:text-blue-100">Invoiced</p>
        <p className="text-xs text-blue-700 dark:text-blue-300">
          This shipment has been invoiced. Billing and charge fields are locked. To make financial
          corrections, issue a credit memo and rebill.
        </p>
      </div>
    </div>
  );
}

function SectionLock({ locked, children }: { locked: boolean; children: ReactNode }) {
  if (!locked) return children;

  return (
    <div className="relative flex flex-col gap-6">
      {children}
      <div className="absolute inset-0 z-10 flex cursor-not-allowed items-center justify-center rounded-lg bg-background/60">
        <div className="flex items-center gap-2 rounded-md bg-muted px-3 py-1.5">
          <LockIcon className="size-3.5 text-muted-foreground" />
          <span className="text-xs font-medium text-muted-foreground">
            Locked — shipment has been invoiced
          </span>
        </div>
      </div>
    </div>
  );
}
