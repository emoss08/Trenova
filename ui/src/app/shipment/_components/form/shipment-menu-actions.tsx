import { EntryAuditViewer } from "@/components/entity-audit-viewer/entry-audit-viewer";
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
import type { ShipmentSchema } from "@/lib/schemas/shipment-schema";
import { ShipmentStatus } from "@/types/shipment";
import { faEllipsisVertical } from "@fortawesome/pro-regular-svg-icons";
import { parseAsBoolean, useQueryState } from "nuqs";
import { ShipmentCancellationDialog } from "../cancellation/shipment-cancellatioin-dialog";
import { ShipmentDocumentDialog } from "../document/shipment-document-dialog";
import { ShipmentDocumentWorkflow } from "../document/shipment-document-workflow";
import { ShipmentDuplicateDialog } from "../duplicate/shipment-duplicate-dialog";

// Map of status that are allowed to be canceled.
const cancellatedStatuses = [
  ShipmentStatus.New,
  ShipmentStatus.InTransit,
  ShipmentStatus.Delayed,
  ShipmentStatus.PartiallyCompleted,
  ShipmentStatus.Completed,
];

const dialogs = {
  auditDialogOpen: parseAsBoolean.withDefault(false),
  documentDialogOpen: parseAsBoolean.withDefault(false),
  addDocumentDialogOpen: parseAsBoolean.withDefault(false),
  cancellationDialogOpen: parseAsBoolean.withDefault(false),
  duplicateDialogOpen: parseAsBoolean.withDefault(false),
};

export function ShipmentActions({
  shipment,
}: {
  shipment?: ShipmentSchema | null;
}) {
  const [cancellationDialogOpen, setCancellationDialogOpen] =
    useQueryState<boolean>(
      "cancellationDialogOpen",
      dialogs.cancellationDialogOpen.withOptions({}),
    );
  const [duplicateDialogOpen, setDuplicateDialogOpen] = useQueryState<boolean>(
    "duplicateDialogOpen",
    dialogs.duplicateDialogOpen.withOptions({}),
  );
  const [documentDialogOpen, setDocumentDialogOpen] = useQueryState<boolean>(
    "documentDialogOpen",
    dialogs.documentDialogOpen.withOptions({}),
  );
  const [auditDialogOpen, setAuditDialogOpen] = useQueryState<boolean>(
    "auditDialogOpen",
    dialogs.auditDialogOpen.withOptions({}),
  );
  const [addDocumentDialogOpen, setAddDocumentDialogOpen] =
    useQueryState<boolean>(
      "addDocumentDialogOpen",
      dialogs.addDocumentDialogOpen.withOptions({}),
    );

  if (!shipment) {
    return null;
  }

  const isCancellable = cancellatedStatuses.includes(shipment.status);

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
          />
          <DropdownMenuItem
            title="Duplicate"
            description="Create a copy of this shipment."
            onClick={() => setDuplicateDialogOpen(!duplicateDialogOpen)}
          />
          <DropdownMenuItem
            title="Cancel"
            description="Cancel this shipment and update its status."
            onClick={() => setCancellationDialogOpen(!cancellationDialogOpen)}
            disabled={!isCancellable}
          />
          <DropdownMenuLabel>Management Actions</DropdownMenuLabel>
          <DropdownMenuSeparator />
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
            onClick={() => setAddDocumentDialogOpen(!addDocumentDialogOpen)}
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
            onClick={() => setDocumentDialogOpen(!documentDialogOpen)}
          />
          <DropdownMenuItem
            title="View Comments"
            description="Check comments and notes related to this shipment."
            disabled
          />
          <DropdownMenuItem
            title="View Audit Log"
            description="Track all modifications and updates to this shipment."
            onClick={() => setAuditDialogOpen(!auditDialogOpen)}
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
      <EntryAuditViewer
        open={auditDialogOpen}
        onOpenChange={setAuditDialogOpen}
        resourceId={shipment.id ?? ""}
      />
      <ShipmentDocumentDialog
        open={documentDialogOpen}
        onOpenChange={setDocumentDialogOpen}
        shipmentId={shipment.id}
      />
      <ShipmentDocumentWorkflow
        open={addDocumentDialogOpen}
        onOpenChange={setAddDocumentDialogOpen}
        shipmentId={shipment.id}
        customerId={shipment.customerId}
        shipmentStatus={shipment.status}
      />
    </>
  );
}
