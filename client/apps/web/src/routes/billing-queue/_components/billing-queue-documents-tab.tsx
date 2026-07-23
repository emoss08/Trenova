import { DocumentTypeAutocompleteField } from "@/components/autocomplete-fields";
import { UploadPanel } from "@/components/documents/upload-panel";
import { Button } from "@/components/ui/button";
import { ScrollArea } from "@/components/ui/scroll-area";
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { useDocumentUpload } from "@/hooks/use-document-upload";
import { queries } from "@/lib/queries";
import { cn } from "@/lib/utils";
import { apiService } from "@/services/api";
import type { ShipmentBillingRequirement } from "@/types/shipment";
import type { Document } from "@/types/document";
import { useQuery } from "@tanstack/react-query";
import { formatDistanceToNowStrict, fromUnixTime } from "date-fns";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import {
  CheckCircle2Icon,
  CircleDashedIcon,
  FileIcon,
  FileTextIcon,
  ImageIcon,
  RefreshCwIcon,
  TrashIcon,
  UploadIcon,
} from "lucide-react";
import { useMemo, useState } from "react";
import { useForm, useWatch } from "react-hook-form";
import { toast } from "sonner";

function getFileIcon(fileType: string) {
  if (fileType.startsWith("image/")) return ImageIcon;
  if (fileType === "application/pdf") return FileTextIcon;
  return FileIcon;
}

function formatSize(bytes: number) {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}

