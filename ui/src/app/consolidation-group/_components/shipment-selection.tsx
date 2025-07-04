import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { queries } from "@/lib/queries";
import { type CreateConsolidationSchema } from "@/lib/schemas/consolidation-schema";
import { ShipmentStatus } from "@/lib/schemas/shipment-schema";
import { useQuery } from "@tanstack/react-query";
import { useFormContext } from "react-hook-form";

export function ShipmentSelection() {
  const { setValue, watch } = useFormContext<CreateConsolidationSchema>();
  const selectedShipmentIds = watch("shipmentIds") || [];

  // Fetch available shipments
  const { data: shipments, isLoading } = useQuery({
    ...queries.shipment.list({
      limit: 100,
      offset: 0,
      filters: [
        {
          field: "status",
          operator: "eq",
          value: ShipmentStatus.enum.New,
        },
      ],
    }),
  });

  const handleToggle = (shipmentId: string) => {
    const currentIds = [...selectedShipmentIds];
    const index = currentIds.indexOf(shipmentId);

    if (index > -1) {
      currentIds.splice(index, 1);
    } else {
      currentIds.push(shipmentId);
    }

    setValue("shipmentIds", currentIds, {
      shouldDirty: true,
      shouldValidate: true,
    });
  };

  if (isLoading) {
    return <div>Loading shipments...</div>;
  }

  if (!shipments?.results?.length) {
    return <div>No shipments available for consolidation</div>;
  }

  return (
    <div className="space-y-4">
      <div className="flex gap-2">
        <Button
          type="button"
          variant="outline"
          size="sm"
          onClick={() => {
            const allIds = shipments.results
              .map((s) => s.id)
              .filter(Boolean) as string[];
            setValue("shipmentIds", allIds);
          }}
        >
          Select All
        </Button>
        <Button
          type="button"
          variant="outline"
          size="sm"
          onClick={() => setValue("shipmentIds", [])}
        >
          Clear All
        </Button>
      </div>

      <div className="border rounded-lg p-4 space-y-2 max-h-[400px] overflow-y-auto">
        {shipments.results.map((shipment) => (
          <label
            key={shipment.id}
            className="flex items-center space-x-3 p-2 hover:bg-accent rounded cursor-pointer"
          >
            <Checkbox
              checked={selectedShipmentIds.includes(shipment.id!)}
              onCheckedChange={() => handleToggle(shipment.id!)}
            />
            <div className="flex-1">
              <div className="font-medium">
                {shipment.proNumber || "No Pro #"} - {shipment.bol || "No BOL"}
              </div>
              <div className="text-sm text-muted-foreground">
                {shipment.customer?.name || "Unknown Customer"}
              </div>
            </div>
          </label>
        ))}
      </div>

      {selectedShipmentIds.length > 0 && (
        <div className="text-sm text-muted-foreground">
          {selectedShipmentIds.length} shipment(s) selected
        </div>
      )}
    </div>
  );
}
