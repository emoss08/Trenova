import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { queries } from "@/lib/queries";
import { type CreateConsolidationSchema } from "@/lib/schemas/consolidation-schema";
import { ShipmentSchema, ShipmentStatus } from "@/lib/schemas/shipment-schema";
import { useQuery } from "@tanstack/react-query";
import { useFormContext } from "react-hook-form";

export function ShipmentSelection({
  shipmentErrors,
}: {
  shipmentErrors?: string | null;
}) {
  const { setValue, watch } = useFormContext<CreateConsolidationSchema>();
  const selectedShipments = watch("shipments") || [];

  // * Get the selected shipment IDs, for the checkbo
  const selectedShipmentIds = selectedShipments.map((s) => s.id);

  // Fetch available shipments
  const { data: shipments, isLoading } = useQuery({
    ...queries.shipment.list({
      limit: 100,
      offset: 0,
      expandShipmentDetails: true,
      filters: [
        {
          field: "status",
          operator: "eq",
          value: ShipmentStatus.enum.New,
        },
      ],
    }),
  });

  const handleToggle = (shipment: ShipmentSchema) => {
    const currentShipments = [...selectedShipments];
    const index = currentShipments.indexOf(shipment);

    // * Check to ensure the same shipment is not added twice
    if (currentShipments.some((s) => s.id === shipment.id)) {
      return;
    }

    if (index > -1) {
      currentShipments.splice(index, 1);
    } else {
      currentShipments.push(shipment);
    }

    setValue("shipments", currentShipments, {
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
            const allShipments = shipments.results.map((s) => s);
            setValue("shipments", allShipments);
          }}
        >
          Select All
        </Button>
        <Button
          type="button"
          variant="outline"
          size="sm"
          onClick={() => setValue("shipments", [])}
        >
          Clear All
        </Button>
      </div>

      <div className="border rounded-lg p-4 space-y-2 max-h-[400px] overflow-y-auto">
        {shipmentErrors && (
          <div className="bg-destructive/10 border border-destructive/20 rounded-md p-2">
            {shipmentErrors.split(";").map((error, idx) => (
              <div key={idx} className="text-red-100 text-sm">
                {error}
              </div>
            ))}
          </div>
        )}
        {shipments.results.map((shipment) => (
          <label
            key={shipment.id}
            className="flex items-center space-x-3 p-2 hover:bg-accent rounded cursor-pointer"
          >
            <Checkbox
              checked={selectedShipmentIds.includes(shipment.id)}
              onCheckedChange={() => handleToggle(shipment)}
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

      {selectedShipments.length > 0 && (
        <div className="text-sm text-muted-foreground">
          {selectedShipments.length} shipment(s) selected
        </div>
      )}
    </div>
  );
}
