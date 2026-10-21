import { useT } from "@trenova/shared/i18n/use-t";
import { DocumentShipmentDraftReviewDialog } from "@/components/documents/document-shipment-draft-review-dialog";
import { DocumentUploadZone, type RejectedFile } from "@/components/documents/document-upload-zone";
import {
  ExtractionNeedsProviderAlert,
  useDocumentExtractionAvailable,
} from "@/components/documents/extraction-needs-provider-alert";
import { useDocumentUpload } from "@/hooks/use-document-upload";
import { Alert, AlertDescription, AlertTitle } from "@trenova/shared/components/ui/alert";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { DescriptionItem, DescriptionList } from "@trenova/shared/components/ui/description-list";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@trenova/shared/components/ui/card";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@trenova/shared/components/ui/dialog";
import { Progress } from "@trenova/shared/components/ui/progress";
import { documentControl } from "@/lib/queries/document-control";
import { apiService } from "@/services/api";
import { cn } from "@trenova/shared/lib/utils";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  AlertCircleIcon,
  CheckCircleIcon,
  FileUploadIcon,
  SpinnerIcon,
} from "@trenova/shared/components/icons";
import { nanoid } from "nanoid";
import { useCallback, useMemo, useState } from "react";
import { Link } from "react-router";
import { toast } from "@trenova/shared/components/ui/toast";
import { AssistMark } from "@trenova/shared/components/ui/assist-mark";
import { isBlockedByProvider, processingSummary } from "./rate-confirmation-import-status";

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

function ImportRecoveryActions({
  canRetry,
  retrying,
  onRetry,
  onReplace,
  className,
}: {
  canRetry: boolean;
  retrying: boolean;
  onRetry: () => void;
  onReplace: () => void;
  className?: string;
}) {
  const t = useT();

  return (
    <div className={cn("flex flex-wrap gap-2", className)}>
      {canRetry ? (
        <Button variant="outline" onClick={onRetry} disabled={retrying}>
          {retrying ? <SpinnerIcon className="size-4 animate-spin" /> : null}
          {t("Retry extraction")}
        </Button>
      ) : null}
      <Button variant="outline" onClick={onReplace}>
        {t("Replace file")}
      </Button>
    </div>
  );
}

