import { ScrollArea } from "@/components/ui/scroll-area";
import { useAutoCalculateTotals } from "@/hooks/shipment/use-auto-calculate-totals";
import { ShipmentSchema } from "@/lib/schemas/shipment-schema";
import { cn } from "@/lib/utils";
import { lazy } from "react";
import { useFormContext } from "react-hook-form";

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

export function ShipmentGeneralInfoForm({ className }: { className?: string }) {
  const form = useFormContext<ShipmentSchema>();

  if (!form) {
    throw new Error(
      "ShipmentGeneralInfoForm must be used within a FormProvider",
    );
  }

  useAutoCalculateTotals({
    enabled: true,
    debounceMs: 1000,
  });

  return (
    <ScrollArea className={cn("flex flex-col px-4 h-full", className)}>
      <ShipmentGeneralInfoFormInner>
        <ShipmentServiceDetails />
        <ShipmentBillingDetails />
        <ShipmentGeneralInformation />
        <ShipmentCommodityDetails />
        <ShipmentMovesDetails />
      </ShipmentGeneralInfoFormInner>
    </ScrollArea>
  );
}

function ShipmentGeneralInfoFormInner({
  children,
}: {
  children: React.ReactNode;
}) {
  return <div className="space-y-4 pt-2 pb-16">{children}</div>;
}
