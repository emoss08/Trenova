import { searchParamsParser } from "@/hooks/data-table/use-data-table-state";
import { api } from "@/lib/api";
import { apiService } from "@/services/api";
import type { Shipment } from "@/types/shipment";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import type { Row } from "@tanstack/react-table";
import { useQueryStates } from "nuqs";
import { useCallback, useMemo, useState } from "react";
import { toast } from "sonner";
import { CommandCenterTable } from "./command-center/command-center-table";
import { buildShipmentRowActions } from "./command-center/row-actions";
import { getMandatoryFieldFilters } from "./command-center/saved-views";
import { useCommandCenterUrl } from "./command-center/url-state";
import { ShipmentCancelDialog } from "./shipment-cancel-dialog";
import { getColumns } from "./shipment-columns";
import { ShipmentDuplicateDialog } from "./shipment-duplicate-dialog";
import { ShipmentPanel } from "./shipment-panel";
import { ShipmentTransferOwnershipDialog } from "./shipment-transfer-ownership-dialog";

const SHIPMENT_DETAIL_PARAMS = "?expandShipmentDetails=true";

export default function ShipmentTable() {
  const [duplicateShipmentId, setDuplicateShipmentId] = useState<string | null>(null);
  const [cancelShipmentId, setCancelShipmentId] = useState<string | null>(null);
  const [transferOwnershipShipmentId, setTransferOwnershipShipmentId] = useState<string | null>(
    null,
  );
  const queryClient = useQueryClient();

  const [{ view: selectedView, chips }] = useCommandCenterUrl();

  // Reuse the same nuqs parser the existing DataTable uses so deep-links to
  // ?panelType=edit&panelEntityId=<id> still open the shipment editor.
  const [searchParams, setSearchParams] = useQueryStates(searchParamsParser);
  const { panelType, panelEntityId } = searchParams;

  const isPanelOpen = panelType === "edit";

  const { data: panelRow } = useQuery({
    queryKey: ["shipment-list", "detail", panelEntityId],
    queryFn: () => api.get<Shipment>(`/shipments/${panelEntityId}/${SHIPMENT_DETAIL_PARAMS}`),
    enabled: !!panelEntityId && panelType === "edit",
    staleTime: 0,
  });

  const closePanel = useCallback(() => {
    void setSearchParams({ panelType: null, panelEntityId: null });
  }, [setSearchParams]);

  const handlePanelOpenChange = useCallback(
    (open: boolean) => {
      if (!open) closePanel();
    },
    [closePanel],
  );

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

  const handleEdit = useCallback(
    (row: Row<Shipment>) => {
      const id = row.original.id;
      if (!id) return;
      void setSearchParams({ panelType: "edit", panelEntityId: id });
    },
    [setSearchParams],
  );

  const handleTransferToBilling = useCallback(
    (row: Row<Shipment>) => transferToBillingMutation(row.original.id || ""),
    [transferToBillingMutation],
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

  const rowActions = useMemo(
    () =>
      buildShipmentRowActions({
        onEdit: handleEdit,
        onDuplicate: handleDuplicate,
        onCancel: handleCancel,
        onUncancel: handleUncancel,
        onTransferOwnership: handleTransferOwnership,
        onTransferToBilling: handleTransferToBilling,
      }),
    [
      handleEdit,
      handleDuplicate,
      handleCancel,
      handleUncancel,
      handleTransferOwnership,
      handleTransferToBilling,
    ],
  );

  const columns = useMemo(() => getColumns(rowActions), [rowActions]);

  const mandatoryFieldFilters = useMemo(
    () => getMandatoryFieldFilters(selectedView, chips),
    [selectedView, chips],
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
      <CommandCenterTable columns={columns} mandatoryFieldFilters={mandatoryFieldFilters} />
      <ShipmentPanel
        open={isPanelOpen}
        onOpenChange={handlePanelOpenChange}
        mode="edit"
        row={panelRow ?? null}
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
