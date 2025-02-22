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
import { useNavigate } from "react-router";
import { ShipmentCancellationDialog } from "../cancellation/shipment-cancellatioin-dialog";
import { ShipmentDuplicateDialog } from "../duplicate/shipment-duplicate-dialog";

// Map of status that are allowed to be canceled.
const cancellatedStatuses = [
  ShipmentStatus.New,
  ShipmentStatus.InTransit,
  ShipmentStatus.Delayed,
  ShipmentStatus.PartiallyCompleted,
  ShipmentStatus.Completed,
];

export function ShipmentActions({ shipment }: { shipment?: Shipment | null }) {
  const navigate = useNavigate();

  const [cancellationDialogOpen, setCancellationDialogOpen] =
    useState<boolean>(false);
  const [duplicateDialogOpen, setDuplicateDialogOpen] =
    useState<boolean>(false);

  if (!shipment) {
    return null;
  }

  const isCancellable = cancellatedStatuses.includes(shipment.status);

  return (
    <>
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button variant="outline" size="sm" className="p-2">
            <Icon icon={faEllipsisVertical} className="size-4" />
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent side="bottom" align="end">
          <DropdownMenuLabel>General Actions</DropdownMenuLabel>
          <DropdownMenuSeparator />
          <DropdownMenuItem
            title="Assign"
            description="Assign this shipment to a worker(s)."
          />
          <DropdownMenuItem
            title="Edit"
            description="Modify shipment details."
            onClick={() => {
              navigate(`/shipments/${shipment.id}`);
            }}
          />
          <DropdownMenuItem
            title="Duplicate"
            description="Create a copy of this shipment."
            onClick={() => setDuplicateDialogOpen(true)}
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
      <ShipmentDuplicateDialog
        open={duplicateDialogOpen}
        onOpenChange={setDuplicateDialogOpen}
        shipment={shipment}
      />
      <ShipmentCancellationDialog
        open={cancellationDialogOpen}
        onOpenChange={setCancellationDialogOpen}
        shipmentId={shipment.id ?? ""}
      />
    </>
  );
}
