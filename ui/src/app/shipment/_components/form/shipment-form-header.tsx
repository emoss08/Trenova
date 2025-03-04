import { Button } from "@/components/ui/button";
import { Icon } from "@/components/ui/icons";
import { Shipment } from "@/types/shipment";
import { faChevronLeft } from "@fortawesome/pro-regular-svg-icons";
import { ShipmentActions } from "./shipment-menu-actions";

type ShipmentFormHeaderProps = {
  onBack: () => void;
  selectedShipment?: Shipment | null;
};

export function ShipmentFormHeader({
  onBack,
  selectedShipment,
}: ShipmentFormHeaderProps) {
  return (
    <div className="flex items-center px-4 justify-between">
      <HeaderBackButton onBack={onBack} />
      <ShipmentActions shipment={selectedShipment} />
    </div>
  );
}

function HeaderBackButton({ onBack }: { onBack: () => void }) {
  return (
    <Button variant="outline" size="sm" onClick={onBack}>
      <Icon icon={faChevronLeft} className="size-4" />
      <span className="text-sm">Back</span>
    </Button>
  );
}
