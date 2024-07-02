import { ShipmentAsideMenus } from "@/components/shipment-management/map-view/shipment-aside-menu";
import { ShipmentList } from "@/components/shipment-management/shipment-list";
import { ShipmentSearchForm } from "@/types/shipment";
import { Control, UseFormSetValue, UseFormWatch } from "react-hook-form";

export function ShipmentListView({
  finalStatuses,
  progressStatuses,
  control,
  setValue,
  watch,
}: {
  finalStatuses: string[];
  progressStatuses: string[];
  control: Control<ShipmentSearchForm>;
  setValue: UseFormSetValue<ShipmentSearchForm>;
  watch: UseFormWatch<ShipmentSearchForm>;
}) {
  return (
    <div className="flex w-full space-x-10">
      <div className="w-1/4">
        <ShipmentAsideMenus
          control={control}
          setValue={setValue}
          watch={watch}
        />
      </div>
      <div className="w-3/4">
        <ShipmentList
          finalStatuses={finalStatuses}
          progressStatuses={progressStatuses}
          watch={watch}
        />
      </div>
    </div>
  );
}
