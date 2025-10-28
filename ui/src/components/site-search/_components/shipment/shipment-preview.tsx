import { useShipmentDetails } from "@/app/shipment/queries/shipment";
import { Spinner } from "@/components/ui/shadcn-io/spinner";
import type { ShipmentSchema } from "@/lib/schemas/shipment-schema";
import { ShipmentPreviewContent } from "./shipment-preview-content";

export function ShipmentSearchPreview({
  shipmentId,
}: {
  shipmentId: ShipmentSchema["id"];
}) {
  const enabled = Boolean(shipmentId);
  const { data, isLoading, isError } = useShipmentDetails({
    shipmentId,
    enabled,
  });

  if (!shipmentId) {
    return (
      <div className="h-full w-full flex items-center justify-center text-2xs text-muted-foreground">
        Hover a shipment to preview details
      </div>
    );
  }

  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-full">
        <Spinner variant="circle" className="size-8" />
      </div>
    );
  }

  if (isError || !data) {
    return (
      <div className="text-2xs text-muted-foreground">
        Unable to load shipment.
      </div>
    );
  }

  return <ShipmentPreviewContent shipmentId={shipmentId} />;
}
