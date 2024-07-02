import { ShipmentMapView } from "@/components/shipment-management/map-view/shipment-map-view";
import { ShipmentBreadcrumb } from "@/components/shipment-management/shipment-breadcrumb";
import { ShipmentListView } from "@/components/shipment-management/shipment-list-view";
import { useShipmentStore } from "@/stores/ShipmentStore";
import { ShipmentSearchForm } from "@/types/shipment";
import { useForm } from "react-hook-form";

const finalStatuses = ["C", "H", "B", "V"];
const progressStatuses = ["N", "P", "C"];

export default function ShipmentManagement() {
  const { control, watch, setValue } = useForm<ShipmentSearchForm>({
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
            watch={watch}
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
