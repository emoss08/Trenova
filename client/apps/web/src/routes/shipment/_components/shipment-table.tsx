import { formatFileSize, type RejectedFile } from "@/components/documents/document-upload-zone";
import { UploadPanel } from "@/components/documents/upload-panel";
import { panelSearchParamsParser } from "@/hooks/data-table/use-data-table-state";
import { useDocumentUpload } from "@/hooks/use-document-upload";
import { getShipmentGraphQL } from "@/lib/graphql/shipment";
import { queries } from "@/lib/queries";
import { apiService } from "@/services/api";
import { usePermissionStore } from "@trenova/shared/stores/permission-store";
import type { PanelMode } from "@trenova/shared/types/data-table";
import { Operation, Resource } from "@trenova/shared/types/permission";
import type { Shipment } from "@trenova/shared/types/shipment";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import type { Row } from "@tanstack/react-table";
import { useQueryStates } from "nuqs";
import { useCallback, useEffect, useMemo, useState } from "react";
import { toast } from "sonner";
import {
  CommandCenterTable,
  type CommandCenterTableSummary,
} from "./command-center/command-center-table";
import type { ShipmentDocumentUploadContext } from "./command-center/expanded-row/document-stack";
import { buildShipmentRowActions } from "./command-center/row-actions";
import { getMandatoryFieldFilters } from "./command-center/saved-views";
import { useCommandCenterUrl } from "./command-center/url-state";
import { ShipmentCancelDialog } from "./shipment-cancel-dialog";
import { getColumns } from "./shipment-columns";
import { ShipmentDuplicateDialog } from "./shipment-duplicate-dialog";
import { ShipmentSendEDIDialog } from "./shipment-send-edi-dialog";
import { ShipmentPanel } from "./shipment-panel";
import { ShipmentTransferOwnershipDialog } from "./shipment-transfer-ownership-dialog";

type ShipmentTableProps = {
  onSummaryChange?: (summary: CommandCenterTableSummary) => void;
};

