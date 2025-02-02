import { Button } from "@/components/ui/button";
import { CardTitle } from "@/components/ui/card";
import { Icon } from "@/components/ui/icons";
import { ShipmentLocations } from "@/lib/shipment/utils";
import { type Shipment as ShipmentResponse } from "@/types/shipment";
import { faChevronLeft } from "@fortawesome/pro-regular-svg-icons";

interface ShipmentDetailsProps {
  selectedShipment?: ShipmentResponse | null;

  isLoading: boolean;
  onBack: () => void;
}

export function ShipmentDetails({
  selectedShipment,
  isLoading,
  onBack,
}: ShipmentDetailsProps) {
  if (!selectedShipment) {
    return null;
  }

  const { origin, destination } =
    ShipmentLocations.useLocations(selectedShipment);

  if (isLoading) {
    return <p>Loading...</p>;
  }

  return (
    <div className="size-full">
      <div className="px-2">
        <div className="flex items-center gap-2">
          <Button
            variant="ghost"
            size="icon"
            className="h-8 w-8"
            onClick={onBack}
          >
            <Icon icon={faChevronLeft} className="size-4" />
          </Button>
          <CardTitle className="text-lg">Shipment Details</CardTitle>
        </div>
      </div>
    </div>
  );
}
