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
import { shipmentActionsParser } from "@/hooks/use-shipment-actions-state";
import {
  ShipmentStatus,
  type ShipmentSchema,
} from "@/lib/schemas/shipment-schema";
import { useShipmentPermissions } from "@/types/_gen/permissions";
import { faEllipsisVertical } from "@fortawesome/pro-regular-svg-icons";
import { useQueryStates } from "nuqs";
import { ShipmentCancellationDialog } from "../cancellation/shipment-cancellation-dialog";
import { UnCancelShipmentDialog } from "../cancellation/shipment-uncanel-dialog";
import { ShipmentDuplicateDialog } from "../duplicate/shipment-duplicate-dialog";
import { TransferOwnershipDialog } from "../transfer-ownership/transfer-ownership-dialog";

// Map of status that are allowed to be canceled.
// const cancellatedStatuses = [
//   ShipmentStatus.New,
//   ShipmentStatus.InTransit,
//   ShipmentStatus.Delayed,
//   ShipmentStatus.PartiallyCompleted,
//   ShipmentStatus.Completed,
// ];

export function ShipmentActions({
  shipment,
}: {
  shipment?: ShipmentSchema | null;
}) {
  const { canDuplicate, canAssign } = useShipmentPermissions();
  const [searchParams, setSearchParams] = useQueryStates(shipmentActionsParser);

  if (!shipment) {
    return null;
  }

  return (
    <>
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button variant="outline" size="icon" className="p-2">
            <Icon icon={faEllipsisVertical} className="size-4" />
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent side="bottom" align="end">
          <DropdownMenuLabel>General Actions</DropdownMenuLabel>
          <DropdownMenuSeparator />
          <DropdownMenuItem
            title="Assign"
            description="Assign this shipment to a worker(s)."
            disabled={!canAssign}
          />
          <DropdownMenuItem
            title="Duplicate"
            description="Create a copy of this shipment."
            onClick={() => setSearchParams({ duplicateDialogOpen: true })}
            disabled={!canDuplicate}
          />
          <DropdownMenuItem
            title={
              shipment.status === ShipmentStatus.enum.Canceled
                ? "Un-Cancel"
                : "Cancel"
            }
            description="Cancel this shipment and update its status."
            onClick={() => {
              if (shipment.status === ShipmentStatus.enum.Canceled) {
                setSearchParams({ unCancelDialogOpen: true });
              } else {
                setSearchParams({ cancellationDialogOpen: true });
              }
            }}
          />
          <DropdownMenuItem
            title="Transfer Ownership"
            description="Transfer this shipment to a different user."
            onClick={() => setSearchParams({ transferDialogOpen: true })}
          />
          {/* 
          <DropdownMenuLabel>Management Actions</DropdownMenuLabel>
          <DropdownMenuSeparator />
          <DropdownMenuItem
            title="Hold Shipment"
            description="Place this shipment on hold."
            onClick={() => setSearchParams({ holdDialogOpen: true })}
          />
          <DropdownMenuItem
            title="Merge Shipment"
            description="Combine multiple shipments into one."
            disabled
          />
          <DropdownMenuItem
            title="Send to Worker"
            description="Assign this shipment for processing."
            disabled
          />
          <DropdownMenuLabel>Documentation & Communication</DropdownMenuLabel>
          <DropdownMenuSeparator />
          <DropdownMenuItem
            title="Add Document(s)"
            description="Attach relevant documents to this shipment."
            onClick={() => setSearchParams({ addDocumentDialogOpen: true })}
          />
          <DropdownMenuItem
            title="Add Comment(s)"
            description="Leave internal notes or comments on this shipment."
            disabled
          />
          <DropdownMenuLabel>View Actions</DropdownMenuLabel>
          <DropdownMenuSeparator />
          <DropdownMenuItem
            title="View Documents"
            description="Review attached shipment documents."
            onClick={() => setSearchParams({ documentDialogOpen: true })}
          />
          <DropdownMenuItem
            title="View Comments"
            description="Check comments and notes related to this shipment."
            disabled
          />
          <DropdownMenuItem
            title="View Audit Log"
            description="Track all modifications and updates to this shipment."
            onClick={() => setSearchParams({ auditDialogOpen: true })}
          /> */}
        </DropdownMenuContent>
      </DropdownMenu>
      {searchParams.duplicateDialogOpen && (
        <ShipmentDuplicateDialog
          open={searchParams.duplicateDialogOpen}
          onOpenChange={(open) =>
            setSearchParams({ duplicateDialogOpen: open })
          }
          shipment={shipment}
        />
      )}
      {searchParams.cancellationDialogOpen && (
        <ShipmentCancellationDialog
          open={searchParams.cancellationDialogOpen}
          onOpenChange={(open) =>
            setSearchParams({ cancellationDialogOpen: open })
          }
          shipmentId={shipment.id ?? ""}
        />
      )}
      {searchParams.unCancelDialogOpen && (
        <UnCancelShipmentDialog
          open={searchParams.unCancelDialogOpen}
          onOpenChange={(open) => setSearchParams({ unCancelDialogOpen: open })}
          shipmentId={shipment.id}
        />
      )}
      {searchParams.transferDialogOpen && (
        <TransferOwnershipDialog
          open={searchParams.transferDialogOpen}
          onOpenChange={(open) => setSearchParams({ transferDialogOpen: open })}
          shipmentId={shipment.id}
          currentOwnerId={shipment.ownerId}
        />
      )}
      {/*
      <EntryAuditViewer
        open={searchParams.auditDialogOpen}
        onOpenChange={(open) => setSearchParams({ auditDialogOpen: open })}
        resourceId={shipment.id ?? ""}
      />
      <ShipmentDocumentDialog
        open={searchParams.documentDialogOpen}
        onOpenChange={(open) => setSearchParams({ documentDialogOpen: open })}
        shipmentId={shipment.id}
      />
      <ShipmentDocumentWorkflow
        open={searchParams.addDocumentDialogOpen}
        onOpenChange={(open) =>
          setSearchParams({ addDocumentDialogOpen: open })
        }
        shipmentId={shipment.id}
        customerId={shipment.customerId}
        shipmentStatus={shipment.status}
      />
      <ShipmentHoldDialog
        open={searchParams.holdDialogOpen}
        onOpenChange={(open) => setSearchParams({ holdDialogOpen: open })}
        shipmentId={shipment.id}
      /> */}
    </>
  );
}
