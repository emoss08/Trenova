import { Autocomplete } from "@/components/fields/autocomplete/autocomplete";
import { fetchOptions } from "@/components/fields/autocomplete/autocomplete-content";
import { ColorOptionValue } from "@/components/fields/select-components";
import { useDocumentUpload } from "@/hooks/use-document-upload";
import { useLocalStorage } from "@/hooks/use-local-storage";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { queries } from "@/lib/queries";
import { apiService } from "@/services/api";
import type { Document, DocumentPacketSummary } from "@/types/document";
import type { DocumentType } from "@/types/document-type";
import type {
  Shipment,
  ShipmentBillingReadiness,
  ShipmentBillingRequirement,
} from "@/types/shipment";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { parseAsBoolean, useQueryState } from "nuqs";
import { useCallback, useDeferredValue, useMemo, useState } from "react";
import { toast } from "sonner";
import { DocumentBulkActionDock } from "./document-bulk-action-dock";
import { DocumentIntelligenceDialog } from "./document-intelligence-dialog";
import { DocumentList } from "./document-list";
import { DocumentVersionDialog } from "./document-version-dialog";
import {
  DocumentToolbar,
  type FileTypeFilter,
  type SortDirection,
  type SortField,
  type ViewMode,
} from "./document-toolbar";
import { formatFileSize, type RejectedFile } from "./document-upload-zone";
import { getFileCategory } from "./document-utils";
import { PacketCompletenessPanel } from "./packet-completeness-panel";
import { ShipmentBillingReadinessPanel } from "./shipment-billing-readiness-panel";
import { UploadPanel } from "./upload-panel";

interface DocumentsTabProps {
  resourceId: string;
  resourceType: string;
  disabled?: boolean;
}

function matchesFileTypeFilter(doc: Document, filter: FileTypeFilter): boolean {
  if (filter === "all") return true;

  const category = getFileCategory(doc.fileType, doc.originalName);

  switch (filter) {
    case "pdf":
      return category === "pdf";
    case "images":
      return category === "image";
    case "documents":
      return category === "document";
    case "spreadsheets":
      return category === "spreadsheet";
    default:
      return true;
  }
}

function hasTargetedProcessing(doc: Document): boolean {
  return doc.processingProfile === "rate_confirmation_import";
}

function sortDocuments(docs: Document[], field: SortField, direction: SortDirection): Document[] {
  const sorted = [...docs].sort((a, b) => {
    let comparison = 0;

    switch (field) {
      case "name":
        comparison = a.originalName.localeCompare(b.originalName);
        break;
      case "date":
        comparison = a.createdAt - b.createdAt;
        break;
      case "size":
        comparison = a.fileSize - b.fileSize;
        break;
    }

    return direction === "asc" ? comparison : -comparison;
  });

  return sorted;
}

