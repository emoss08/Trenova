import { Button } from "@/components/ui/button";
import { Icon } from "@/components/ui/icons";
import { ScrollArea } from "@/components/ui/scroll-area";
import { type Shipment } from "@/types/shipment";
import { faChevronLeft } from "@fortawesome/pro-solid-svg-icons";
import { ShipmentCommodityDetails } from "./shipment-commodity-details";
import {
  ShipmentBillingDetails,
  ShipmentDetailsHeader,
  ShipmentServiceDetails,
} from "./shipment-details-components";
import { ShipmentDetailActions } from "./shipment-menu-actions";
import { ShipmentMovesDetails } from "./shipment-move-details";

interface ShipmentDetailsProps {
  selectedShipment?: Shipment | null;

  isLoading: boolean;
  onBack: () => void;
}

export default function ShipmentDetails({
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
      <div className="py-2">
        <div className="flex items-center gap-2 px-4 justify-between">
          <Button variant="outline" size="sm" onClick={onBack}>
            <Icon icon={faChevronLeft} className="size-4" />
            <span className="text-sm">Back</span>
          </Button>
          <ShipmentDetailActions />
        </div>
        <div className="flex flex-col gap-2 mt-4">
          <ShipmentDetailsHeader />
          <ScrollArea className="flex max-h-[calc(100vh-12rem)] flex-col overflow-y-auto px-4">
            <ShipmentServiceDetails />
            <ShipmentBillingDetails />
            <ShipmentCommodityDetails />
            <ShipmentMovesDetails />
            <div className="pointer-events-none absolute bottom-0 left-0 right-0 h-8 bg-gradient-to-t from-sidebar to-transparent" />
          </ScrollArea>
        </div>
      </div>
    </div>
  );
}
