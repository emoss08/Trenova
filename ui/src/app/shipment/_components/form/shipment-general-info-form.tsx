import { ScrollArea } from "@/components/ui/scroll-area";
import { useAutoCalculateTotals } from "@/hooks/shipment/use-auto-calculate-totals";
import { ShipmentSchema } from "@/lib/schemas/shipment-schema";
import { cn } from "@/lib/utils";
import { lazy, memo } from "react";
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

function ShipmentGeneralInfoFormComponent({
  className,
}: {
  className?: string;
}) {
  const form = useFormContext<ShipmentSchema>();

  if (!form) {
    throw new Error(
      "ShipmentGeneralInfoForm must be used within a FormProvider",
    );
  }

  // Enable automatic billing calculations
  useAutoCalculateTotals({
    enabled: true,
    debounceMs: 1000,
  });

  return (
    <ScrollArea
      className={cn(
        "flex flex-col overflow-y-auto px-4 max-h-[calc(100vh-8rem)]",
        className,
      )}
    >
      <div className="space-y-4 pb-16 pt-2">
        <ShipmentServiceDetails />
        <ShipmentBillingDetails />
        <ShipmentGeneralInformation />
        <ShipmentCommodityDetails />
        <ShipmentMovesDetails />
      </div>
    </ScrollArea>
  );
}

export const ShipmentGeneralInfoForm = memo(ShipmentGeneralInfoFormComponent);
