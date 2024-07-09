import { ShipmentMapView } from "@/components/shipment-management/map-view/shipment-map-view";
import { ShipmentBreadcrumb } from "@/components/shipment-management/shipment-breadcrumb";
import { ShipmentListView } from "@/components/shipment-management/shipment-list-view";
import { useShipmentStore } from "@/stores/ShipmentStore";
import { ShipmentSearchForm, ShipmentStatus } from "@/types/shipment";
import { useForm } from "react-hook-form";

const finalStatuses: ShipmentStatus[] = [
  "Completed",
  "Hold",
  "Billed",
  "Voided",
];
const progressStatuses: ShipmentStatus[] = ["New", "InProgress", "Completed"];

export default function ShipmentManagement() {
  const { control, setValue } = useForm<ShipmentSearchForm>({
    defaultValues: {
      searchQuery: "",
      statusFilter: "",
    },
  });

  const [currentView] = useShipmentStore.use("currentView");

  const renderView = () => {
    switch (currentView) {
      case "list":
        return (
          <ShipmentListView
            control={control}
            progressStatuses={progressStatuses}
            finalStatuses={finalStatuses}
            setValue={setValue}
          />
        );
      case "map":
        return <ShipmentMapView />;
      default:
        return null;
    }
  };

  return (
    <>
      <ShipmentBreadcrumb />
      <div className="flex space-x-10 p-4">
        {/* Render the view */}
        {renderView()}
      </div>
    </>
  );
}
