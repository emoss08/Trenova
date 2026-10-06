import { useT } from "@trenova/shared/i18n/use-t";
import { formatFileSize, type RejectedFile } from "@/components/documents/document-upload-zone";
import { UploadPanel } from "@/components/documents/upload-panel";
import { panelSearchParamsParser } from "@/hooks/data-table/use-data-table-state";
import { useDocumentUpload } from "@/hooks/use-document-upload";
import { useGuardedRowActions } from "@/hooks/use-pending-actions";
import { useShipmentBillingActions } from "@/hooks/use-shipment-billing-actions";
import { queries } from "@/lib/queries";
import { apiService } from "@/services/api";
import { usePermissionStore } from "@trenova/shared/stores/permission-store";
import type { PanelMode, Row, RowAction } from "@trenova/shared/types/data-table";
import { Operation, Resource } from "@trenova/shared/types/permission";
import type { Shipment } from "@trenova/shared/types/shipment";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useQueryStates } from "nuqs";
import { recordPath } from "@/config/record-links";
import {
  createContext,
  use,
  useCallback,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from "react";
import { toast } from "sonner";
import { buildShipmentRowActions } from "./row-actions";
import { ShipmentCancelDialog } from "../shipment-cancel-dialog";
import { ShipmentDuplicateDialog } from "../shipment-duplicate-dialog";
import { ShipmentSendEDIDialog } from "../shipment-send-edi-dialog";
import { ShipmentPanel } from "../shipment-panel";
import { shipmentPanelDetailQuery } from "../shipment-queries";
import { ShipmentTransferOwnershipDialog } from "../shipment-transfer-ownership-dialog";

export type ShipmentDocumentUploadContext = {
  documentTypeId: string;
  documentTypeName: string;
};

type ShipmentRecordActions = {
  rowActions: RowAction<Shipment>[];
  edit: (shipment: Shipment) => void;
  copyLink: (shipment: Shipment) => void;
  copyProNumber: (shipment: Shipment) => void;
  uploadDocument: (shipment: Shipment, context?: ShipmentDocumentUploadContext) => void;
  addComment: (shipment: Shipment) => void;
};

const RecordActionsContext = createContext<ShipmentRecordActions | null>(null);

export function useShipmentRecordActions(): ShipmentRecordActions {
  const value = use(RecordActionsContext);
  if (!value) {
    throw new Error("useShipmentRecordActions must be used inside ShipmentRecordActionsProvider");
  }
  return value;
}

function asRow(shipment: Shipment): Row<Shipment> {
  return { id: shipment.id ?? "", original: shipment } as Row<Shipment>;
}

async function copyText(text: string) {
  await navigator.clipboard.writeText(text);
}

/**
 * Everything a dispatcher can do to one shipment, and the dialogs those
 * actions open. The row menu, the right-click menu, the open row's quick
 * actions and the keyboard all reach the same handlers through it.
 */
export function ShipmentRecordActionsProvider({ children }: { children: ReactNode }) {
  const t = useT();

  const [duplicateShipmentId, setDuplicateShipmentId] = useState<string | null>(null);
  const [cancelShipmentId, setCancelShipmentId] = useState<string | null>(null);
  const [transferOwnershipShipmentId, setTransferOwnershipShipmentId] = useState<string | null>(
    null,
  );
  const [ediShipment, setEDIShipment] = useState<Shipment | null>(null);
  const canSendEDI = usePermissionStore((state) =>
    state.hasPermission(Resource.EDI, Operation.Create),
  );
  const [uploadShipment, setUploadShipment] = useState<Shipment | null>(null);
  const [uploadDocumentType, setUploadDocumentType] =
    useState<ShipmentDocumentUploadContext | null>(null);
  const [isUploadOpen, setIsUploadOpen] = useState(false);
  const queryClient = useQueryClient();

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
    ...shipmentPanelDetailQuery(panelEntityId ?? ""),
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

  const billingActions = useShipmentBillingActions();

  const { mutateAsync: uncancelShipment } = useMutation({
    mutationFn: (shipmentId: string) => apiService.shipmentService.uncancel(shipmentId),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["shipment-list"] });
      toast.success(t("Shipment uncanceled"), {
        description: t("The shipment has been restored."),
      });
    },
    onError: () => {
      toast.error(t("Failed to uncancel shipment"));
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
      toast.success(t("Document uploaded successfully"));
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
        toast.warning(t("Finish the current shipment upload before starting another."));
        return;
      }

      setUploadShipment(shipment);
      setUploadDocumentType(context ?? null);
      setIsUploadOpen(true);
    },
    [isUploading, uploadShipmentId, t],
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

  const handleDuplicate = useCallback(
    (row: Row<Shipment>) => setDuplicateShipmentId(row.original.id || ""),
    [],
  );

  const handleCancel = useCallback(
    (row: Row<Shipment>) => setCancelShipmentId(row.original.id || ""),
    [],
  );

  const handleUncancel = useCallback(
    (row: Row<Shipment>) => uncancelShipment(row.original.id || "").catch(() => undefined),
    [uncancelShipment],
  );

  const handleTransferOwnership = useCallback(
    (row: Row<Shipment>) => setTransferOwnershipShipmentId(row.original.id || ""),
    [],
  );
  const handleSendEDI = useCallback((row: Row<Shipment>) => setEDIShipment(row.original), []);

  const copyLink = useCallback(
    (shipment: Shipment) => {
      if (!shipment.id) return;
      const url = new URL(recordPath("shipment", shipment.id), window.location.origin).toString();
      void copyText(url).then(
        () => toast.success(t("Link to {0} copied", shipment.proNumber || shipment.id)),
        () => toast.error(t("The link could not be copied")),
      );
    },
    [t],
  );

  const copyProNumber = useCallback(
    (shipment: Shipment) => {
      if (!shipment.proNumber) return;
      void copyText(shipment.proNumber).then(
        () => toast.success(t("Copied {0}", shipment.proNumber)),
        () => toast.error(t("The PRO number could not be copied")),
      );
    },
    [t],
  );

  const handleCopyLink = useCallback((row: Row<Shipment>) => copyLink(row.original), [copyLink]);

  const handleOpenRecord = useCallback((row: Row<Shipment>) => {
    if (!row.original.id) return;
    window.open(recordPath("shipment", row.original.id), "_blank", "noopener");
  }, []);

  const addComment = useCallback(
    (shipment: Shipment) => {
      if (!shipment.id) return;
      void setSearchParams({ panelType: "edit", panelEntityId: shipment.id });
    },
    [setSearchParams],
  );

  const unguardedRowActions = useMemo(
    () =>
      buildShipmentRowActions({
        onEdit: handleEdit,
        onDuplicate: handleDuplicate,
        onCancel: handleCancel,
        onUncancel: handleUncancel,
        onTransferOwnership: handleTransferOwnership,
        billingActions,
        onSendEDI: handleSendEDI,
        onCopyLink: handleCopyLink,
        onOpenRecord: handleOpenRecord,
        canSendEDI,
      }),
    [
      handleEdit,
      handleDuplicate,
      handleCancel,
      handleUncancel,
      handleTransferOwnership,
      billingActions,
      handleSendEDI,
      handleCopyLink,
      handleOpenRecord,
      canSendEDI,
    ],
  );

  const rowActions = useGuardedRowActions(unguardedRowActions);

  const value = useMemo<ShipmentRecordActions>(
    () => ({
      rowActions,
      edit: (shipment) => handleEdit(asRow(shipment)),
      copyLink,
      copyProNumber,
      uploadDocument: handleUploadDocument,
      addComment,
    }),
    [rowActions, handleEdit, copyLink, copyProNumber, handleUploadDocument, addComment],
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
    if (!open) setEDIShipment(null);
  }, []);

  return (
    <RecordActionsContext value={value}>
      {children}
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
      {ediShipment && (
        <ShipmentSendEDIDialog
          open={!!ediShipment}
          onOpenChange={handleEDIOpenChange}
          shipment={ediShipment}
        />
      )}
    </RecordActionsContext>
  );
}
