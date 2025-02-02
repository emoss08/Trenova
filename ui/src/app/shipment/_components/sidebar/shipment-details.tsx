import { ShipmentStatusBadge } from "@/components/status-badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Icon } from "@/components/ui/icons";
import { ShipmentLocations } from "@/lib/shipment/utils";
import { formatLocation } from "@/lib/utils";
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
    <Card className="border-none shadow-none">
      <CardHeader className="px-2">
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
      </CardHeader>
      <CardContent className="px-2 space-y-6">
        <div className="space-y-2">
          <h3 className="text-sm font-medium">Status</h3>
          <ShipmentStatusBadge status={selectedShipment.status} />
        </div>

        <div className="space-y-2">
          <h3 className="text-sm font-medium">PRO Number</h3>
          <p className="text-sm text-muted-foreground">
            {selectedShipment.proNumber}
          </p>
        </div>

        <div className="space-y-2">
          <h3 className="text-sm font-medium">Customer</h3>
          <p className="text-sm text-muted-foreground">
            {selectedShipment.customer.name}
          </p>
        </div>

        {origin && (
          <div className="space-y-2">
            <h3 className="text-sm font-medium">Origin</h3>
            <p className="text-sm text-muted-foreground">
              {formatLocation(origin)}
            </p>
          </div>
        )}

        {destination && (
          <div className="space-y-2">
            <h3 className="text-sm font-medium">Destination</h3>
            <p className="text-sm text-muted-foreground">
              {formatLocation(destination)}
            </p>
          </div>
        )}
      </CardContent>
    </Card>
  );
}
