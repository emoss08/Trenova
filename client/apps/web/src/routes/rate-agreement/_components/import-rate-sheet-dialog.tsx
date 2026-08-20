import { RateAgreementAutocompleteField } from "@/components/autocomplete-fields";
import { AutoCompleteDateField } from "@/components/fields/date-field/date-field";
import { downloadCsv } from "@/lib/data-table-export";
import { apiService } from "@/services/api";
import { zodResolver } from "@hookform/resolvers/zod";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
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
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@trenova/shared/components/ui/table";
import { ApiRequestError } from "@trenova/shared/lib/api";
import { getTodayDate } from "@trenova/shared/lib/date";
import {
  canCommit,
  changeKindLabel,
  changeTone,
  commitWarnings,
  describeFieldChange,
  importHeadline,
  meaningfulChanges,
  type ChangeTone,
} from "@trenova/shared/lib/rate-import";
import { cn } from "@trenova/shared/lib/utils";
import { apiProblem } from "@trenova/shared/types/errors";
import {
  rateImportUploadSchema,
  type RateImportBatch,
  type RateImportUploadValues,
} from "@trenova/shared/types/rate";
import { CircleAlertIcon, DownloadIcon, FileUpIcon } from "lucide-react";
import { useCallback, useEffect, useRef, useState } from "react";
import { FormProvider, useForm, type Resolver } from "react-hook-form";
import { toast } from "sonner";

const TONE_CLASS: Record<ChangeTone, string> = {
  danger: "text-destructive",
  warning: "text-warning",
  neutral: "text-foreground",
  quiet: "text-muted-foreground",
};

type ImportRateSheetDialogProps = {
  readonly open: boolean;
  readonly onOpenChange: (open: boolean) => void;
};

/**
 * Upload a rate sheet, read what it would do, then apply it.
 *
 * The dry run between the upload and the commit is the whole point. An import
 * that went straight from a file to a live contract is what bulk ingest is most
 * often blamed for: a tariff nobody read replacing one somebody negotiated.
 */