export default function ShipmentTable({ onSummaryChange }: ShipmentTableProps) {
  const [duplicateShipmentId, setDuplicateShipmentId] = useState<string | null>(null);
  const [cancelShipmentId, setCancelShipmentId] = useState<string | null>(null);
  const [transferOwnershipShipmentId, setTransferOwnershipShipmentId] = useState<string | null>(
    null,
  );
  const [ediShipmentId, setEDIShipmentId] = useState<string | null>(null);
  const canSendEDI = usePermissionStore((state) =>
    state.hasPermission(Resource.EDI, Operation.Create),
  );
  const [uploadShipment, setUploadShipment] = useState<Shipment | null>(null);
  const [uploadDocumentType, setUploadDocumentType] =
    useState<ShipmentDocumentUploadContext | null>(null);
  const [isUploadOpen, setIsUploadOpen] = useState(false);
  const queryClient = useQueryClient();

  const [{ view: selectedView, chips }] = useCommandCenterUrl();

  // Reuse the same nuqs parser the existing DataTable uses so deep-links to
  // ?panelType=edit&panelEntityId=<id> still open the shipment editor.
  const [searchParams, setSearchParams] = useQueryStates(panelSearchParamsParser);
  const { panelType, panelEntityId } = searchParams;
  const panelMode: PanelMode = panelType ?? "create";
  const uploadShipmentId = uploadShipment?.id ?? "";
  const uploadDocumentsQueryKey = useMemo(
    () => ["documents", "shipment", uploadShipmentId] as const,
    [uploadShipmentId],
  );
  const uploadMetadata = useMemo((): Record<string, string> => {
    if (!uploadDocumentType) return {};
    return { documentTypeId: uploadDocumentType.documentTypeId };
  }, [uploadDocumentType]);
  const uploadBillingReadinessQuery = queries.shipment.billingReadiness(uploadShipmentId);

  const isPanelOpen = panelType === "edit" || panelType === "create";

  const { data: panelRow } = useQuery({
    queryKey: ["shipment-list", "detail", panelEntityId],
    queryFn: () => getShipmentGraphQL(panelEntityId ?? ""),
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

  const {
    uploads,
    uploadFiles,
    cancelUpload,
    retryUpload,
    removeUpload,
    clearCompleted,
    isUploading,
  } = useDocumentUpload({
    resourceId: uploadShipmentId,
    resourceType: "shipment",
    uploadMetadata,
    invalidateQueryKey: uploadDocumentsQueryKey,
    onSuccess: () => {
      void queryClient.invalidateQueries({
        queryKey: uploadBillingReadinessQuery.queryKey,
      });
      toast.success("Document uploaded successfully");
    },
    onError: (error) => {
      toast.error(`Upload failed: ${error.message}`);
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

  const handleUploadDocument = useCallback(
    (shipment: Shipment, context?: ShipmentDocumentUploadContext) => {
      if (!shipment.id) return;

      if (isUploading && uploadShipmentId && uploadShipmentId !== shipment.id) {
        setIsUploadOpen(true);
        toast.warning("Finish the current shipment upload before starting another.");
        return;
      }

      setUploadShipment(shipment);
      setUploadDocumentType(context ?? null);
      setIsUploadOpen(true);
    },
    [isUploading, uploadShipmentId],
  );

  const handleFilesSelected = useCallback(
    (files: File[]) => {
      if (!uploadShipmentId) return;
      uploadFiles(files);
    },
    [uploadFiles, uploadShipmentId],
  );

  const handleFilesRejected = useCallback((rejectedFiles: RejectedFile[]) => {
    rejectedFiles.forEach(({ file, reason }) => {
      if (reason === "size") {
        toast.error(`File too large: ${file.name}`, {
          description: `Maximum file size is 50MB. This file is ${formatFileSize(file.size)}.`,
        });
      }
    });
  }, []);

  const handleUploadClose = useCallback(() => {
    setIsUploadOpen(false);
    if (!isUploading) {
      setUploadShipment(null);
      setUploadDocumentType(null);
    }
  }, [isUploading]);

  useEffect(() => {
    if (isUploadOpen || isUploading) return;
    setUploadShipment(null);
    setUploadDocumentType(null);
  }, [isUploadOpen, isUploading]);

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
  const handleSendEDI = useCallback(
    (row: Row<Shipment>) => setEDIShipmentId(row.original.id || ""),
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
        onSendEDI: handleSendEDI,
        canSendEDI,
      }),
    [
      handleEdit,
      handleDuplicate,
      handleCancel,
      handleUncancel,
      handleTransferOwnership,
      handleTransferToBilling,
      handleSendEDI,
      canSendEDI,
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
  const handleEDIOpenChange = useCallback((open: boolean) => {
    if (!open) setEDIShipmentId(null);
  }, []);

  return (
    <>
      <CommandCenterTable
        columns={columns}
        rowActions={rowActions}
        mandatoryFieldFilters={mandatoryFieldFilters}
        onUploadDocument={handleUploadDocument}
        onSummaryChange={onSummaryChange}
      />
      <ShipmentPanel
        open={isPanelOpen}
        onOpenChange={handlePanelOpenChange}
        mode={panelMode}
        row={panelMode === "edit" ? (panelRow ?? null) : null}
      />
      <UploadPanel
        isOpen={isUploadOpen}
        onClose={handleUploadClose}
        uploads={uploads}
        onFilesSelected={handleFilesSelected}
        onFilesRejected={handleFilesRejected}
        onCancel={cancelUpload}
        onRetry={retryUpload}
        onRemove={removeUpload}
        onClearCompleted={clearCompleted}
        disabled={!uploadShipmentId}
        description={
          uploadDocumentType
            ? `This upload will be classified as ${uploadDocumentType.documentTypeName}.`
            : undefined
        }
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
      {ediShipmentId && (
        <ShipmentSendEDIDialog
          open={!!ediShipmentId}
          onOpenChange={handleEDIOpenChange}
          shipmentId={ediShipmentId}
        />
      )}
    </>
  );
}