export function DocumentsTab({ resourceId, resourceType, disabled = false }: DocumentsTabProps) {
  const queryClient = useQueryClient();
  const [deletingId, setDeletingId] = useState<string | null>(null);
  const [selectedIds, setSelectedIds] = useState<Set<string>>(new Set());
  const [inspectedDocument, setInspectedDocument] = useState<Document | null>(null);
  const [versionDocument, setVersionDocument] = useState<Document | null>(null);
  const [replacementLineageId, setReplacementLineageId] = useState<string | undefined>(undefined);

  const [isUploadOpen, setIsUploadOpen] = useQueryState(
    "upload",
    parseAsBoolean.withDefault(false),
  );

  const [viewMode, setViewMode] = useLocalStorage<ViewMode>("documents-view-mode", "grid");
  const [searchQuery, setSearchQuery] = useState("");
  const deferredSearchQuery = useDeferredValue(searchQuery);
  const [fileTypeFilter, setFileTypeFilter] = useState<FileTypeFilter>("all");
  const [sortField, setSortField] = useState<SortField>("date");
  const [sortDirection, setSortDirection] = useState<SortDirection>("desc");
  const [selectedDocumentTypeId, setSelectedDocumentTypeId] = useState<string | undefined>(
    undefined,
  );
  const [requiredUploadTypeId, setRequiredUploadTypeId] = useState<string | undefined>(undefined);

  const isShipment = resourceType === "shipment";

  const { data: documentTypesData } = useQuery({
    queryKey: ["document-types-select-options"],
    queryFn: () => fetchOptions<DocumentType>("/document-types/select-options/", "", 1, 100),
    enabled: isShipment,
    staleTime: 5 * 60 * 1000,
  });

  const documentTypeMap = useMemo(() => {
    const map = new Map<string, string>();
    for (const dt of documentTypesData?.results ?? []) {
      if (dt.id) {
        map.set(dt.id, dt.name);
      }
    }
    return map;
  }, [documentTypesData]);

  const uploadMetadata = useMemo((): Record<string, string> => {
    const metadata: Record<string, string> = {};
    if (isShipment && selectedDocumentTypeId) {
      metadata.documentTypeId = selectedDocumentTypeId;
    }
    if (replacementLineageId) {
      metadata.lineageId = replacementLineageId;
    }
    return metadata;
  }, [isShipment, replacementLineageId, selectedDocumentTypeId]);

  const queryKey = ["documents", resourceType, resourceId, deferredSearchQuery];

  const shipmentDetailsQuery = queries.shipment.get(resourceId, {
    expandShipmentDetails: "true",
  });
  const billingReadinessQuery = queries.shipment.billingReadiness(resourceId);

  const { data: shipment } = useQuery({
    queryKey: shipmentDetailsQuery.queryKey,
    queryFn: shipmentDetailsQuery.queryFn,
    enabled: isShipment && !!resourceId,
    refetchInterval: (query) => {
      const status = query.state.data?.status;
      if (!status || status === "Invoiced" || status === "ReadyToInvoice") return false;
      return 10_000;
    },
  });

  const { data: billingReadiness } = useQuery({
    queryKey: billingReadinessQuery.queryKey,
    queryFn: billingReadinessQuery.queryFn,
    enabled: isShipment && !!resourceId,
    refetchInterval: (query) => {
      const data = query.state.data;
      if (data?.canMarkReadyToInvoice && data.missingRequirements.length === 0) return false;
      return 10_000;
    },
  });

  const { data: documents = [], isLoading } = useQuery({
    queryKey,
    queryFn: () =>
      apiService.documentService.getByResource(resourceType, resourceId, deferredSearchQuery, { includeDocumentType: "true" }),
    enabled: !!resourceId,
    refetchInterval: (query) => {
      const docs = query.state.data;
      if (!docs) return false;

      const hasPendingWork = docs.some(
        (doc) =>
          doc.previewStatus === "Pending" ||
          (hasTargetedProcessing(doc) &&
            (doc.contentStatus === "Pending" ||
              doc.contentStatus === "Extracting" ||
              doc.shipmentDraftStatus === "Pending")),
      );

      return hasPendingWork ? 3000 : false;
    },
  });

  const { data: packetSummary } = useQuery<DocumentPacketSummary>({
    queryKey: ["document-packet-summary", resourceType, resourceId],
    queryFn: () => apiService.documentService.getPacketSummary(resourceType, resourceId),
    enabled: !!resourceId,
  });
  const versionDocumentID = versionDocument?.id;

  const { data: versionHistory = [], isLoading: isLoadingVersions } = useQuery({
    queryKey: ["document-versions", versionDocumentID],
    queryFn: () =>
      versionDocumentID
        ? apiService.documentService.getVersions(versionDocumentID)
        : Promise.resolve([]),
    enabled: !!versionDocumentID,
  });

  const filteredDocuments = useMemo(() => {
    let result = documents;

    result = result.filter((doc) => matchesFileTypeFilter(doc, fileTypeFilter));

    result = sortDocuments(result, sortField, sortDirection);

    return result;
  }, [documents, fileTypeFilter, sortField, sortDirection]);

  const { uploads, uploadFiles, cancelUpload, retryUpload, removeUpload, clearCompleted } =
    useDocumentUpload({
      resourceId,
      resourceType,
      uploadMetadata,
      onSuccess: (uploadedDocument) => {
        setReplacementLineageId(undefined);
        setRequiredUploadTypeId(undefined);
        if (isShipment && uploadedDocument.documentTypeId) {
          void queryClient
            .fetchQuery<ShipmentBillingReadiness>({
              queryKey: ["shipment", "billing-readiness", resourceId],
              queryFn: () => apiService.shipmentService.getBillingReadiness(resourceId),
            })
            .then((nextReadiness) => {
              const nextMissing = nextReadiness.missingRequirements[0];
              setSelectedDocumentTypeId(
                nextMissing?.documentTypeId ?? uploadedDocument.documentTypeId ?? undefined,
              );
            });
        }
        void queryClient.invalidateQueries({
          queryKey: billingReadinessQuery.queryKey,
        });
        void queryClient.invalidateQueries({
          queryKey: shipmentDetailsQuery.queryKey,
        });
        toast.success("Document uploaded successfully");
      },
      onError: (error) => {
        toast.error(`Upload failed: ${error.message}`);
      },
    });

  const deleteMutation = useMutation({
    mutationFn: (documentId: string) => apiService.documentService.delete(documentId),
    onSuccess: () => {
      void queryClient.invalidateQueries({
        queryKey: ["documents", resourceType, resourceId],
      });
      if (isShipment) {
        void queryClient.invalidateQueries({
          queryKey: billingReadinessQuery.queryKey,
        });
        void queryClient.invalidateQueries({
          queryKey: shipmentDetailsQuery.queryKey,
        });
      }
      toast.success("Document deleted");
      setDeletingId(null);
    },
    onError: (error) => {
      toast.error(`Delete failed: ${error.message}`);
      setDeletingId(null);
    },
  });

  const { mutateAsync: bulkDelete, isPending: isBulkDeleting } = useMutation({
    mutationFn: (documentIds: string[]) => apiService.documentService.bulkDelete(documentIds),
    onSuccess: (result) => {
      void queryClient.invalidateQueries({
        queryKey: ["documents", resourceType, resourceId],
      });
      if (isShipment) {
        void queryClient.invalidateQueries({
          queryKey: billingReadinessQuery.queryKey,
        });
        void queryClient.invalidateQueries({
          queryKey: shipmentDetailsQuery.queryKey,
        });
      }
      toast.success(`${result.deletedCount} document(s) deleted`);
      setSelectedIds(new Set());
    },
    onError: (error) => {
      toast.error(`Bulk delete failed: ${error.message}`);
    },
  });

  const restoreVersionMutation = useMutation({
    mutationFn: (documentId: string) => apiService.documentService.restoreVersion(documentId),
    onSuccess: () => {
      void queryClient.invalidateQueries({
        queryKey: ["documents", resourceType, resourceId],
      });
      void queryClient.invalidateQueries({
        queryKey: ["document-versions", versionDocumentID],
      });
      void queryClient.invalidateQueries({
        queryKey: ["document-packet-summary", resourceType, resourceId],
      });
      if (isShipment) {
        void queryClient.invalidateQueries({
          queryKey: billingReadinessQuery.queryKey,
        });
        void queryClient.invalidateQueries({
          queryKey: shipmentDetailsQuery.queryKey,
        });
      }
      toast.success("Document version restored");
    },
    onError: (error) => {
      toast.error(`Restore failed: ${error.message}`);
    },
  });

  const handleSelectDocument = useCallback((documentId: string) => {
    setSelectedIds((prev) => {
      const next = new Set(prev);
      if (next.has(documentId)) {
        next.delete(documentId);
      } else {
        next.add(documentId);
      }
      return next;
    });
  }, []);

  const handleClearSelection = useCallback(() => {
    setSelectedIds(new Set());
  }, []);

  const handleBulkDelete = useCallback(async () => {
    const ids = Array.from(selectedIds);
    await bulkDelete(ids);
  }, [selectedIds, bulkDelete]);

  const handleSelectAll = useCallback(() => {
    if (selectedIds.size === filteredDocuments.length) {
      setSelectedIds(new Set());
    } else {
      setSelectedIds(new Set(filteredDocuments.map((doc) => doc.id)));
    }
  }, [selectedIds.size, filteredDocuments]);

  const handleFilesSelected = useCallback(
    (files: File[]) => {
      if (disabled) return;
      uploadFiles(files);
    },
    [disabled, uploadFiles],
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

  const handlePreview = useCallback(async (document: Document) => {
    try {
      const url = await apiService.documentService.getViewUrl(document.id);
      window.open(url, "_blank");
    } catch {
      toast.error("Failed to open document");
    }
  }, []);

  const handleDownload = useCallback(async (document: Document) => {
    try {
      const url = await apiService.documentService.getDownloadUrl(document.id);
      window.open(url, "_blank");
    } catch {
      toast.error("Failed to get download URL");
    }
  }, []);

  const { mutate: deleteDocument } = deleteMutation;

  const handleInspect = useCallback((document: Document) => {
    setInspectedDocument(document);
  }, []);

  const handleOpenVersions = useCallback((document: Document) => {
    setVersionDocument(document);
  }, []);

  const handleUploadNewVersion = useCallback(
    (document: Document) => {
      setReplacementLineageId(document.lineageId);
      setRequiredUploadTypeId(undefined);
      if (document.documentTypeId) {
        setSelectedDocumentTypeId(document.documentTypeId);
      }
      setVersionDocument(null);
      void setIsUploadOpen(true);
    },
    [setIsUploadOpen],
  );

  const handleDelete = useCallback(
    (document: Document) => {
      setDeletingId(document.id);
      deleteDocument(document.id);
    },
    [deleteDocument],
  );

  const handleDrop = useCallback(
    (e: React.DragEvent) => {
      e.preventDefault();
      if (disabled) return;

      const files = Array.from(e.dataTransfer.files);
      if (files.length > 0) {
        setRequiredUploadTypeId(undefined);
        void setIsUploadOpen(true);
        uploadFiles(files);
      }
    },
    [disabled, setIsUploadOpen, uploadFiles],
  );

  const handleRequiredUpload = useCallback(
    (requirement: ShipmentBillingRequirement) => {
      setRequiredUploadTypeId(requirement.documentTypeId);
      setSelectedDocumentTypeId(requirement.documentTypeId);
      void setIsUploadOpen(true);
    },
    [setIsUploadOpen],
  );

  const { mutateAsync: markReadyToInvoice, isPending: isMarkingReady } = useApiMutation<
    Shipment,
    Shipment
  >({
    mutationFn: async (payload) => apiService.shipmentService.update(payload.id, payload),
    onSuccess: async () => {
      await Promise.all([
        queryClient.invalidateQueries({
          queryKey: shipmentDetailsQuery.queryKey,
        }),
        queryClient.invalidateQueries({
          queryKey: billingReadinessQuery.queryKey,
        }),
        queryClient.invalidateQueries({ queryKey: ["shipment-list"] }),
      ]);
      toast.success("Shipment marked ready to invoice");
    },
    resourceName: "Shipment",
  });

  const handleMarkReadyToInvoice = useCallback(async () => {
    if (!shipment) return;

    await markReadyToInvoice({
      ...shipment,
      status: "ReadyToInvoice",
    });
  }, [markReadyToInvoice, shipment]);

  if (!resourceId) {
    return (
      <div className="flex flex-col items-center justify-center py-12 text-center">
        <p className="text-sm text-muted-foreground">Save the record first to manage documents.</p>
      </div>
    );
  }

  return (
    <div className="space-y-4" onDragOver={(e) => e.preventDefault()} onDrop={handleDrop}>
      {packetSummary && packetSummary.totalRules > 0 && (
        <PacketCompletenessPanel summary={packetSummary} />
      )}

      {isShipment && billingReadiness && (
        <ShipmentBillingReadinessPanel
          readiness={billingReadiness}
          shipment={shipment}
          onUploadRequired={handleRequiredUpload}
          onMarkReadyToInvoice={() => {
            void handleMarkReadyToInvoice();
          }}
          isMarkingReady={isMarkingReady}
          disabled={disabled}
        />
      )}

      <DocumentToolbar
        searchQuery={searchQuery}
        onSearchChange={setSearchQuery}
        fileTypeFilter={fileTypeFilter}
        onFileTypeFilterChange={setFileTypeFilter}
        sortField={sortField}
        onSortFieldChange={setSortField}
        sortDirection={sortDirection}
        onSortDirectionChange={setSortDirection}
        viewMode={viewMode}
        onViewModeChange={setViewMode}
        onUploadClick={() => {
          setRequiredUploadTypeId(undefined);
          void setIsUploadOpen(true);
        }}
        disabled={disabled}
      />

      {isShipment && (
        <Autocomplete<DocumentType, Record<string, any>>
          link="/document-types/select-options/"
          extraSearchParams={{ documentCategory: "Shipment" }}
          value={selectedDocumentTypeId ?? null}
          onChange={(value: string | null) => setSelectedDocumentTypeId(value ?? undefined)}
          getOptionValue={(option) => option.id || ""}
          getDisplayValue={(option) => (
            <ColorOptionValue color={option.color ?? undefined} value={option.code} />
          )}
          renderOption={(option) => (
            <div className="flex size-full flex-col items-start">
              <ColorOptionValue color={option.color ?? undefined} value={option.code} />
              {option?.name && (
                <span className="w-full truncate text-2xs text-muted-foreground">
                  {option.name}
                </span>
              )}
            </div>
          )}
          placeholder="Document type (optional)"
          clearable
          triggerClassName="w-[220px]"
          disabled={!!requiredUploadTypeId}
        />
      )}

      <DocumentList
        documents={filteredDocuments}
        viewMode={viewMode}
        onPreview={handlePreview}
        onDownload={handleDownload}
        onDelete={handleDelete}
        onInspect={handleInspect}
        onVersions={handleOpenVersions}
        deletingId={deletingId}
        isLoading={isLoading}
        selectedIds={selectedIds}
        onSelectDocument={handleSelectDocument}
        documentTypeMap={isShipment ? documentTypeMap : undefined}
      />

      <DocumentBulkActionDock
        selectedCount={selectedIds.size}
        totalCount={filteredDocuments.length}
        onDelete={handleBulkDelete}
        onClearSelection={handleClearSelection}
        onSelectAll={handleSelectAll}
        isDeleting={isBulkDeleting}
      />

      <UploadPanel
        isOpen={isUploadOpen}
        onClose={() => {
          setReplacementLineageId(undefined);
          setRequiredUploadTypeId(undefined);
          void setIsUploadOpen(false);
        }}
        uploads={uploads}
        onFilesSelected={handleFilesSelected}
        onFilesRejected={handleFilesRejected}
        onCancel={cancelUpload}
        onRetry={retryUpload}
        onRemove={removeUpload}
        onClearCompleted={clearCompleted}
        disabled={disabled}
        title={replacementLineageId ? "Upload New Version" : undefined}
        description={
          requiredUploadTypeId && billingReadiness
            ? `This upload will be classified as ${
                billingReadiness.requirements.find(
                  (item) => item.documentTypeId === requiredUploadTypeId,
                )?.documentTypeName ?? "the selected required document"
              }.`
            : replacementLineageId
              ? "This file will be added as a new version in the same document lineage."
              : undefined
        }
        multiple={!replacementLineageId}
      />

      <DocumentIntelligenceDialog
        open={!!inspectedDocument}
        onOpenChange={(nextOpen) => {
          if (!nextOpen) {
            setInspectedDocument(null);
          }
        }}
        document={inspectedDocument}
        resourceType={resourceType}
        resourceId={resourceId}
      />
      <DocumentVersionDialog
        open={!!versionDocument}
        onOpenChange={(nextOpen) => {
          if (!nextOpen) {
            setVersionDocument(null);
          }
        }}
        document={versionDocument}
        versions={versionHistory}
        isLoading={isLoadingVersions}
        isRestoring={restoreVersionMutation.isPending}
        onRestore={(document) => restoreVersionMutation.mutate(document.id)}
        onUploadNewVersion={handleUploadNewVersion}
      />
    </div>
  );
}

export default DocumentsTab;
