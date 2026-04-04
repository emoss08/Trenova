import { DataTable } from "@/components/data-table/data-table";
import { apiService } from "@/services/api";
import type { AddRecordAction, RowAction } from "@/types/data-table";
import { Resource } from "@/types/permission";
import type { Shipment } from "@/types/shipment";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import type { Row } from "@tanstack/react-table";
import {
  ArrowRightLeftIcon,
  BanIcon,
  CopyIcon,
  UndoIcon,
} from "lucide-react";
import { useCallback, useMemo, useState } from "react";
import { useNavigate } from "react-router";
import { toast } from "sonner";
import { ShipmentCancelDialog } from "./shipment-cancel-dialog";
import { getColumns } from "./shipment-columns";
import { ShipmentDuplicateDialog } from "./shipment-duplicate-dialog";
import { ShipmentPanel } from "./shipment-panel";
import { ShipmentTransferOwnershipDialog } from "./shipment-transfer-ownership-dialog";

export default function ShipmentTable() {
  const [duplicateShipmentId, setDuplicateShipmentId] = useState<string | null>(null);
  const [cancelShipmentId, setCancelShipmentId] = useState<string | null>(null);
  const [transferOwnershipShipmentId, setTransferOwnershipShipmentId] = useState<string | null>(
    null,
  );
  const queryClient = useQueryClient();
  const navigate = useNavigate();

  const { mutate: uncancelMutation } = useMutation({
    mutationFn: (shipmentId: string) => apiService.shipmentService.uncancel(shipmentId),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["shipment-list"] });
      toast.success("Shipment uncanceled", {
        description: "The shipment has been restored.",
      });
    },
    onError: () => {
      toast.error("Failed to uncancel shipment");
    },
  });

  const handleDuplicate = useCallback(
    (row: Row<Shipment>) => setDuplicateShipmentId(row.original.id || ""),
    [],
  );

  const handleCancel = useCallback(
    (row: Row<Shipment>) => setCancelShipmentId(row.original.id || ""),
    [],
  );

  const handleUncancel = useCallback(
    (row: Row<Shipment>) => uncancelMutation(row.original.id || ""),
    [uncancelMutation],
  );

  const handleTransferOwnership = useCallback(
    (row: Row<Shipment>) => setTransferOwnershipShipmentId(row.original.id || ""),
    [],
  );

  const columns = useMemo(() => getColumns(), []);

  const contextMenuActions = useMemo<RowAction<Shipment>[]>(
    () => [
      {
        id: "duplicate",
        label: "Duplicate",
        icon: CopyIcon,
        onClick: handleDuplicate,
      },
      {
        id: "cancel",
        label: "Cancel",
        icon: BanIcon,
        variant: "destructive",
        onClick: handleCancel,
        hidden: (row) => row.original.status === "Canceled",
      },
      {
        id: "uncancel",
        label: "Uncancel",
        icon: UndoIcon,
        onClick: handleUncancel,
        hidden: (row) => row.original.status !== "Canceled",
      },
      {
        id: "transfer-ownership",
        label: "Transfer Ownership",
        icon: ArrowRightLeftIcon,
        onClick: handleTransferOwnership,
        hidden: (row) => row.original.status === "Canceled",
      },
    ],
    [handleDuplicate, handleCancel, handleUncancel, handleTransferOwnership],
  );

  const addRecordActions = useMemo<AddRecordAction[]>(
    () => [
      {
        id: "import-rate-confirmation",
        label: "Import from Rate Confirmation",
        description: "Upload a rate confirmation and build a shipment draft from it.",
        onClick: () => navigate("/shipment-management/shipments/import"),
      },
    ],
    [navigate],
  );

  const handleDuplicateOpenChange = useCallback((open: boolean) => {
    if (!open) setDuplicateShipmentId(null);
  }, []);

  const handleCancelOpenChange = useCallback((open: boolean) => {
    if (!open) setCancelShipmentId(null);
  }, []);

  const handleTransferOwnershipOpenChange = useCallback((open: boolean) => {
    if (!open) setTransferOwnershipShipmentId(null);
  }, []);

  return (
    <>
      <DataTable<Shipment>
        name="Shipment"
        link="/shipments/"
        queryKey="shipment-list"
        exportModelName="shipment"
        resource={Resource.Shipment}
        columns={columns}
        TablePanel={ShipmentPanel}
        addRecordActions={addRecordActions}
        contextMenuActions={contextMenuActions}
        extraSearchParams={{
          expandShipmentDetails: true,
        }}
        preferDetailRowForEdit
      />
      {duplicateShipmentId && (
        <ShipmentDuplicateDialog
          open={!!duplicateShipmentId}
          onOpenChange={handleDuplicateOpenChange}
          shipmentId={duplicateShipmentId}
        />
      )}
      {cancelShipmentId && (
        <ShipmentCancelDialog
          open={!!cancelShipmentId}
          onOpenChange={handleCancelOpenChange}
          shipmentId={cancelShipmentId}
        />
      )}
      {transferOwnershipShipmentId && (
        <ShipmentTransferOwnershipDialog
          open={!!transferOwnershipShipmentId}
          onOpenChange={handleTransferOwnershipOpenChange}
          shipmentId={transferOwnershipShipmentId}
        />
      )}
    </>
  );
}
