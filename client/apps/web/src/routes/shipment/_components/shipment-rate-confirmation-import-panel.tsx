import { DocumentShipmentDraftReviewDialog } from "@/components/documents/document-shipment-draft-review-dialog";
import { DocumentUploadZone, type RejectedFile } from "@/components/documents/document-upload-zone";
import { useDocumentUpload } from "@/hooks/use-document-upload";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Progress } from "@/components/ui/progress";
import { apiService } from "@/services/api";
import type { Document, DocumentShipmentDraft } from "@/types/document";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  AlertCircleIcon,
  CheckCircle2Icon,
  FileUpIcon,
  LoaderCircleIcon,
  SparklesIcon,
} from "lucide-react";
import { nanoid } from "nanoid";
import { useCallback, useEffect, useMemo, useState } from "react";
import { Link } from "react-router";
import { toast } from "sonner";

type ShipmentRateConfirmationImportPanelProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
};

type ImportStep = "upload" | "processing" | "review" | "success";

const IMPORT_RESOURCE_TYPE = "shipment_import";

function createImportResourceId() {
  return `shipment-import-${nanoid(12)}`;
}

function stepNumber(step: ImportStep) {
  switch (step) {
    case "upload":
      return 0;
    case "processing":
      return 1;
    case "review":
      return 2;
    case "success":
      return 3;
  }
}

function uploadStatusLabel(status?: string) {
  switch (status) {
    case "pending":
      return "Preparing upload";
    case "uploading":
      return "Uploading file";
    case "uploaded":
      return "Upload complete";
    case "verifying":
      return "Verifying upload";
    case "completing":
      return "Finalizing upload";
    case "retrying":
      return "Retrying upload";
    case "paused":
      return "Upload paused";
    case "error":
      return "Upload failed";
    case "success":
      return "Upload complete";
    default:
      return "Waiting for file";
  }
}

function processingSummary(
  document: Document | null | undefined,
  draft: DocumentShipmentDraft | null | undefined,
) {
  if (!document) {
    return {
      title: "Waiting for uploaded document",
      description: "The upload must complete before extraction can start.",
      progress: 10,
      variant: "default" as const,
    };
  }

  if (document.contentStatus === "Failed") {
    return {
      title: "Extraction failed",
      description: document.contentError || "We could not extract text from this rate confirmation.",
      progress: 100,
      variant: "error" as const,
    };
  }

  if (draft?.status === "Failed") {
    return {
      title: "Draft generation failed",
      description: draft.failureMessage || "We extracted text but could not build a shipment draft.",
      progress: 100,
      variant: "error" as const,
    };
  }

  if (draft?.status === "Ready") {
    return {
      title: "Shipment draft ready",
      description: "The extracted shipment draft is ready for review.",
      progress: 100,
      variant: "success" as const,
    };
  }

  if (
    document.shipmentDraftStatus === "Unavailable" &&
    document.contentStatus !== "Pending" &&
    document.contentStatus !== "Extracting"
  ) {
    return {
      title: "No shipment draft available",
      description: "This file did not produce a usable shipment draft.",
      progress: 100,
      variant: "error" as const,
    };
  }

  if (document.contentStatus === "Pending") {
    return {
      title: "Preparing extraction",
      description: "We are queuing OCR and intelligence work for this rate confirmation.",
      progress: 35,
      variant: "default" as const,
    };
  }

  if (document.contentStatus === "Extracting") {
    return {
      title: "Extracting shipment details",
      description: "We are extracting text, classifying the document, and assembling the shipment draft.",
      progress: 70,
      variant: "default" as const,
    };
  }

  if (document.shipmentDraftStatus === "Pending") {
    return {
      title: "Building shipment draft",
      description: "Extraction finished. We are mapping the results into a shipment draft now.",
      progress: 85,
      variant: "default" as const,
    };
  }

  return {
    title: "Processing rate confirmation",
    description: "We are still evaluating the uploaded document.",
    progress: 55,
    variant: "default" as const,
  };
}