export function ImportRateSheetDialog({ open, onOpenChange }: ImportRateSheetDialogProps) {
  const queryClient = useQueryClient();
  const fileInput = useRef<HTMLInputElement>(null);

  const [batch, setBatch] = useState<RateImportBatch | undefined>();
  const [fileProblems, setFileProblems] = useState<string[]>([]);
  const [isDragging, setIsDragging] = useState(false);

  const form = useForm<RateImportUploadValues>({
    resolver: zodResolver(rateImportUploadSchema) as Resolver<RateImportUploadValues>,
    defaultValues: { rateAgreementId: "", effectiveFrom: getTodayDate() },
  });

  useEffect(() => {
    if (open) {
      form.reset({ rateAgreementId: "", effectiveFrom: getTodayDate() });
      setBatch(undefined);
      setFileProblems([]);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open, form.reset]);

  const { mutate: upload, isPending: isUploading } = useMutation({
    mutationFn: (payload: RateImportUploadValues & { file: File }) =>
      apiService.rateImportService.upload(payload),
    onSuccess: async (created) => {
      setBatch(created);
      setFileProblems([]);
      await queryClient.invalidateQueries({
        queryKey: ["rate-imports", created.rateAgreementId],
      });
    },
    onError: (error) => setFileProblems(fileProblemsOf(error)),
  });

  const { mutate: commit, isPending: isCommitting } = useMutation({
    mutationFn: () => apiService.rateImportService.commit(batch?.id as string),
    onSuccess: async (applied) => {
      setBatch(applied);
      toast.success("Rates applied to the agreement");
      await queryClient.invalidateQueries({ queryKey: ["rate-agreement-list"] });
      await queryClient.invalidateQueries({ queryKey: ["rate-imports", applied.rateAgreementId] });
    },
    onError: () => toast.error("The rates could not be applied"),
  });

  const { mutate: discard } = useMutation({
    mutationFn: () => apiService.rateImportService.discard(batch?.id as string),
    onSuccess: (discarded) => {
      setBatch(discarded);
      void queryClient.invalidateQueries({ queryKey: ["rate-imports", discarded.rateAgreementId] });
    },
    onError: () => toast.error("The import could not be discarded"),
  });

  const { data: failedRows } = useQuery({
    queryKey: ["rate-import-failed-rows", batch?.id],
    queryFn: () => apiService.rateImportService.listFailedRows(batch?.id as string),
    enabled: Boolean(batch?.id) && (batch?.errorCount ?? 0) > 0,
  });

  const { mutate: downloadTemplate, isPending: isDownloading } = useMutation({
    mutationFn: () => apiService.rateImportService.template(),
    onSuccess: ({ fileName, content }) => downloadCsv(content, fileName),
    onError: () => toast.error("The template could not be downloaded"),
  });

  const stage = useCallback(
    async (file: File) => {
      const valid = await form.trigger();
      if (!valid) return;

      upload({ ...form.getValues(), file });
    },
    [form, upload],
  );

  const onFileChosen = useCallback(
    (event: React.ChangeEvent<HTMLInputElement>) => {
      const file = event.target.files?.[0];
      if (file) void stage(file);

      // Clearing lets the same file be re-uploaded after it has been fixed,
      // which is the ordinary next step after reading the errors.
      event.target.value = "";
    },
    [stage],
  );

  const onDrop = useCallback(
    (event: React.DragEvent) => {
      event.preventDefault();
      setIsDragging(false);

      const file = event.dataTransfer.files?.[0];
      if (file) void stage(file);
    },
    [stage],
  );

  const changes = meaningfulChanges(batch?.changes);
  const warnings = commitWarnings(batch);
  const reviewing = Boolean(batch);

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-h-[85vh] overflow-y-auto sm:max-w-2xl">
        <DialogHeader>
          <DialogTitle>Import Rate Sheet</DialogTitle>
          <DialogDescription>
            Upload a CSV or XLSX rate sheet into an agreement. Nothing is applied until you have
            read exactly what it would change.
          </DialogDescription>
        </DialogHeader>

        {!reviewing && (
          <>
            <FormProvider {...form}>
              <Form onSubmit={(event) => event.preventDefault()}>
                <FormGroup cols={2}>
                  <FormControl>
                    <RateAgreementAutocompleteField
                      control={form.control}
                      name="rateAgreementId"
                      label="Agreement"
                      placeholder="Select agreement"
                      rules={{ required: true }}
                      description="The contract this sheet's lanes are imported into."
                    />
                  </FormControl>
                  <FormControl>
                    <AutoCompleteDateField
                      control={form.control}
                      name="effectiveFrom"
                      label="Rates Take Effect"
                      rules={{ required: true }}
                      description="The day the imported rates start pricing — the negotiated date, not the upload date."
                    />
                  </FormControl>
                </FormGroup>
              </Form>
            </FormProvider>

            <button
              type="button"
              onClick={() => fileInput.current?.click()}
              onDragOver={(event) => {
                event.preventDefault();
                setIsDragging(true);
              }}
              onDragLeave={() => setIsDragging(false)}
              onDrop={onDrop}
              disabled={isUploading}
              className={cn(
                "flex flex-col items-center justify-center rounded-lg border border-dashed px-4 py-10 text-center transition-colors",
                isDragging ? "border-primary bg-primary/5" : "hover:bg-muted/50",
              )}
            >
              <FileUpIcon className="mb-2 size-6 text-muted-foreground" />
              <p className="text-sm font-medium">
                {isUploading ? "Reading the sheet…" : "Drop a rate sheet here, or click to browse"}
              </p>
              <p className="mt-1 text-xs text-muted-foreground">
                CSV or XLSX. Columns are matched by name.
              </p>
            </button>
            <input
              ref={fileInput}
              type="file"
              accept=".csv,.xlsx,.xlsm"
              className="hidden"
              onChange={onFileChosen}
            />

            {fileProblems.length > 0 && (
              <Alert variant="destructive">
                <CircleAlertIcon className="size-4" />
                <AlertDescription>
                  <p className="font-medium">This sheet could not be imported</p>
                  <ul className="mt-1 list-inside list-disc space-y-0.5">
                    {fileProblems.map((problem) => (
                      <li key={problem}>{problem}</li>
                    ))}
                  </ul>
                  <p className="mt-1.5 text-xs">
                    The template below has every column the importer recognises, with two example
                    rows.
                  </p>
                </AlertDescription>
              </Alert>
            )}

            <div className="flex items-center justify-between rounded-lg border bg-muted/30 px-3 py-2">
              <p className="text-xs text-muted-foreground">
                Not sure how to lay out the sheet? Start from the template.
              </p>
              <Button
                type="button"
                variant="outline"
                size="sm"
                isLoading={isDownloading}
                onClick={() => downloadTemplate()}
                className="gap-1.5"
              >
                <DownloadIcon className="size-3.5" />
                Download Template
              </Button>
            </div>
          </>
        )}

        {batch && (
          <div className="flex flex-col gap-3">
            <div className="rounded-lg border bg-muted/30 p-3">
              <div className="mb-1.5 flex items-center gap-2">
                <Badge variant={batch.status === "Failed" ? "warning" : "secondary"}>
                  {batch.status}
                </Badge>
                <span className="font-mono text-xs">{batch.fileName}</span>
                <span className="text-xs text-muted-foreground">{batch.rowCount} rows</span>
              </div>
              <p className="text-sm">{importHeadline(batch)}</p>
            </div>

            {warnings.length > 0 && (
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
            )}

            {failedRows && failedRows.length > 0 && (
              <div className="overflow-hidden rounded-lg border">
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead className="w-16 text-xs">Row</TableHead>
                      <TableHead className="text-xs">Why it would not read</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {failedRows.map((row) => (
                      <TableRow key={row.rowNumber}>
                        <TableCell className="font-mono text-xs tabular-nums">
                          {row.rowNumber}
                        </TableCell>
                        <TableCell className="text-xs text-muted-foreground">{row.error}</TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </div>
            )}

            {changes.length > 0 && (
              <div className="overflow-hidden rounded-lg border">
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead className="text-xs">Lane</TableHead>
                      <TableHead className="text-xs">What Happens</TableHead>
                      <TableHead className="text-xs">Changes</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {changes.map((change) => (
                      <TableRow key={`${change.kind}-${change.laneKey}`}>
                        <TableCell>
                          <span className="text-xs font-medium">
                            {change.label || change.laneKey}
                          </span>
                          <p className="font-mono text-2xs text-muted-foreground">
                            {change.laneKey}
                          </p>
                        </TableCell>
                        <TableCell className={cn("text-xs", TONE_CLASS[changeTone(change.kind)])}>
                          {changeKindLabel(change.kind)}
                        </TableCell>
                        <TableCell className="text-xs text-muted-foreground">
                          {(change.fields ?? []).map((field) => (
                            <p key={field.field}>{describeFieldChange(field)}</p>
                          ))}
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </div>
            )}
          </div>
        )}

        <DialogFooter>
          {canCommit(batch) ? (
            <>
              <Button
                type="button"
                variant="outline"
                onClick={() => {
                  discard();
                  setBatch(undefined);
                }}
              >
                Discard
              </Button>
              <Button
                type="button"
                isLoading={isCommitting}
                loadingText="Applying..."
                onClick={() => commit()}
              >
                Apply These Rates
              </Button>
            </>
          ) : (
            <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
              {batch?.status === "Committed" ? "Done" : "Close"}
            </Button>
          )}
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

/**
 * What to tell somebody whose sheet was refused.
 *
 * The server reports every template problem at once — a missing rate column
 * and a missing destination arrive together — and each deserves its own line
 * rather than a joined sentence.
 */
function fileProblemsOf(error: unknown): string[] {
  if (error instanceof ApiRequestError) {
    const normalized = error.normalize();

    if (apiProblem.isValidationError(normalized) && normalized.fieldErrors.length > 0) {
      return normalized.fieldErrors.map((fieldError) => fieldError.message);
    }

    return [normalized.message];
  }

  return ["That file could not be read."];
}
