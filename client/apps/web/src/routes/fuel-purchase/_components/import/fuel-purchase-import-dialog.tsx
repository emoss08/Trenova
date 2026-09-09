import { FuelCardAutocompleteField } from "@/components/autocomplete-fields";
import { DocumentUploadZone } from "@/components/documents/document-upload-zone";
import { InputField } from "@/components/fields/input-field";
import { SelectField } from "@/components/fields/select-field";
import { handleMutationError } from "@/hooks/use-api-mutation";
import { useDocumentUpload } from "@/hooks/use-document-upload";
import { fuelCardProviderChoices, iftaFuelTypeChoices } from "@/lib/choices";
import { downloadCsv } from "@/lib/data-table-export";
import {
  canCommitImport,
  commitLabel,
  discardImportNotice,
  importHeadline,
  importStep,
  importWarnings,
  needsDiscardConfirmation,
  type ImportRowFilter,
} from "@/lib/fuel-purchase-import";
import { FUEL_PURCHASE_LIST_KEY } from "@/lib/graphql/fuel-purchase";
import {
  commitFuelPurchaseImport,
  createFuelPurchaseImport,
  discardFuelPurchaseImport,
  fetchFuelPurchaseImportTemplate,
  stageFuelPurchaseImport,
  type FuelPurchaseImportBatch,
} from "@/lib/graphql/fuel-purchase-import";
import { zodResolver } from "@hookform/resolvers/zod";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import type {
  CreateFuelPurchaseImportInput,
  StageFuelPurchaseImportInput,
} from "@trenova/graphql/generated/graphql";
import { FuelPurchaseImportStatusBadge } from "@trenova/shared/components/status-badge";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogMedia,
  AlertDialogTitle,
} from "@trenova/shared/components/ui/alert-dialog";
import { Alert, AlertDescription, AlertTitle } from "@trenova/shared/components/ui/alert";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@trenova/shared/components/ui/dialog";
import { Form, FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import { Progress } from "@trenova/shared/components/ui/progress";
import type { Document } from "@trenova/shared/types/document";
import { blankToNull, formatCurrency } from "@trenova/shared/lib/utils";
import {
  fuelPurchaseImportSetupSchema,
  type FuelPurchaseImportSetupValues,
} from "@trenova/shared/types/fuel-purchase";
import { CircleAlertIcon, CircleCheckIcon, DownloadIcon, Trash2Icon } from "lucide-react";
import { useCallback, useEffect, useRef, useState } from "react";
import { FormProvider, useForm, useWatch, type Resolver } from "react-hook-form";
import { toast } from "sonner";
import { ImportReviewTable } from "./import-review-table";

const STEPS = [
  { id: "setup", label: "Set up" },
  { id: "review", label: "Review" },
  { id: "done", label: "Done" },
] as const;

const SETUP_DEFAULTS: FuelPurchaseImportSetupValues = {
  provider: "Comdata",
  defaultFuelCardId: null,
  defaultFuelType: "Diesel",
  defaultCurrency: "USD",
};

type FuelPurchaseImportDialogProps = {
  readonly open: boolean;
  readonly onOpenChange: (open: boolean) => void;
};

type StatementUploaderProps = {
  batch: FuelPurchaseImportBatch;
  file: File;
  onUploaded: (document: Document) => void;
  onFailed: (message: string) => void;
};

function StatementUploader({ batch, file, onUploaded, onFailed }: StatementUploaderProps) {
  const { uploads, uploadFiles } = useDocumentUpload({
    resourceId: batch.id,
    resourceType: "fuel_purchase_import",
    onSuccess: onUploaded,
    onError: (error) => onFailed(error.message),
  });
  const started = useRef(false);

  useEffect(() => {
    if (started.current) return;
    started.current = true;
    uploadFiles([file]);
  }, [file, uploadFiles]);

  const upload = uploads[0];
  const progress = upload?.progress ?? 0;
  const label = upload?.status === "success" ? "Reading the statement…" : `Uploading ${file.name}…`;

  return (
    <div className="flex flex-col gap-2 rounded-lg border p-3" aria-busy="true">
      <p className="text-sm font-medium">{label}</p>
      <Progress value={progress} aria-label="Upload progress" />
      <p className="text-muted-foreground text-xs">
        {progress}% · Nothing is recorded until you review the rows and confirm.
      </p>
    </div>
  );
}

function StepIndicator({ current }: { current: (typeof STEPS)[number]["id"] }) {
  const currentIndex = STEPS.findIndex((step) => step.id === current);
  return (
    <ol className="flex items-center gap-2 text-xs" aria-label="Import steps">
      {STEPS.map((step, index) => (
        <li key={step.id} className="flex items-center gap-2">
          <span
            aria-current={step.id === current ? "step" : undefined}
            className={
              index <= currentIndex ? "text-foreground font-medium" : "text-muted-foreground"
            }
          >
            {index + 1}. {step.label}
          </span>
          {index < STEPS.length - 1 ? <span className="text-muted-foreground">›</span> : null}
        </li>
      ))}
    </ol>
  );
}

function SummaryChips({ batch }: { batch: FuelPurchaseImportBatch }) {
  const summary = batch.summary;
  if (!summary) return null;
  const chips: Array<{ label: string; value: string }> = [
    { label: "Rows", value: String(summary.rowCount) },
    { label: "New", value: String(summary.newCount) },
    { label: "Duplicates in file", value: String(summary.duplicateInFileCount) },
    { label: "Already on file", value: String(summary.alreadyImportedCount) },
    { label: "Errors", value: String(summary.errorCount) },
    { label: "Gallons", value: summary.totalGallons },
    {
      label: "Amount",
      value: formatCurrency(Number(summary.totalAmount), batch.defaultCurrency),
    },
  ];
  return (
    <div className="flex flex-wrap gap-1.5">
      {chips.map((chip) => (
        <Badge key={chip.label} variant="secondary" className="gap-1 px-2 py-0.5 text-[11px]">
          <span className="text-muted-foreground">{chip.label}</span>
          <span className="tabular-nums">{chip.value}</span>
        </Badge>
      ))}
    </div>
  );
}

export function FuelPurchaseImportDialog({ open, onOpenChange }: FuelPurchaseImportDialogProps) {
  if (!open) return null;
  return <ImportSession onOpenChange={onOpenChange} />;
}

function ImportSession({ onOpenChange }: Pick<FuelPurchaseImportDialogProps, "onOpenChange">) {
  const queryClient = useQueryClient();
  const [batch, setBatch] = useState<FuelPurchaseImportBatch | undefined>();
  const [pendingFile, setPendingFile] = useState<File | null>(null);
  const [problem, setProblem] = useState<string | null>(null);
  const [filter, setFilter] = useState<ImportRowFilter>("all");
  const [discardPrompt, setDiscardPrompt] = useState<"close" | "stay" | null>(null);

  const form = useForm<FuelPurchaseImportSetupValues>({
    resolver: zodResolver(fuelPurchaseImportSetupSchema) as Resolver<FuelPurchaseImportSetupValues>,
    defaultValues: SETUP_DEFAULTS,
  });
  const provider = useWatch({ control: form.control, name: "provider" });
  const { reset: resetForm } = form;

  const resetAll = useCallback(() => {
    setBatch(undefined);
    setPendingFile(null);
    setProblem(null);
    setFilter("all");
    setDiscardPrompt(null);
    resetForm(SETUP_DEFAULTS);
  }, [resetForm]);

  const { mutate: createBatch, isPending: creating } = useMutation({
    mutationFn: (input: CreateFuelPurchaseImportInput) => createFuelPurchaseImport(input),
    onSuccess: (created) => setBatch(created),
    onError: (error) => {
      setPendingFile(null);
      handleMutationError({ error, form, resourceName: "Statement import" });
    },
  });

  const { mutate: stageBatch, isPending: staging } = useMutation({
    mutationFn: (input: StageFuelPurchaseImportInput) => stageFuelPurchaseImport(input),
    onSuccess: (staged) => {
      setBatch(staged);
      setPendingFile(null);
      setFilter(staged.status === "Failed" ? "errors" : "all");
    },
    onError: (error) => {
      setPendingFile(null);
      setProblem(error instanceof Error ? error.message : "The statement could not be staged.");
    },
  });

  const { mutate: commitBatch, isPending: committing } = useMutation({
    mutationFn: (current: FuelPurchaseImportBatch) =>
      commitFuelPurchaseImport(current.id, current.version),
    onSuccess: async (committed) => {
      setBatch(committed);
      toast.success("Statement imported", {
        description: `${committed.committedCount} ${committed.committedCount === 1 ? "purchase" : "purchases"} recorded.`,
      });
      await queryClient.invalidateQueries({ queryKey: [FUEL_PURCHASE_LIST_KEY] });
    },
    onError: (error) => handleMutationError({ error, resourceName: "Statement import" }),
  });

  const { mutate: discardBatch, isPending: discarding } = useMutation({
    mutationFn: (current: FuelPurchaseImportBatch) =>
      discardFuelPurchaseImport(current.id, current.version),
    onError: (error) => handleMutationError({ error, resourceName: "Statement import" }),
  });

  const { mutate: downloadTemplate, isPending: downloadingTemplate } = useMutation({
    mutationFn: () => fetchFuelPurchaseImportTemplate(provider),
    onSuccess: ({ fileName, content }) => downloadCsv(content, fileName),
    onError: () => toast.error("The template could not be downloaded"),
  });

  const handleFile = useCallback(
    async (file: File) => {
      const valid = await form.trigger();
      if (!valid) return;
      const values = form.getValues();
      setProblem(null);
      setPendingFile(file);
      createBatch({
        provider: values.provider,
        defaultFuelCardId: blankToNull(values.defaultFuelCardId),
        defaultFuelType: values.defaultFuelType,
        defaultCurrency: values.defaultCurrency,
      });
    },
    [createBatch, form],
  );

  const handleUploaded = useCallback(
    (document: Document) => {
      if (!batch) return;
      stageBatch({ id: batch.id, documentId: document.id });
    },
    [batch, stageBatch],
  );

  const handleUploadFailed = useCallback((message: string) => {
    setPendingFile(null);
    setProblem(message);
  }, []);

  const close = useCallback(() => {
    onOpenChange(false);
  }, [onOpenChange]);

  const requestClose = useCallback(() => {
    if (needsDiscardConfirmation(batch)) {
      setDiscardPrompt("close");
      return;
    }
    close();
  }, [batch, close]);

  const requestDiscard = useCallback(() => {
    if (!batch) return;
    if (needsDiscardConfirmation(batch)) {
      setDiscardPrompt("stay");
      return;
    }
    discardBatch(batch, { onSuccess: () => resetAll() });
  }, [batch, discardBatch, resetAll]);

  const confirmDiscard = useCallback(() => {
    if (!batch) return;
    const closeAfter = discardPrompt === "close";
    discardBatch(batch, {
      onSuccess: (discarded) => {
        setDiscardPrompt(null);
        if (closeAfter) {
          close();
        } else {
          setBatch(discarded);
        }
      },
    });
  }, [batch, close, discardBatch, discardPrompt]);

  const step = importStep(batch);
  const uploading = Boolean(batch && pendingFile && batch.status === "Pending");
  const busy = creating || staging || uploading;
  const failed = batch?.status === "Failed";
  const warnings = importWarnings(batch);

  return (
    <>
      <Dialog
        open
        onOpenChange={(next) => {
          if (next) onOpenChange(true);
          else requestClose();
        }}
      >
        <DialogContent className="max-h-[85vh] overflow-y-auto sm:max-w-3xl">
          <DialogHeader>
            <DialogTitle>Import Fuel Card Statement</DialogTitle>
            <DialogDescription>
              Upload a provider statement, read what it would record, then confirm. Nothing becomes
              a purchase until you do.
            </DialogDescription>
            <StepIndicator current={step} />
          </DialogHeader>

          {step === "setup" ? (
            <div className="flex flex-col gap-4">
              <FormProvider {...form}>
                <Form onSubmit={(event) => event.preventDefault()}>
                  <FormGroup cols={2}>
                    <FormControl>
                      <SelectField
                        control={form.control}
                        name="provider"
                        label="Provider"
                        options={fuelCardProviderChoices}
                        rules={{ required: true }}
                        placeholder="Select a provider"
                        isReadOnly={busy}
                        description="Picks the column layout the statement is read with."
                      />
                    </FormControl>
                    <FormControl>
                      <FuelCardAutocompleteField<FuelPurchaseImportSetupValues>
                        control={form.control}
                        name="defaultFuelCardId"
                        label="Default card"
                        placeholder="Select a card"
                        clearable
                        disabled={busy}
                        description="Used for rows the statement cannot match to a card of their own."
                      />
                    </FormControl>
                    <FormControl>
                      <SelectField
                        control={form.control}
                        name="defaultFuelType"
                        label="Default fuel type"
                        options={iftaFuelTypeChoices}
                        placeholder="Select a fuel type"
                        isClearable
                        isReadOnly={busy}
                        description="Applied when the statement has no product column."
                      />
                    </FormControl>
                    <FormControl>
                      <InputField
                        control={form.control}
                        name="defaultCurrency"
                        label="Currency"
                        placeholder="USD"
                        maxLength={3}
                        rules={{ required: true }}
                        readOnly={busy}
                        description="Three-letter code for amounts the statement does not label."
                      />
                    </FormControl>
                  </FormGroup>
                </Form>
              </FormProvider>

              {uploading && batch && pendingFile ? (
                <StatementUploader
                  batch={batch}
                  file={pendingFile}
                  onUploaded={handleUploaded}
                  onFailed={handleUploadFailed}
                />
              ) : (
                <DocumentUploadZone
                  accept=".csv,.xlsx"
                  disabled={busy}
                  onFilesSelected={(files) => {
                    const file = files[0];
                    if (file) void handleFile(file);
                  }}
                  onFilesRejected={() => setProblem("That file is too large to import.")}
                  className="min-h-32"
                />
              )}

              {problem ? (
                <Alert variant="destructive">
                  <CircleAlertIcon className="size-4" />
                  <AlertTitle>The statement could not be staged</AlertTitle>
                  <AlertDescription>{problem}</AlertDescription>
                </Alert>
              ) : null}

              <div className="bg-muted/30 flex items-center justify-between rounded-lg border px-3 py-2">
                <p className="text-muted-foreground text-xs">
                  Not sure of the layout? Start from the {provider} template.
                </p>
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  isLoading={downloadingTemplate}
                  onClick={() => downloadTemplate()}
                  className="gap-1.5"
                >
                  <DownloadIcon className="size-3.5" />
                  Download template
                </Button>
              </div>
            </div>
          ) : null}

          {step === "review" && batch ? (
            <div className="flex flex-col gap-3">
              <div className="bg-muted/30 rounded-lg border p-3">
                <div className="mb-1.5 flex flex-wrap items-center gap-2">
                  <FuelPurchaseImportStatusBadge status={batch.status} />
                  {batch.fileName ? (
                    <span className="font-mono text-xs">{batch.fileName}</span>
                  ) : null}
                  <span className="text-muted-foreground text-xs">
                    {batch.rowCount} {batch.rowCount === 1 ? "row" : "rows"}
                  </span>
                </div>
                <p className="text-sm">{importHeadline(batch)}</p>
              </div>

              {failed ? (
                <Alert variant="destructive">
                  <CircleAlertIcon className="size-4" />
                  <AlertTitle>The statement could not be read</AlertTitle>
                  <AlertDescription>
                    {batch.error ?? "No reason was reported."} Every row that failed is listed
                    below; fix the file and upload it again.
                  </AlertDescription>
                </Alert>
              ) : (
                <SummaryChips batch={batch} />
              )}

              {warnings.length > 0 ? (
                <Alert>
                  <CircleAlertIcon className="size-4" />
                  <AlertDescription>
                    <ul className="list-inside list-disc space-y-0.5">
                      {warnings.map((warning) => (
                        <li key={warning}>{warning}</li>
                      ))}
                    </ul>
                  </AlertDescription>
                </Alert>
              ) : null}

              <ImportReviewTable
                batch={batch}
                filter={failed ? "errors" : filter}
                onFilterChange={setFilter}
                showFilters={!failed}
              />
            </div>
          ) : null}

          {step === "done" && batch ? (
            <div className="flex flex-col items-center gap-2 py-6 text-center">
              {batch.status === "Committed" ? (
                <CircleCheckIcon className="text-success size-8" />
              ) : (
                <Trash2Icon className="text-muted-foreground size-8" />
              )}
              <p className="text-sm font-medium">{importHeadline(batch)}</p>
              {batch.status === "Committed" ? (
                <p className="text-muted-foreground text-xs">
                  They are in the purchases table now and will be counted the next time the
                  quarter&apos;s return is computed.
                </p>
              ) : null}
            </div>
          ) : null}

          <DialogFooter>
            {step === "setup" ? (
              <Button type="button" variant="outline" onClick={requestClose} disabled={busy}>
                Close
              </Button>
            ) : null}
            {step === "review" ? (
              <>
                <Button
                  type="button"
                  variant="outline"
                  onClick={requestDiscard}
                  disabled={discarding || committing}
                >
                  Discard
                </Button>
                {failed ? (
                  <Button type="button" onClick={resetAll}>
                    Try another file
                  </Button>
                ) : (
                  <Button
                    type="button"
                    isLoading={committing}
                    loadingText="Importing..."
                    disabled={!canCommitImport(batch)}
                    onClick={() => batch && commitBatch(batch)}
                  >
                    {commitLabel(batch)}
                  </Button>
                )}
              </>
            ) : null}
            {step === "done" ? (
              <Button type="button" onClick={close}>
                Done
              </Button>
            ) : null}
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <AlertDialog
        open={discardPrompt !== null}
        onOpenChange={(next) => {
          if (!next) setDiscardPrompt(null);
        }}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogMedia className="bg-destructive/10 text-destructive">
              <Trash2Icon />
            </AlertDialogMedia>
            <AlertDialogTitle>{batch ? discardImportNotice(batch).title : ""}</AlertDialogTitle>
            <AlertDialogDescription>
              {batch ? discardImportNotice(batch).description : ""}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={discarding}>Keep reviewing</AlertDialogCancel>
            <AlertDialogAction variant="destructive" onClick={confirmDiscard} disabled={discarding}>
              {discarding ? "Discarding..." : "Discard import"}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  );
}
