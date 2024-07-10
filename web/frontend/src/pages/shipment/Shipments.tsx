import { ShipmentListView } from "@/components/shipment-management/shipment-list-view";
import { ShipmentSearchForm, ShipmentStatus } from "@/types/shipment";
import { FormProvider, useForm } from "react-hook-form";

const finalStatuses: ShipmentStatus[] = [
  "Completed",
  "Hold",
  "Billed",
  "Voided",
];
const progressStatuses: ShipmentStatus[] = ["New", "InProgress", "Completed"];

export default function ShipmentManagement() {
  const shipmentManagementForm = useForm<ShipmentSearchForm>({
    defaultValues: {
      searchQuery: "",
      statusFilter: "",
    },
  });

  return (
    <FormProvider {...shipmentManagementForm}>
      <div className="flex space-x-10 p-4">
        <ShipmentListView
          progressStatuses={progressStatuses}
          finalStatuses={finalStatuses}
        />
      </div>
    </FormProvider>
  );
}
