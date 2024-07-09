import { ShipmentAsideMenus } from "@/components/shipment-management/map-view/shipment-aside-menu";
import { ShipmentInfo } from "@/components/shipment-management/shipment-list";
import { getShipments } from "@/services/ShipmentRequestService";
import { ShipmentSearchForm, ShipmentStatus } from "@/types/shipment";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { Control, UseFormSetValue } from "react-hook-form";

export function ShipmentListView({
  finalStatuses,
  progressStatuses,
  control,
  setValue,
}: {
  finalStatuses: ShipmentStatus[];
  progressStatuses: ShipmentStatus[];
  control: Control<ShipmentSearchForm>;
  setValue: UseFormSetValue<ShipmentSearchForm>;
}) {
  const queryClient = useQueryClient();

  const { data: shipments } = useQuery({
    queryKey: ["shipments"],
    queryFn: async () => getShipments(),
    initialData: () => queryClient.getQueryData(["shipments"]),
    staleTime: Infinity,
  });

  return (
    <div className="flex w-full space-x-10">
      <div className="w-1/4">
        <ShipmentAsideMenus control={control} setValue={setValue} />
      </div>
      <div className="w-3/4 space-y-4">
        {shipments &&
          shipments?.results.map((shipment) => (
            <ShipmentInfo
              key={shipment.id}
              shipment={shipment}
              finalStatuses={finalStatuses}
              progressStatuses={progressStatuses}
            />
          ))}
      </div>
    </div>
  );
}