export function ShipmentRateConfirmationImportPanel({
  open,
  onOpenChange,
}: ShipmentRateConfirmationImportPanelProps) {
  const queryClient = useQueryClient();
  const [importResourceId, setImportResourceId] = useState(createImportResourceId);
  const [uploadedDocumentId, setUploadedDocumentId] = useState<string | null>(null);
  const [createdShipmentId, setCreatedShipmentId] = useState<string | null>(null);
  const [attachErrorMessage, setAttachErrorMessage] = useState<string | null>(null);

  const {
    uploads,
    uploadFiles,
    cancelUpload,
    retryUpload,
    removeUpload,
    clearAll,
  } = useDocumentUpload({
    resourceId: importResourceId,
    resourceType: IMPORT_RESOURCE_TYPE,
    processingProfile: "rate_confirmation_import",
    invalidateQueryKey: ["documents", IMPORT_RESOURCE_TYPE, importResourceId],
    onSuccess: (document) => {
      setUploadedDocumentId(document.id);
    },
    onError: (error) => {
      toast.error(`Rate confirmation upload failed: ${error.message}`);
    },
  });

  const resetFlow = useCallback(() => {
    clearAll();
    setUploadedDocumentId(null);
    setCreatedShipmentId(null);
    setAttachErrorMessage(null);
    setImportResourceId(createImportResourceId());
  }, [clearAll]);

  const closeAndReset = useCallback(() => {
    resetFlow();
    onOpenChange(false);
  }, [onOpenChange, resetFlow]);

  const currentUpload = uploads[0] ?? null;

  const { data: importedDocument } = useQuery({
    queryKey: ["shipment-import-document", uploadedDocumentId],
    queryFn: () =>
      uploadedDocumentId
        ? apiService.documentService.getById(uploadedDocumentId)
        : Promise.resolve(null),
    enabled: !!uploadedDocumentId,
    refetchInterval: (query) => {
      const document = query.state.data;
      if (!document) return 1500;

      const waitingForDraft =
        document.processingProfile === "rate_confirmation_import" &&
        (document.contentStatus === "Pending" ||
          document.contentStatus === "Extracting" ||
          document.shipmentDraftStatus === "Pending");

      return waitingForDraft ? 1500 : false;
    },
  });

  const { data: importedDraft } = useQuery({
    queryKey: ["shipment-import-draft", uploadedDocumentId, importedDocument?.shipmentDraftStatus],
    queryFn: async () => {
      if (!uploadedDocumentId) return null;
      try {
        return await apiService.documentService.getShipmentDraft(uploadedDocumentId);
      } catch {
        return null;
      }
    },
    enabled:
      !!uploadedDocumentId &&
      importedDocument?.shipmentDraftStatus !== undefined &&
      importedDocument.shipmentDraftStatus !== "Unavailable",
    refetchInterval: (query) => {
      const draft = query.state.data;
      if (!uploadedDocumentId) return false;
      if (!draft || draft.status === "Pending") return 1500;
      return false;
    },
  });

  const retryExtraction = useMutation({
    mutationFn: async () => {
      if (!uploadedDocumentId) return;
      await apiService.documentService.reextract(uploadedDocumentId);
    },
    onSuccess: () => {
      if (uploadedDocumentId) {
        void queryClient.invalidateQueries({
          queryKey: ["shipment-import-document", uploadedDocumentId],
        });
        void queryClient.invalidateQueries({
          queryKey: ["shipment-import-draft", uploadedDocumentId],
        });
      }
      toast.success("Re-extraction started");
    },
    onError: (error) => {
      toast.error(`Failed to restart extraction: ${error.message}`);
    },
  });

  useEffect(() => {
    if (!open) {
      return;
    }

    if (!currentUpload && !uploadedDocumentId && !createdShipmentId) {
      setAttachErrorMessage(null);
    }
  }, [createdShipmentId, currentUpload, open, uploadedDocumentId]);

  const processingFailure = useMemo(() => {
    if (importedDocument?.contentStatus === "Failed") {
      return importedDocument.contentError || "We could not extract text from this rate confirmation.";
    }
    if (importedDraft?.status === "Failed") {
      return importedDraft.failureMessage || "We extracted the document but could not create a shipment draft.";
    }
    if (
      importedDocument &&
      importedDocument.shipmentDraftStatus === "Unavailable" &&
      importedDocument.contentStatus !== "Pending" &&
      importedDocument.contentStatus !== "Extracting"
    ) {
      return "This rate confirmation did not produce a usable shipment draft.";
    }
    return null;
  }, [importedDocument, importedDraft]);

  const currentStep = useMemo<ImportStep>(() => {
    if (createdShipmentId) {
      return "success";
    }
    if (uploadedDocumentId && importedDraft?.status === "Ready" && !importedDraft.attachedShipmentId) {
      return "review";
    }
    if (uploadedDocumentId) {
      return "processing";
    }
    return "upload";
  }, [createdShipmentId, importedDraft, uploadedDocumentId]);

  const steps = useMemo(
    () => [
      {
        key: "upload" as const,
        label: "Upload",
        description: "Select a rate confirmation file.",
      },
      {
        key: "processing" as const,
        label: "Process",
        description: "Extract shipment data and build the draft.",
      },
      {
        key: "review" as const,
        label: "Review",
        description: "Confirm the extracted shipment draft.",
      },
      {
        key: "success" as const,
        label: "Done",
        description: "Create the shipment and finish import.",
      },
    ],
    [],
  );

  const processStatus = processingSummary(importedDocument, importedDraft);

  const handleFilesSelected = useCallback(
    (files: File[]) => {
      const firstFile = files[0];
      if (!firstFile) return;
      uploadFiles([firstFile]);
    },
    [uploadFiles],
  );

  const handleRejectedFiles = useCallback((rejectedFiles: RejectedFile[]) => {
    rejectedFiles.forEach(({ file, reason }) => {
      if (reason === "size") {
        toast.error(`${file.name} is too large. Upload files up to 50 MB.`);
        return;
      }
      toast.error(`${file.name} is not a supported rate confirmation file.`);
    });
  }, []);

  const handleReplaceFile = useCallback(() => {
    resetFlow();
  }, [resetFlow]);

  return (
    <Dialog
      open={open}
      onOpenChange={(nextOpen) => {
        if (nextOpen) {
          onOpenChange(true);
        }
      }}
    >
      <DialogContent className="gap-0 overflow-hidden p-0 sm:max-w-6xl" showCloseButton={false}>
        <DialogHeader className="border-b px-6 pt-6 pb-4">
          <div className="flex items-start justify-between gap-4">
            <div className="space-y-2">
              <div className="flex items-center gap-2">
                <DialogTitle>Import from Rate Confirmation</DialogTitle>
                <Badge variant="secondary">Guided workflow</Badge>
              </div>
              <DialogDescription>
                Upload a rate confirmation, wait for extraction to finish, review the shipment
                draft, and create the shipment without leaving this flow.
              </DialogDescription>
            </div>
            {currentStep !== "success" ? (
              <Button variant="outline" onClick={closeAndReset}>
                Cancel Import
              </Button>
            ) : null}
          </div>
          <div className="mt-4 grid gap-3 md:grid-cols-4">
            {steps.map((step, index) => {
              const stepIndex = stepNumber(step.key);
              const activeIndex = stepNumber(currentStep);
              const isActive = stepIndex === activeIndex;
              const isComplete = stepIndex < activeIndex;

              return (
                <Card
                  key={step.key}
                  size="sm"
                  className={
                    isActive
                      ? "ring-primary/30"
                      : isComplete
                        ? "bg-emerald-50/40 ring-emerald-500/20"
                        : "bg-muted/20"
                  }
                >
                  <CardHeader className="gap-2">
                    <div className="flex items-center gap-2">
                      <div
                        className={
                          isComplete
                            ? "flex size-6 items-center justify-center rounded-full bg-emerald-500 text-white"
                            : isActive
                              ? "flex size-6 items-center justify-center rounded-full bg-primary text-primary-foreground"
                              : "flex size-6 items-center justify-center rounded-full bg-muted text-muted-foreground"
                        }
                      >
                        {isComplete ? <CheckCircle2Icon className="size-4" /> : index + 1}
                      </div>
                      <CardTitle>{step.label}</CardTitle>
                    </div>
                    <CardDescription>{step.description}</CardDescription>
                  </CardHeader>
                </Card>
              );
            })}
          </div>
        </DialogHeader>

        {currentStep === "upload" ? (
          <div className="grid gap-6 p-6">
            <Card>
              <CardHeader>
                <div className="flex items-center gap-2">
                  <FileUpIcon className="size-4" />
                  <CardTitle>Upload rate confirmation</CardTitle>
                </div>
                <CardDescription>
                  Use a PDF or image of the rate confirmation. We will extract shipment details only
                  for this import workflow.
                </CardDescription>
              </CardHeader>
              <CardContent className="grid gap-4">
                <DocumentUploadZone
                  onFilesSelected={handleFilesSelected}
                  onFilesRejected={handleRejectedFiles}
                  disabled={!!currentUpload && currentUpload.status !== "error"}
                  accept=".pdf,.jpg,.jpeg,.png,.webp"
                />
                {currentUpload ? (
                  <div className="rounded-lg border p-4">
                    <div className="flex items-center justify-between gap-3">
                      <div>
                        <div className="font-medium">{currentUpload.file.name}</div>
                        <div className="text-sm text-muted-foreground">
                          {uploadStatusLabel(currentUpload.status)}
                        </div>
                      </div>
                      {currentUpload.status === "uploading" ? (
                        <LoaderCircleIcon className="size-4 animate-spin text-primary" />
                      ) : null}
                    </div>
                    <div className="mt-4 grid gap-3">
                      <Progress
                        value={currentUpload.progress}
                        variant={currentUpload.status === "error" ? "error" : "default"}
                        showLabel
                      />
                      {currentUpload.error ? (
                        <div className="flex items-start gap-2 rounded-md border border-destructive/20 bg-destructive/5 p-3 text-sm text-destructive">
                          <AlertCircleIcon className="mt-0.5 size-4 shrink-0" />
                          <div>{currentUpload.error}</div>
                        </div>
                      ) : null}
                      <div className="flex flex-wrap gap-2">
                        {currentUpload.status === "error" ? (
                          <Button variant="outline" onClick={() => retryUpload(currentUpload.id)}>
                            Retry Upload
                          </Button>
                        ) : null}
                        {currentUpload.status !== "success" ? (
                          <Button
                            variant="outline"
                            onClick={() => cancelUpload(currentUpload.id)}
                          >
                            Cancel Upload
                          </Button>
                        ) : null}
                        {currentUpload.status === "error" ? (
                          <Button variant="ghost" onClick={() => removeUpload(currentUpload.id)}>
                            Remove File
                          </Button>
                        ) : null}
                      </div>
                    </div>
                  </div>
                ) : null}
              </CardContent>
            </Card>
          </div>
        ) : null}

        {currentStep === "processing" ? (
          <div className="grid gap-6 p-6">
            <Card>
              <CardHeader>
                <div className="flex items-center gap-2">
                  <SparklesIcon className="size-4" />
                  <CardTitle>{processStatus.title}</CardTitle>
                </div>
                <CardDescription>{processStatus.description}</CardDescription>
              </CardHeader>
              <CardContent className="grid gap-4">
                <Progress
                  value={processStatus.progress}
                  variant={processStatus.variant}
                  showLabel
                />
                <div className="grid gap-3 md:grid-cols-3">
                  <div className="rounded-lg border p-3">
                    <div className="text-xs font-medium tracking-wide text-muted-foreground uppercase">
                      Uploaded File
                    </div>
                    <div className="mt-1 text-sm">
                      {importedDocument?.originalName ?? currentUpload?.file.name ?? "Waiting"}
                    </div>
                  </div>
                  <div className="rounded-lg border p-3">
                    <div className="text-xs font-medium tracking-wide text-muted-foreground uppercase">
                      Content Status
                    </div>
                    <div className="mt-1 text-sm">
                      {importedDocument?.contentStatus ?? "Uploading"}
                    </div>
                  </div>
                  <div className="rounded-lg border p-3">
                    <div className="text-xs font-medium tracking-wide text-muted-foreground uppercase">
                      Draft Status
                    </div>
                    <div className="mt-1 text-sm">
                      {importedDocument?.shipmentDraftStatus ?? "Waiting"}
                    </div>
                  </div>
                </div>
                {processingFailure ? (
                  <div className="rounded-lg border border-destructive/20 bg-destructive/5 p-4">
                    <div className="flex items-start gap-3">
                      <AlertCircleIcon className="mt-0.5 size-4 shrink-0 text-destructive" />
                      <div className="grid gap-3">
                        <div>
                          <div className="font-medium text-destructive">Import failed</div>
                          <div className="text-sm text-destructive/80">{processingFailure}</div>
                        </div>
                        <div className="flex flex-wrap gap-2">
                          {uploadedDocumentId ? (
                            <Button
                              variant="outline"
                              onClick={() => retryExtraction.mutate()}
                              disabled={retryExtraction.isPending}
                            >
                              {retryExtraction.isPending ? (
                                <LoaderCircleIcon className="size-4 animate-spin" />
                              ) : null}
                              Retry Extraction
                            </Button>
                          ) : null}
                          <Button variant="outline" onClick={handleReplaceFile}>
                            Replace File
                          </Button>
                        </div>
                      </div>
                    </div>
                  </div>
                ) : (
                  <div className="rounded-lg border border-dashed p-4 text-sm text-muted-foreground">
                    Stay on this screen while we process the rate confirmation. The workflow will
                    advance automatically when the shipment draft is ready.
                  </div>
                )}
              </CardContent>
            </Card>
          </div>
        ) : null}

        {currentStep === "review" ? (
          <div className="grid gap-4">
            <div className="border-b px-6 pt-6 pb-4">
              <div className="text-sm font-medium">Review shipment draft</div>
              <div className="mt-1 text-sm text-muted-foreground">
                Confirm the extracted details, complete any missing shipment fields, and create the
                shipment from this draft.
              </div>
            </div>
            <DocumentShipmentDraftReviewDialog
              open
              onOpenChange={() => undefined}
              embedded
              document={importedDocument ?? null}
              draft={importedDraft ?? null}
              sourceResourceType={IMPORT_RESOURCE_TYPE}
              sourceResourceId={importResourceId}
              onShipmentCreated={({ shipmentId, attachError }) => {
                setCreatedShipmentId(shipmentId);
                setAttachErrorMessage(attachError?.message ?? null);
              }}
            />
          </div>
        ) : null}

        {currentStep === "success" ? (
          <div className="grid gap-6 p-6">
            <Card className="border-emerald-200 bg-emerald-50/60">
              <CardHeader>
                <div className="flex items-center gap-2">
                  <CheckCircle2Icon className="size-5 text-emerald-600" />
                  <CardTitle>Shipment created</CardTitle>
                </div>
                <CardDescription>
                  The rate confirmation workflow is complete.
                </CardDescription>
              </CardHeader>
              <CardContent className="grid gap-4">
                <div className="rounded-lg border border-emerald-200 bg-background/80 p-4">
                  <div className="text-xs font-medium tracking-wide text-muted-foreground uppercase">
                    Shipment ID
                  </div>
                  <div className="mt-1 text-sm font-medium">{createdShipmentId}</div>
                </div>
                {attachErrorMessage ? (
                  <div className="rounded-lg border border-amber-200 bg-amber-50 p-4 text-sm text-amber-950">
                    Shipment creation succeeded, but the source document could not be attached.
                    <div className="mt-1 text-amber-900/80">{attachErrorMessage}</div>
                  </div>
                ) : (
                  <div className="rounded-lg border border-emerald-200 bg-background/80 p-4 text-sm text-emerald-950">
                    The source document was attached to the new shipment successfully.
                  </div>
                )}
                <div className="flex flex-wrap gap-2">
                  <Button
                    variant="outline"
                    render={<Link to="/shipment-management/shipments" />}
                  >
                    Open Shipments
                  </Button>
                  <Button onClick={closeAndReset}>Done</Button>
                </div>
              </CardContent>
            </Card>
          </div>
        ) : null}
      </DialogContent>
    </Dialog>
  );
}