export function ShipmentRateConfirmationImportPanel({
  open,
  onOpenChange,
}: ShipmentRateConfirmationImportPanelProps) {
  const t = useT();

  const queryClient = useQueryClient();
  const [importResourceId, setImportResourceId] = useState(createImportResourceId);
  const [uploadedDocumentId, setUploadedDocumentId] = useState<string | null>(null);
  const [createdShipmentId, setCreatedShipmentId] = useState<string | null>(null);
  const [attachErrorMessage, setAttachErrorMessage] = useState<string | null>(null);

  const { uploads, uploadFiles, cancelUpload, retryUpload, removeUpload, clearAll } =
    useDocumentUpload({
      resourceId: importResourceId,
      resourceType: IMPORT_RESOURCE_TYPE,
      processingProfile: "rate_confirmation_import",
      invalidateQueryKey: ["documents", IMPORT_RESOURCE_TYPE, importResourceId],
      onSuccess: (document) => {
        setUploadedDocumentId(document.id);
      },
      onError: (error) => {
        toast.error(t("Rate confirmation upload failed: {0}", error.message));
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
      await queryClient.invalidateQueries({
        queryKey: documentControl.extractionStatus().queryKey,
        refetchType: "all",
      });
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
    },
    onError: (error) => {
      toast.error(t("Failed to restart extraction: {0}", error.message));
    },
  });

  const [idleCheck, setIdleCheck] = useState({
    createdShipmentId,
    currentUpload,
    open,
    uploadedDocumentId,
  });
  if (
    idleCheck.createdShipmentId !== createdShipmentId ||
    idleCheck.currentUpload !== currentUpload ||
    idleCheck.open !== open ||
    idleCheck.uploadedDocumentId !== uploadedDocumentId
  ) {
    setIdleCheck({ createdShipmentId, currentUpload, open, uploadedDocumentId });
    if (open && !currentUpload && !uploadedDocumentId && !createdShipmentId) {
      setAttachErrorMessage(null);
    }
  }

  const extractionAvailable = useDocumentExtractionAvailable();
  const blockedByProvider = isBlockedByProvider(importedDocument, extractionAvailable);

  const processingFailure = useMemo(() => {
    if (importedDocument?.contentStatus === "Failed") {
      return (
        importedDocument.contentError || "We could not extract text from this rate confirmation."
      );
    }
    if (importedDraft?.status === "Failed") {
      return (
        importedDraft.failureMessage ||
        "We extracted the document but could not create a shipment draft."
      );
    }
    if (
      importedDocument &&
      !blockedByProvider &&
      importedDocument.shipmentDraftStatus === "Unavailable" &&
      importedDocument.contentStatus !== "Pending" &&
      importedDocument.contentStatus !== "Extracting"
    ) {
      return "This rate confirmation did not produce a usable shipment draft.";
    }
    return null;
  }, [blockedByProvider, importedDocument, importedDraft]);

  const currentStep = useMemo<ImportStep>(() => {
    if (createdShipmentId) {
      return "success";
    }
    if (
      uploadedDocumentId &&
      importedDraft?.status === "Ready" &&
      !importedDraft.attachedShipmentId
    ) {
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
        label: t("Upload"),
        description: t("Select a rate confirmation file."),
      },
      {
        key: "processing" as const,
        label: t("Process"),
        description: t("Extract shipment data and build the draft."),
      },
      {
        key: "review" as const,
        label: t("Review"),
        description: t("Confirm the extracted shipment draft."),
      },
      {
        key: "success" as const,
        label: t("Done"),
        description: t("Create the shipment and finish import."),
      },
    ],
    [t],
  );

  const processStatus = processingSummary(importedDocument, importedDraft, extractionAvailable);

  const handleFilesSelected = useCallback(
    (files: File[]) => {
      const firstFile = files[0];
      if (!firstFile) return;
      uploadFiles([firstFile]);
    },
    [uploadFiles],
  );

  const handleRejectedFiles = useCallback(
    (rejectedFiles: RejectedFile[]) => {
      rejectedFiles.forEach(({ file, reason }) => {
        if (reason === "size") {
          toast.error(t("{0} is too large. Upload files up to 50 MB.", file.name));
          return;
        }
        toast.error(t("{0} is not a supported rate confirmation file.", file.name));
      });
    },
    [t],
  );

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
      <DialogContent size="2xl" className="gap-0 overflow-hidden p-0" showCloseButton={false}>
        <DialogHeader className="border-b px-6 pt-6 pb-4">
          <div className="flex items-start justify-between gap-4">
            <div className="space-y-2">
              <div className="flex items-center gap-2">
                <DialogTitle>{t("Import from rate confirmation")}</DialogTitle>
                <Badge variant="neutral">{t("Guided workflow")}</Badge>
              </div>
              <DialogDescription>
                {t(
                  "Upload a rate confirmation, wait for extraction to finish, review the shipment draft, and create the shipment without leaving this flow.",
                )}
              </DialogDescription>
            </div>
            {currentStep !== "success" ? (
              <Button variant="outline" onClick={closeAndReset}>
                {t("Cancel import")}
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
                      ? "ring-brand-border"
                      : isComplete
                        ? "bg-success-subtle ring-success-border"
                        : "bg-muted/20"
                  }
                >
                  <CardHeader className="gap-2">
                    <div className="flex items-center gap-2">
                      <div
                        className={
                          isComplete
                            ? "flex size-6 items-center justify-center rounded-full bg-success text-foreground-on-solid"
                            : isActive
                              ? "bg-primary text-primary-foreground flex size-6 items-center justify-center rounded-full"
                              : "bg-muted text-muted-foreground flex size-6 items-center justify-center rounded-full"
                        }
                      >
                        {isComplete ? <CheckCircleIcon className="size-4" /> : index + 1}
                      </div>
                      <CardTitle>{t(step.label)}</CardTitle>
                    </div>
                    <CardDescription>{t(step.description)}</CardDescription>
                  </CardHeader>
                </Card>
              );
            })}
          </div>
        </DialogHeader>

        {currentStep === "upload" ? (
          <div className="grid gap-6 p-6">
            <ExtractionNeedsProviderAlert />
            <Card>
              <CardHeader>
                <div className="flex items-center gap-2">
                  <FileUploadIcon className="size-4" />
                  <CardTitle>{t("Upload rate confirmation")}</CardTitle>
                </div>
                <CardDescription>
                  {t(
                    "Use a PDF or image of the rate confirmation. We will extract shipment details only for this import workflow.",
                  )}
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
                        <div className="text-muted-foreground text-sm">
                          {uploadStatusLabel(currentUpload.status)}
                        </div>
                      </div>
                      {currentUpload.status === "uploading" ? (
                        <SpinnerIcon className="text-primary size-4 animate-spin" />
                      ) : null}
                    </div>
                    <div className="mt-4 grid gap-3">
                      <Progress
                        value={currentUpload.progress}
                        variant={currentUpload.status === "error" ? "error" : "default"}
                        showLabel
                      />
                      {currentUpload.error ? (
                        <Alert variant="destructive" size="sm">
                          <AlertCircleIcon />
                          <AlertDescription>{currentUpload.error}</AlertDescription>
                        </Alert>
                      ) : null}
                      <div className="flex flex-wrap gap-2">
                        {currentUpload.status === "error" ? (
                          <Button variant="outline" onClick={() => retryUpload(currentUpload.id)}>
                            {t("Retry upload")}
                          </Button>
                        ) : null}
                        {currentUpload.status !== "success" ? (
                          <Button variant="outline" onClick={() => cancelUpload(currentUpload.id)}>
                            {t("Cancel upload")}
                          </Button>
                        ) : null}
                        {currentUpload.status === "error" ? (
                          <Button variant="ghost" onClick={() => removeUpload(currentUpload.id)}>
                            {t("Remove file")}
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
                  <AssistMark className="size-4" />
                  <CardTitle>{t(processStatus.title)}</CardTitle>
                </div>
                <CardDescription>{t(processStatus.description)}</CardDescription>
              </CardHeader>
              <CardContent className="grid gap-4">
                <Progress
                  value={processStatus.progress}
                  variant={processStatus.variant}
                  showLabel
                />
                <DescriptionList columns={3} className="rounded-lg border p-3">
                  <DescriptionItem label={t("Uploaded file")}>
                    {importedDocument?.originalName ?? currentUpload?.file.name ?? t("Waiting")}
                  </DescriptionItem>
                  <DescriptionItem label={t("Content status")}>
                    {importedDocument?.contentStatus ?? t("Uploading")}
                  </DescriptionItem>
                  <DescriptionItem label={t("Draft status")}>
                    {importedDocument?.shipmentDraftStatus ?? t("Waiting")}
                  </DescriptionItem>
                </DescriptionList>
                {processingFailure ? (
                  <Alert variant="destructive">
                    <AlertCircleIcon />
                    <AlertTitle>{t("Import failed")}</AlertTitle>
                    <AlertDescription>
                      {processingFailure}
                      <ImportRecoveryActions
                        className="mt-2"
                        canRetry={!!uploadedDocumentId}
                        retrying={retryExtraction.isPending}
                        onRetry={() => retryExtraction.mutate()}
                        onReplace={handleReplaceFile}
                      />
                    </AlertDescription>
                  </Alert>
                ) : blockedByProvider ? (
                  <div className="grid gap-3">
                    <ExtractionNeedsProviderAlert />
                    <ImportRecoveryActions
                      canRetry={!!uploadedDocumentId}
                      retrying={retryExtraction.isPending}
                      onRetry={() => retryExtraction.mutate()}
                      onReplace={handleReplaceFile}
                    />
                  </div>
                ) : (
                  <div className="text-muted-foreground rounded-lg border border-dashed p-4 text-sm">
                    {t(
                      "Stay on this screen while we process the rate confirmation. The workflow will advance automatically when the shipment draft is ready.",
                    )}
                  </div>
                )}
              </CardContent>
            </Card>
          </div>
        ) : null}

        {currentStep === "review" ? (
          <div className="grid gap-4">
            <div className="border-b px-6 pt-6 pb-4">
              <div className="text-sm font-medium">{t("Review shipment draft")}</div>
              <div className="text-muted-foreground mt-1 text-sm">
                {t(
                  "Confirm the extracted details, complete any missing shipment fields, and create the shipment from this draft.",
                )}
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
            <Card className="border-success-border bg-success-subtle">
              <CardHeader>
                <div className="flex items-center gap-2">
                  <CheckCircleIcon className="size-5 text-success-foreground" />
                  <CardTitle>{t("Shipment created")}</CardTitle>
                </div>
                <CardDescription>
                  {t("The rate confirmation workflow is complete.")}
                </CardDescription>
              </CardHeader>
              <CardContent className="grid gap-4">
                <DescriptionList
                  columns={1}
                  className="bg-card rounded-lg border border-success-border p-4"
                >
                  <DescriptionItem label={t("Shipment ID")} valueClassName="font-mono">
                    {createdShipmentId}
                  </DescriptionItem>
                </DescriptionList>
                {attachErrorMessage ? (
                  <Alert variant="warning" size="sm">
                    <AlertCircleIcon />
                    <AlertTitle>
                      {t(
                        "Shipment creation succeeded, but the source document could not be attached.",
                      )}
                    </AlertTitle>
                    <AlertDescription>{attachErrorMessage}</AlertDescription>
                  </Alert>
                ) : (
                  <Alert variant="success" size="sm" className="bg-card">
                    <CheckCircleIcon />
                    <AlertDescription>
                      {t("The source document was attached to the new shipment successfully.")}
                    </AlertDescription>
                  </Alert>
                )}
                <div className="flex flex-wrap gap-2">
                  <Button variant="outline" render={<Link to="/shipment-management/shipments" />}>
                    {t("Open shipments")}
                  </Button>
                  <Button onClick={closeAndReset}>{t("Done")}</Button>
                </div>
              </CardContent>
            </Card>
          </div>
        ) : null}
      </DialogContent>
    </Dialog>
  );
}
