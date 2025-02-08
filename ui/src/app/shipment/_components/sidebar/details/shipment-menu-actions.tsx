import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Icon } from "@/components/ui/icons";
import { ShipmentStatus, type Shipment } from "@/types/shipment";
import { faEllipsisVertical } from "@fortawesome/pro-regular-svg-icons";
import { useState } from "react";
import { ShipmentCancellationDialog } from "../../cancellation/shipment-cancellatioin-dialog";

// Map of status that are allowed to be canceled.
const cancellatedStatuses = [
  ShipmentStatus.New,
  ShipmentStatus.InTransit,
  ShipmentStatus.Delayed,
];

export function ShipmentActions({ shipment }: { shipment: Shipment }) {
  const [cancellationDialogOpen, setCancellationDialogOpen] =
    useState<boolean>(false);

  const isCancellable = cancellatedStatuses.includes(shipment.status);

  return (
    <>
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button variant="outline" size="sm" className="p-2">
            <Icon icon={faEllipsisVertical} className="size-4" />
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="start">
          <DropdownMenuLabel>General Actions</DropdownMenuLabel>
          <DropdownMenuSeparator />
          <DropdownMenuItem
            title="Assign"
            description="Assign this shipment to a worker(s)."
          />
          <DropdownMenuItem
            title="Edit"
            description="Modify shipment details."
          />
          <DropdownMenuItem
            title="Duplicate"
            description="Create a copy of this shipment."
          />
          <DropdownMenuItem
            title="Cancel"
            description="Cancel this shipment and update its status."
            onClick={() => setCancellationDialogOpen(true)}
            disabled={!isCancellable}
          />
          <DropdownMenuLabel>Management Actions</DropdownMenuLabel>
          <DropdownMenuSeparator />
          <DropdownMenuItem
            title="Split Shipment"
            description="Divide this shipment into multiple parts."
          />
          <DropdownMenuItem
            title="Merge Shipment"
            description="Combine multiple shipments into one."
          />
          <DropdownMenuItem
            title="Send to Worker"
            description="Assign this shipment for processing."
          />
          <DropdownMenuLabel>Documentation & Communication</DropdownMenuLabel>
          <DropdownMenuSeparator />
          <DropdownMenuItem
            title="Add Document(s)"
            description="Attach relevant documents to this shipment."
          />
          <DropdownMenuItem
            title="Add Comment(s)"
            description="Leave internal notes or comments on this shipment."
          />
          <DropdownMenuLabel>View Actions</DropdownMenuLabel>
          <DropdownMenuSeparator />
          <DropdownMenuItem
            title="View Documents"
            description="Review attached shipment documents."
          />
          <DropdownMenuItem
            title="View Comments"
            description="Check comments and notes related to this shipment."
          />
          <DropdownMenuItem
            title="View Audit Log"
            description="Track all modifications and updates to this shipment."
          />
        </DropdownMenuContent>
      </DropdownMenu>
      <ShipmentCancellationDialog
        open={cancellationDialogOpen}
        onOpenChange={setCancellationDialogOpen}
        shipmentId={shipment.id ?? ""}
      />
    </>
  );
}