export function BillingQueueDocumentsTab({
  shipmentId,
  selectedDocumentId,
  onDocumentSelect,
  isEditable,
  context = "billing-queue",
}: {
  shipmentId: string;
  selectedDocumentId: string | null;
  onDocumentSelect: (docId: string, fileName: string) => void;
  isEditable?: boolean;
  context?: "billing-queue" | "invoice";
}) {
  const [uploadOpen, setUploadOpen] = useState(false);
  const [replacingLineageId, setReplacingLineageId] = useState<string | null>(
    null,
  );
  const queryClient = useQueryClient();
  const docTypeForm = useForm<{ documentTypeId: string }>({
    defaultValues: { documentTypeId: "" },
  });
  const selectedDocTypeId = useWatch({
    control: docTypeForm.control,
    name: "documentTypeId",
  });

  const queryKey = [
    "documents",
    "shipment",
    shipmentId,
    "includeDocumentType",
  ] as const;

  const billingReadinessQuery = queries.shipment.billingReadiness(shipmentId);
  const { data: billingReadiness } = useQuery({
    ...billingReadinessQuery,
    enabled: !!shipmentId,
  });

  const { data: documents = [], isLoading } = useQuery({
    queryKey,
    queryFn: () =>
      apiService.documentService.getByResource(
        "shipment",
        shipmentId,
        undefined,
        {
          includeDocumentType: "true",
        },
      ),
    enabled: !!shipmentId,
  });

  const uploadMetadata = useMemo((): Record<string, string> => {
    const metadata: Record<string, string> = {};
    if (selectedDocTypeId) {
      metadata.documentTypeId = selectedDocTypeId;
    }
    if (replacingLineageId) {
      metadata.lineageId = replacingLineageId;
    }
    return metadata;
  }, [selectedDocTypeId, replacingLineageId]);

  const invalidateAll = () => {
    void queryClient.invalidateQueries({ queryKey });
    void queryClient.invalidateQueries({
      queryKey: billingReadinessQuery.queryKey,
    });
  };

  const { mutate: deleteDocument } = useMutation({
    mutationFn: (documentId: string) =>
      apiService.documentService.delete(documentId),
    onSuccess: () => {
      invalidateAll();
      toast.success("Document deleted");
    },
    onError: () => {
      toast.error("Failed to delete document");
    },
  });

  const {
    uploads,
    uploadFiles,
    cancelUpload,
    retryUpload,
    removeUpload,
    clearCompleted,
  } = useDocumentUpload({
    resourceId: shipmentId,
    resourceType: "shipment",
    uploadMetadata,
    invalidateQueryKey: queryKey,
    onSuccess: () => {
      docTypeForm.setValue("documentTypeId", "");
      setReplacingLineageId(null);
      void queryClient.invalidateQueries({
        queryKey: billingReadinessQuery.queryKey,
      });
    },
  });

  const handleUploadForRequirement = (
    requirement: ShipmentBillingRequirement,
  ) => {
    docTypeForm.setValue("documentTypeId", requirement.documentTypeId);
    setUploadOpen(true);
  };

  const requirementTitle =
    context === "invoice" ? "Supporting Requirements" : "Required Documents";
  const loadingLabel =
    context === "invoice"
      ? "Loading supporting documents..."
      : "Loading documents...";
  const emptyLabel =
    context === "invoice"
      ? "No supporting documents attached"
      : "No documents attached";

  return (
    <div className="flex h-full flex-col">
      {billingReadiness && billingReadiness.requirements.length > 0 && (
        <div className="shrink-0 border-b px-3 py-2">
          <div className="mb-1.5 flex items-center justify-between">
            <span className="text-[11px] font-medium tracking-wide text-muted-foreground uppercase">
              {requirementTitle}
            </span>
            <span className="text-[11px] text-muted-foreground tabular-nums">
              {billingReadiness.requirements.length -
                billingReadiness.missingRequirements.length}
              /{billingReadiness.requirements.length}
            </span>
          </div>
          <div className="flex flex-col gap-1">
            {billingReadiness.requirements.map((req) => (
              <div
                key={req.documentTypeId}
                className="flex items-center justify-between gap-2 text-xs"
              >
                <div className="flex min-w-0 items-center gap-1.5">
                  {req.satisfied ? (
                    <CheckCircle2Icon className="size-3.5 shrink-0 text-green-500" />
                  ) : (
                    <CircleDashedIcon className="size-3.5 shrink-0 text-muted-foreground" />
                  )}
                  <span
                    className={cn(
                      "truncate",
                      req.satisfied ? "text-muted-foreground" : "font-medium",
                    )}
                  >
                    {req.documentTypeName}
                  </span>
                </div>
                {!req.satisfied && isEditable && (
                  <Button
                    size="xxs"
                    variant="outline"
                    onClick={() => handleUploadForRequirement(req)}
                  >
                    Upload
                  </Button>
                )}
              </div>
            ))}
          </div>
        </div>
      )}
      {isEditable && (
        <div className="flex shrink-0 items-center gap-2 border-b p-2">
          <div className="flex-1">
            <DocumentTypeAutocompleteField
              control={docTypeForm.control}
              name="documentTypeId"
              placeholder="Document type (optional)"
              clearable
            />
          </div>
          <Button
            size="sm"
            variant="outline"
            onClick={() => setUploadOpen(true)}
          >
            <UploadIcon className="size-3.5" />
            Upload
          </Button>
        </div>
      )}

      {isLoading ? (
        <div className="flex items-center justify-center py-8 text-sm text-muted-foreground">
          {loadingLabel}
        </div>
      ) : documents.length === 0 ? (
        <div className="flex flex-col items-center justify-center gap-2 py-12 text-muted-foreground">
          <FileIcon className="size-8" />
          <p className="text-sm">{emptyLabel}</p>
        </div>
      ) : (
        <ScrollArea className="flex-1">
          <div className="flex flex-col gap-1 p-3">
            {documents.map((doc: Document) => {
              const Icon = getFileIcon(doc.fileType);
              const isSelected = doc.id === selectedDocumentId;

              return (
                <div
                  key={doc.id}
                  className={cn(
                    "group flex cursor-pointer items-center gap-3 rounded-md px-3 py-2 text-left transition-colors",
                    "hover:bg-accent/50",
                    isSelected && "border border-primary/20 bg-accent",
                  )}
                  onClick={() => onDocumentSelect(doc.id, doc.originalName)}
                  role="button"
                  tabIndex={0}
                  onKeyDown={(e) => {
                    if (e.key === "Enter")
                      onDocumentSelect(doc.id, doc.originalName);
                  }}
                >
                  <Icon className="size-4 shrink-0 text-muted-foreground" />
                  <div className="flex min-w-0 flex-1 flex-col">
                    <span className="truncate text-sm">{doc.originalName}</span>
                    <div className="flex items-center gap-1.5 text-[11px] text-muted-foreground">
                      <span>{formatSize(doc.fileSize)}</span>
                      <span>&middot;</span>
                      <span>
                        {formatDistanceToNowStrict(
                          fromUnixTime(doc.createdAt),
                          {
                            addSuffix: true,
                          },
                        )}
                      </span>
                      {doc.documentType && (
                        <>
                          <span>&middot;</span>
                          <span>{doc.documentType.name}</span>
                        </>
                      )}
                    </div>
                  </div>
                  {isEditable && (
                    <div className="flex shrink-0 items-center gap-0.5 opacity-0 transition-opacity group-hover:opacity-100">
                      <Tooltip>
                        <TooltipTrigger
                          render={
                            <Button
                              size="icon-xs"
                              variant="ghost"
                              onClick={(e) => {
                                e.stopPropagation();
                                setReplacingLineageId(doc.lineageId);
                                setUploadOpen(true);
                              }}
                            >
                              <RefreshCwIcon className="size-3" />
                            </Button>
                          }
                        />
                        <TooltipContent side="top" sideOffset={10}>
                          Replace
                        </TooltipContent>
                      </Tooltip>
                      <Tooltip>
                        <TooltipTrigger
                          render={
                            <Button
                              size="icon-xs"
                              variant="ghost"
                              onClick={(e) => {
                                e.stopPropagation();
                                deleteDocument(doc.id);
                              }}
                            >
                              <TrashIcon className="size-3" />
                            </Button>
                          }
                        />
                        <TooltipContent side="top" sideOffset={10}>
                          Delete
                        </TooltipContent>
                      </Tooltip>
                    </div>
                  )}
                </div>
              );
            })}
          </div>
        </ScrollArea>
      )}

      <UploadPanel
        isOpen={uploadOpen}
        onClose={() => {
          setUploadOpen(false);
          setReplacingLineageId(null);
        }}
        uploads={uploads}
        onFilesSelected={uploadFiles}
        onCancel={cancelUpload}
        onRetry={retryUpload}
        onRemove={removeUpload}
        onClearCompleted={clearCompleted}
        title={replacingLineageId ? "Replace Document" : undefined}
        description={
          replacingLineageId
            ? "Upload a new version to replace the existing document."
            : selectedDocTypeId
              ? "Uploads will be classified with the selected document type."
              : "Select a document type to classify uploads (optional)."
        }
      />
    </div>
  );
}
