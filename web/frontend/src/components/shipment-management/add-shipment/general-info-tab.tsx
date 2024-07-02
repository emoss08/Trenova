import { ScrollArea } from "@/components/ui/scroll-area";
import { useNextProNumber, useShipmentControl } from "@/hooks/useQueries";
import { DispatchInformationCard } from "./cards/dispatch-info";
import EquipmentInformationCard from "./cards/equipment-info";
import { GeneralInformationCard } from "./cards/general-info";
import { LocationInformationCard } from "./cards/location-info";

export function GeneralInfoTab() {
  const { data, isLoading: isShipmentControlLoading } = useShipmentControl();
  const { proNumber, isProNumberLoading } = useNextProNumber();

  return (
    <ScrollArea className="h-[80vh] p-4">
      <div className="grid grid-cols-1 gap-y-8">
        <GeneralInformationCard
          proNumber={proNumber as string}
          isProNumberLoading={isProNumberLoading}
          shipmentControlData={data}
          isShipmentControlLoading={isShipmentControlLoading}
        />
        <LocationInformationCard
          shipmentControlData={data}
          isShipmentControlLoading={isShipmentControlLoading}
        />
        <EquipmentInformationCard />
        <DispatchInformationCard
          shipmentControlData={data}
          isShipmentControlLoading={isShipmentControlLoading}
        />
      </div>
    </ScrollArea>
  );
}
