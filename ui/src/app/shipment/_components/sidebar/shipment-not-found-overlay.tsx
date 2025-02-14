import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Icon } from "@/components/ui/icons";
import {
    faChevronLeft,
    faCircleXmark,
} from "@fortawesome/pro-regular-svg-icons";

interface ShipmentNotFoundOverlayProps {
  onBack: () => void;
}

export function ShipmentNotFoundOverlay({
  onBack,
}: ShipmentNotFoundOverlayProps) {
  return (
    <div className="relative size-full rounded-md">
      <div className="absolute inset-0 flex items-center justify-center bg-background/50 backdrop-blur-[2px]">
        <Card className="w-[400px] p-6 shadow-lg border">
          <div className="flex flex-col items-center mb-4">
            <div className="relative mb-2">
              <div className="absolute -inset-1 rounded-full bg-destructive/20 motion-safe:animate-pulse" />
              <Icon
                icon={faCircleXmark}
                className="size-12 text-destructive relative"
              />
            </div>
            <h3 className="text-xl font-semibold text-destructive">
              Shipment Not Found
            </h3>
          </div>
          <Button
            variant="outline"
            size="sm"
            onClick={onBack}
            className="w-full gap-2"
          >
            <Icon icon={faChevronLeft} className="size-4" />
            Return to Shipments
          </Button>
        </Card>
      </div>
    </div>
  );
}
