import { useShipmentDetails } from "@/app/shipment/queries/shipment";
import { ShipmentStatusBadge } from "@/components/status-badge";
import { Spinner } from "@/components/ui/shadcn-io/spinner";
import { ShipmentSchema } from "@/lib/schemas/shipment-schema";
import { ShipmentMovePreview } from "./shipment-move-preview";
import { ShipmentRouteMap } from "./shipment-preview-map";

export function ShipmentPreviewContent({
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

  return (
    <div className="flex flex-col gap-3 text-sm">
      <div className="flex flex-col gap-1">
        <div className="flex items-center justify-between gap-2">
          <h3 className="text-xl">{data.proNumber || data.id}</h3>
          <ShipmentStatusBadge status={data.status} />
        </div>
        <div className="flex flex-row items-center gap-2">
          <div className="text-2xs text-muted-foreground truncate max-w-[200px]">
            Customer: {data.customer?.name} ({data.customer?.code})
          </div>
          <div className="text-2xs text-muted-foreground truncate max-w-[200px]">
            BOL: {data.bol}
          </div>
        </div>
      </div>
      <ShipmentRouteMap moves={data.moves} />
      <ShipmentMovePreview moves={data.moves} />
    </div>
  );
}
