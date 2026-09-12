import { usePendingActions } from "@/hooks/use-pending-actions";
import { useShipmentBillingActions } from "@/hooks/use-shipment-billing-actions";
import { Button } from "@trenova/shared/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@trenova/shared/components/ui/dropdown-menu";
import type { Shipment } from "@trenova/shared/types/shipment";
import { ChevronDownIcon, ReceiptTextIcon } from "lucide-react";

export function ShipmentBillingActionsMenu({ shipment }: { shipment: Shipment }) {
  const billingActions = useShipmentBillingActions();
  const { pending, run } = usePendingActions();

  const shipmentId = shipment.id;
  const available = Object.values(billingActions).filter((action) => action.isAvailable(shipment));
  if (!shipmentId || available.length === 0) return null;

  const isRunning = pending.has(shipmentId);

  return (
    <DropdownMenu>
      <DropdownMenuTrigger
        render={
          <Button
            type="button"
            variant="outline"
            size="sm"
            isLoading={isRunning}
            loadingText="Billing"
          />
        }
      >
        <ReceiptTextIcon className="size-4" />
        Billing
        <ChevronDownIcon className="size-3.5" />
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" sideOffset={4} className="min-w-56">
        {available.map((action) => {
          const Icon = action.icon;
          return (
            <DropdownMenuItem
              key={action.id}
              title={action.label}
              startContent={<Icon className="size-3.5" />}
              disabled={isRunning}
              onClick={() => void run(shipmentId, action.id, () => action.run(shipmentId))}
            />
          );
        })}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
