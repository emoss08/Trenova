import { Button } from "@/components/ui/button";
import { Icon } from "@/components/ui/icons";
import { type Shipment } from "@/types/shipment";
import { faChevronLeft } from "@fortawesome/pro-solid-svg-icons";
import {
  ShipmentBillingDetails,
  ShipmentCommodityDetails,
  ShipmentDetailsHeader,
  ShipmentServiceDetails
} from "./shipment-details-components";

interface ShipmentDetailsProps {
  selectedShipment?: Shipment | null;

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

  if (isLoading) {
    return <p>Loading...</p>;
  }

  return (
    <div className="size-full">
      <div className="py-2 px-3">
        <div className="flex items-center gap-2">
          <Button variant="outline" size="sm" onClick={onBack}>
            <Icon icon={faChevronLeft} className="size-4" />
            <span className="text-sm">Back</span>
          </Button>
        </div>
        <div className="flex flex-col gap-2 mt-4 px-2">
          <ShipmentDetailsHeader
            proNumber={selectedShipment.proNumber}
            status={selectedShipment.status}
            bol={selectedShipment.bol}
          />
          <ShipmentServiceDetails shipment={selectedShipment} />
          <ShipmentBillingDetails shipment={selectedShipment} />
          <ShipmentCommodityDetails shipment={selectedShipment} />
        </div>
      </div>
    </div>
  );
}
