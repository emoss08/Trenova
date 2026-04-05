import { SuspenseLoader } from "@/components/component-loader";
import { Button } from "@/components/ui/button";
import { useDocumentUpload } from "@/hooks/use-document-upload";
import { clearConversation } from "@/lib/import-chat-store";
import { apiService } from "@/services/api";
import { shipmentCreateSchema } from "@/types/shipment";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { ArrowLeftIcon } from "lucide-react";
import { nanoid } from "nanoid";
import { lazy, useCallback, useEffect, useMemo, useState } from "react";
import { useForm, useWatch } from "react-hook-form";
import { useNavigate } from "react-router";
import { toast } from "sonner";
import { ProcessingPhase } from "./processing-phase";
import { SuccessPhase } from "./success-phase";
import type { ReconciliationPhase, RequiredFieldsForm } from "./types";
import { UploadPhase } from "./upload-phase";
import { useReconciliationState } from "./use-reconciliation-state";

const IMPORT_RESOURCE_TYPE = "shipment_import";

const ReconciliationWorkspace = lazy(() => import("./reconciliation-workspace"));

function createImportResourceId() {
  return `shipment-import-${nanoid(12)}`;
}

export function ImportWorkspace() {
  const queryClient = useQueryClient();
  const navigate = useNavigate();
  const [importResourceId, setImportResourceId] = useState(createImportResourceId);
  const [uploadedDocumentId, setUploadedDocumentId] = useState<string | null>(null);
  const [createdShipmentId, setCreatedShipmentId] = useState<string | null>(null);
  const [attachErrorMessage, setAttachErrorMessage] = useState<string | null>(null);
  const [reconciliationInitialized, setReconciliationInitialized] = useState(false);
  const [lastCreateError, setLastCreateError] = useState<string | null>(null);

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
        toast.error(`Upload failed: ${error.message}`);
      },
    });

  const currentUpload = uploads[0] ?? null;

  const { data: importedDocument } = useQuery({
    queryKey: ["shipment-import-document", uploadedDocumentId],
    queryFn: () =>
      uploadedDocumentId
        ? apiService.documentService.getById(uploadedDocumentId)
        : Promise.resolve(null),
    enabled: !!uploadedDocumentId,
    refetchInterval: (query) => {
      const doc = query.state.data;
      if (!doc) return 1500;
      const waiting =
        doc.processingProfile === "rate_confirmation_import" &&
        (doc.contentStatus === "Pending" ||
          doc.contentStatus === "Extracting" ||
          doc.shipmentDraftStatus === "Pending");
      return waiting ? 1500 : false;
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
      setReconciliationInitialized(false);
      toast.success("Re-extraction started");
    },
    onError: (error) => {
      toast.error(`Failed to restart extraction: ${error.message}`);
    },
  });

  const reconciliation = useReconciliationState();

  const requiredFieldsForm = useForm<RequiredFieldsForm>({
    defaultValues: {
      customerId: "",
      serviceTypeId: "",
      shipmentTypeId: "",
      formulaTemplateId: "",
      tractorTypeId: "",
      trailerTypeId: "",
      stops: [],
    },
  });

  useEffect(() => {
    if (
      importedDraft?.status === "Ready" &&
      importedDraft.draftData &&
      !reconciliationInitialized
    ) {
      reconciliation.initialize(
        importedDraft.draftData,
        importedDraft.draftData.missingFields ?? [],
      );
      setReconciliationInitialized(true);

      const stops = importedDraft.draftData.stops ?? [];
      if (stops.length > 0) {
        requiredFieldsForm.setValue(
          "stops",
          stops.map(() => ({ locationId: "" })),
        );
      }
    }
  }, [importedDraft, reconciliationInitialized, reconciliation, requiredFieldsForm]);

  const currentPhase = useMemo<ReconciliationPhase>(() => {
    if (createdShipmentId) return "success";
    if (
      uploadedDocumentId &&
      importedDraft?.status === "Ready" &&
      !importedDraft.attachedShipmentId &&
      reconciliationInitialized
    ) {
      return "reconciliation";
    }
    if (uploadedDocumentId) return "processing";
    return "upload";
  }, [createdShipmentId, importedDraft, uploadedDocumentId, reconciliationInitialized]);

  const createShipment = useMutation({
    mutationFn: async () => {
      const requiredValues = requiredFieldsForm.getValues();

      const stopLocationIds = (requiredValues.stops ?? []).map((s) => s.locationId);
      reconciliation.state.stops.forEach((_, idx) => {
        const locId = stopLocationIds[idx];
        if (locId) {
          reconciliation.setStopLocation(idx, locId);
        }
      });

      const input = reconciliation.toShipmentCreateInput({
        customerId: requiredValues.customerId,
        serviceTypeId: requiredValues.serviceTypeId,
        shipmentTypeId: requiredValues.shipmentTypeId,
        formulaTemplateId: requiredValues.formulaTemplateId,
        tractorTypeId: requiredValues.tractorTypeId || undefined,
        trailerTypeId: requiredValues.trailerTypeId || undefined,
      });

      if (uploadedDocumentId) {
        input.sourceDocumentId = uploadedDocumentId;
      }

      const parsed = shipmentCreateSchema.parse(input);
      const shipment = await apiService.shipmentService.create(parsed);
      const shipmentId = shipment.id;
      if (!shipmentId) {
        throw new Error("Shipment was created without an ID");
      }

      let attachError: Error | null = null;
      if (importedDocument) {
        try {
          await apiService.documentService.attachToShipment(importedDocument.id, shipmentId);
        } catch (error) {
          attachError = error instanceof Error ? error : new Error("Failed to attach document");
        }
      }

      return { shipmentId, attachError };
    },
    onSuccess: ({ shipmentId, attachError }) => {
      void queryClient.invalidateQueries({ queryKey: ["shipment-list"] });
      setCreatedShipmentId(shipmentId);
      setAttachErrorMessage(attachError?.message ?? null);

      if (uploadedDocumentId) {
        void clearConversation(uploadedDocumentId);
      }

      if (attachError) {
        toast.warning("Shipment created, but document could not be attached");
      } else {
        toast.success("Shipment created from rate confirmation");
      }
    },
    onError: (error) => {
      toast.error(`Failed to create shipment: ${error.message}`);
      setLastCreateError(error.message);
    },
  });

  const resetFlow = useCallback(() => {
    clearAll();
    setUploadedDocumentId(null);
    setCreatedShipmentId(null);
    setAttachErrorMessage(null);
    setReconciliationInitialized(false);
    setImportResourceId(createImportResourceId());
    requiredFieldsForm.reset();
  }, [clearAll, requiredFieldsForm]);

  const handleBack = useCallback(() => {
    navigate("/shipment-management/shipments");
  }, [navigate]);

  const handleFilesSelected = useCallback(
    (files: File[]) => {
      const firstFile = files[0];
      if (!firstFile) return;
      uploadFiles([firstFile]);
    },
    [uploadFiles],
  );

  const watchedCustomerId = useWatch({ control: requiredFieldsForm.control, name: "customerId" });
  const watchedServiceTypeId = useWatch({
    control: requiredFieldsForm.control,
    name: "serviceTypeId",
  });
  const watchedShipmentTypeId = useWatch({
    control: requiredFieldsForm.control,
    name: "shipmentTypeId",
  });
  const watchedFormulaTemplateId = useWatch({
    control: requiredFieldsForm.control,
    name: "formulaTemplateId",
  });

  const requiredFieldValues = useMemo(
    () => ({
      customerId: watchedCustomerId ?? "",
      serviceTypeId: watchedServiceTypeId ?? "",
      shipmentTypeId: watchedShipmentTypeId ?? "",
      formulaTemplateId: watchedFormulaTemplateId ?? "",
    }),
    [watchedCustomerId, watchedServiceTypeId, watchedShipmentTypeId, watchedFormulaTemplateId],
  );

  const canCreateShipment =
    !!requiredFieldValues.customerId &&
    !!requiredFieldValues.serviceTypeId &&
    !!requiredFieldValues.shipmentTypeId &&
    !!requiredFieldValues.formulaTemplateId;

  const handleSetRequiredField = useCallback(
    (fieldKey: string, value: string) => {
      const validKeys = [
        "customerId",
        "serviceTypeId",
        "shipmentTypeId",
        "formulaTemplateId",
      ] as const;
      type ValidKey = (typeof validKeys)[number];
      if (validKeys.includes(fieldKey as ValidKey)) {
        requiredFieldsForm.setValue(fieldKey as ValidKey, value);
      }
    },
    [requiredFieldsForm],
  );

  const handleSetStopLocation = useCallback(
    (stopIndex: number, locationId: string) => {
      reconciliation.setStopLocation(stopIndex, locationId);
    },
    [reconciliation],
  );

  const handleSetStopSchedule = useCallback(
    (stopIndex: number, windowStart: string, windowEnd?: string) => {
      // The backend sends Unix timestamps as strings. Store them for toShipmentCreateInput.
      reconciliation.editStopField(stopIndex, "date", windowStart);
      if (windowEnd) {
        reconciliation.editStopField(stopIndex, "timeWindow", windowEnd);
      }
      // Also update the form stop data if it exists
      const stopsVal = requiredFieldsForm.getValues("stops");
      if (stopsVal && stopsVal[stopIndex]) {
        // Store the unix timestamp for later use
        requiredFieldsForm.setValue(
          `stops.${stopIndex}.locationId` as any,
          stopsVal[stopIndex].locationId,
        );
      }
    },
    [reconciliation, requiredFieldsForm],
  );

  const handleSetShipmentField = useCallback(
    (field: string, value: string) => {
      reconciliation.editField(field, value);
    },
    [reconciliation],
  );

  const handleShipmentCreated = useCallback(
    (shipmentId: string) => {
      setCreatedShipmentId(shipmentId);
      void queryClient.invalidateQueries({ queryKey: ["shipment-list"] });
      toast.success("Shipment created from rate confirmation");
    },
    [queryClient],
  );

  return (
    <div className="flex h-full flex-col">
      {/* Top bar */}
      <div className="flex shrink-0 items-center justify-between border-b bg-background px-4 py-2.5">
        <div className="flex items-center gap-3">
          <Button variant="ghost" size="icon-sm" onClick={handleBack}>
            <ArrowLeftIcon className="size-4" />
          </Button>
          <div>
            <h1 className="text-sm font-medium">Import from Rate Confirmation</h1>
            <p className="text-xs text-muted-foreground">
              {currentPhase === "upload" &&
                "Upload a rate confirmation to extract shipment details."}
              {currentPhase === "processing" && "Extracting shipment details from your document..."}
              {currentPhase === "reconciliation" &&
                "Review extracted fields, resolve issues, and create the shipment."}
              {currentPhase === "success" && "Import complete."}
            </p>
          </div>
        </div>
        {/* Empty — create button is in the reconciliation workspace footer */}
      </div>

      {/* Content area — fills remaining height */}
      {currentPhase === "upload" && (
        <UploadPhase
          currentUpload={
            currentUpload
              ? {
                  id: currentUpload.id,
                  file: currentUpload.file,
                  status: currentUpload.status,
                  progress: currentUpload.progress,
                  error: currentUpload.error,
                }
              : null
          }
          onFilesSelected={handleFilesSelected}
          onRetry={retryUpload}
          onCancel={cancelUpload}
          onRemove={removeUpload}
        />
      )}

      {currentPhase === "processing" && (
        <ProcessingPhase
          document={importedDocument}
          draft={importedDraft}
          fileName={importedDocument?.originalName ?? currentUpload?.file.name}
          onRetryExtraction={() => retryExtraction.mutate()}
          isRetrying={retryExtraction.isPending}
          onReplaceFile={resetFlow}
        />
      )}

      {currentPhase === "reconciliation" && uploadedDocumentId && (
        <SuspenseLoader componentLoaderProps={{ message: "Loading reconciliation workspace..." }}>
          <ReconciliationWorkspace
            documentId={uploadedDocumentId}
            fileName={importedDocument?.originalName}
            state={reconciliation.state}
            counts={reconciliation.counts}
            issueCount={reconciliation.issueCount}
            onAcceptField={reconciliation.acceptField}
            onEditField={reconciliation.editField}
            onResetField={reconciliation.resetField}
            onAcceptAllConfident={reconciliation.acceptAllConfident}
            onEditStopField={reconciliation.editStopField}
            requiredFieldsControl={requiredFieldsForm.control}
            canCreateShipment={canCreateShipment}
            isCreating={createShipment.isPending}
            onCreateShipment={() => createShipment.mutate()}
            hasRequiredValues={canCreateShipment}
            requiredFieldValues={requiredFieldValues}
            onSetRequiredField={handleSetRequiredField}
            onSetStopLocation={handleSetStopLocation}
            onSetStopSchedule={handleSetStopSchedule}
            onSetShipmentField={handleSetShipmentField}
            onShipmentCreated={handleShipmentCreated}
            lastCreateError={lastCreateError}
            onClearCreateError={() => setLastCreateError(null)}
          />
        </SuspenseLoader>
      )}

      {currentPhase === "success" && createdShipmentId && (
        <SuccessPhase
          shipmentId={createdShipmentId}
          attachError={attachErrorMessage}
          onDone={handleBack}
        />
      )}
    </div>
  );
}
