import { ComponentLoader } from "@/components/ui/component-loader";
import { useNextProNumber, useShipmentControl } from "@/hooks/useQueries";
import { Suspense, lazy } from "react";

const GeneralInfoCard = lazy(() => import("./cards/general-info"));
const LocationInformation = lazy(() => import("./cards/location-info"));
const EquipmentInformation = lazy(() => import("./cards/equipment-info"));
const DispatchInformation = lazy(() => import("./cards/dispatch-detail"));

export default function GeneralInfoTab() {
  const { data, isLoading: isShipmentControlLoading } = useShipmentControl();
  const { proNumber, isProNumberLoading } = useNextProNumber();

  return (
    <div className="grid grid-cols-1 gap-y-8">
      <Suspense fallback={<ComponentLoader />}>
        <GeneralInfoCard
          proNumber={proNumber as string}
          isProNumberLoading={isProNumberLoading}
          shipmentControlData={data}
          isShipmentControlLoading={isShipmentControlLoading}
        />
        <LocationInformation
          shipmentControlData={data}
          isShipmentControlLoading={isShipmentControlLoading}
        />
        <EquipmentInformation />
        <DispatchInformation
          shipmentControlData={data}
          isShipmentControlLoading={isShipmentControlLoading}
        />
      </Suspense>
    </div>
  );
}
