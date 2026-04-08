import { DataTable } from "@/components/data-table/data-table";
import { apiService } from "@/services/api";
import type { AddRecordAction, DockAction, RowAction } from "@/types/data-table";
import { Resource } from "@/types/permission";
import type { Shipment } from "@/types/shipment";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import type { Row } from "@tanstack/react-table";
import {
  ArrowRightLeftIcon,
  BanIcon,
  CopyIcon,
  SendIcon,
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

  const { mutate: transferToBillingMutation } = useMutation({
    mutationFn: (shipmentId: string) => apiService.shipmentService.transferToBilling(shipmentId),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["shipment-list"] });
      toast.success("Transferred to billing", {
        description: "The shipment has been added to the billing queue.",
      });
    },
    onError: () => {
      toast.error("Failed to transfer shipment to billing");
    },
  });

  const handleTransferToBilling = useCallback(
    (row: Row<Shipment>) => transferToBillingMutation(row.original.id || ""),
    [transferToBillingMutation],
  );

  const handleBulkTransferToBilling = useCallback(
    async (rows: Shipment[]) => {
      const shipmentIds = rows.map((r) => r.id).filter(Boolean) as string[];
      const response = await apiService.shipmentService.bulkTransferToBilling(shipmentIds);

      if (response.errorCount > 0 && response.successCount > 0) {
        toast.warning(
          `Transferred ${response.successCount} of ${response.totalCount} shipments`,
          { description: `${response.errorCount} shipment(s) failed to transfer.` },
        );
      } else if (response.errorCount > 0) {
        toast.error("Failed to transfer shipments to billing");
      } else {
        toast.success(`Transferred ${response.successCount} shipment(s) to billing`);
      }

      await queryClient.invalidateQueries({ queryKey: ["shipment-list"], refetchType: "all" });
    },
    [queryClient],
  );

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
      {
        id: "transfer-to-billing",
        label: "Transfer to Billing",
        icon: SendIcon,
        onClick: handleTransferToBilling,
        hidden: (row) => {
          const s = row.original;
          if (s.status !== "ReadyToInvoice") return true;
          const bts = s.billingTransferStatus;
          return !!bts && bts !== "SentBackToOps";
        },
      },
    ],
    [handleDuplicate, handleCancel, handleUncancel, handleTransferOwnership, handleTransferToBilling],
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

  const dockActions = useMemo<DockAction<Shipment>[]>(
    () => [
      {
        id: "transfer-to-billing",
        label: "Transfer to Billing",
        loadingLabel: "Transferring...",
        icon: SendIcon,
        onClick: handleBulkTransferToBilling,
        clearSelectionOnSuccess: true,
      },
    ],
    [handleBulkTransferToBilling],
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
        dockActions={dockActions}
        enableRowSelection
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
