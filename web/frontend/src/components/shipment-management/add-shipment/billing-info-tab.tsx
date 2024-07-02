import { ScrollArea } from "@/components/ui/scroll-area";
import { lazy } from "react";

const CustomerInformation = lazy(() => import("./cards/customer-info"));
const ShipmentInformation = lazy(() => import("./cards/shipment-info"));
const RateCalcInformation = lazy(() => import("./cards/rate-calc-info"));
const ChargeInformation = lazy(() => import("./cards/charge-info"));

export function BillingInfoTab() {
  return (
    <ScrollArea className="h-[80vh] p-4">
      <div className="grid grid-cols-1 gap-y-8">
        <CustomerInformation />
        <ShipmentInformation />
        <RateCalcInformation />
        <ChargeInformation />
      </div>
    </ScrollArea>
  );
}
