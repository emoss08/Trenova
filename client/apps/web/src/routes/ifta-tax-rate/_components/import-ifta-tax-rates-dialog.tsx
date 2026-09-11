import { useT } from "@trenova/shared/i18n/use-t";
import { DocumentUploadZone } from "@/components/documents/document-upload-zone";
import { useIftaJurisdictionOptions } from "@/components/fields/ifta-jurisdiction-select-field";
import { NumberField } from "@/components/fields/number-field";
import { SelectField } from "@/components/fields/select-field";
import { handleMutationError } from "@/hooks/use-api-mutation";
import { iftaQuarterChoices } from "@/lib/choices";
import { downloadCsv } from "@/lib/data-table-export";
import { IFTA_TAX_RATE_LIST_KEY, upsertIftaTaxRates } from "@/lib/graphql/ifta-tax-rate";
import {
  IFTA_TAX_RATE_TEMPLATE_CSV,
  iftaTaxRateTemplateFileName,
  parseIftaTaxRateCsv,
  type IftaTaxRateImportResult,
} from "@/lib/ifta-tax-rate-import";
import { zodResolver } from "@hookform/resolvers/zod";
import { useMutation, useQueryClient } from "@tanstack/react-query";
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
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@trenova/shared/components/ui/table";
import { cn, pluralize } from "@trenova/shared/lib/utils";
import type { IftaTaxRateInput } from "@trenova/graphql/generated/graphql";
import {
  IFTA_MAX_YEAR,
  IFTA_MIN_YEAR,
  iftaPeriodFormSchema,
  type IftaPeriodFormValues,
} from "@trenova/shared/types/ifta-tax-rate";
import type { IftaQuarter } from "@trenova/shared/types/fuel-ifta-enums";
import { CircleAlertIcon, DownloadIcon } from "lucide-react";
import { useCallback, useState } from "react";
import { FormProvider, useForm, useWatch, type Resolver } from "react-hook-form";
import { toast } from "sonner";
import { mostRecentCompletedQuarter, type IftaPeriod } from "./ifta-tax-rate-panel";

type ImportIftaTaxRatesDialogProps = {
  readonly open: boolean;
  readonly onOpenChange: (open: boolean) => void;
};

function periodDefaults(period: IftaPeriod): IftaPeriodFormValues {
  return { year: period.year, quarter: String(period.quarter) as IftaQuarter };
}

type Preview = {
  fileName: string;
  year: number;
  quarter: number;
  parsed: IftaTaxRateImportResult;
};

export function ImportIftaTaxRatesDialog({ open, onOpenChange }: ImportIftaTaxRatesDialogProps) {
  if (!open) return null;
  return <ImportRatesSession onOpenChange={onOpenChange} />;
}

function ImportRatesSession({ onOpenChange }: Pick<ImportIftaTaxRatesDialogProps, "onOpenChange">) {
  const t = useT();

  const queryClient = useQueryClient();
  const { jurisdictions, isLoading: jurisdictionsLoading } = useIftaJurisdictionOptions();
  const [preview, setPreview] = useState<Preview | null>(null);
  const [readError, setReadError] = useState<string | null>(null);

  const form = useForm<IftaPeriodFormValues>({
    resolver: zodResolver(iftaPeriodFormSchema) as Resolver<IftaPeriodFormValues>,
    defaultValues: periodDefaults(mostRecentCompletedQuarter()),
  });
  const year = useWatch({ control: form.control, name: "year" });
  const quarter = useWatch({ control: form.control, name: "quarter" });

  const current =
    preview && preview.year === year && preview.quarter === Number(quarter) ? preview : null;
  const result = current?.parsed ?? null;
  const fileName = current?.fileName ?? null;

  const { mutate: importRates, isPending } = useMutation({
    mutationFn: (rates: IftaTaxRateInput[]) => upsertIftaTaxRates(rates),
    onSuccess: async (saved) => {
      toast.success(t("Rates imported"), {
        description: `${saved.length} ${pluralize("rate", saved.length)} published for Q${quarter} ${year}.`,
      });
      await queryClient.invalidateQueries({ queryKey: [IFTA_TAX_RATE_LIST_KEY] });
      onOpenChange(false);
    },
    onError: (error) => handleMutationError({ error, resourceName: "IFTA Tax Rates" }),
  });

  const readFile = useCallback(
    async (file: File) => {
      const valid = await form.trigger();
      if (!valid) return;
      setReadError(null);
      try {
        const text = await file.text();
        const values = form.getValues();
        setPreview({
          fileName: file.name,
          year: values.year,
          quarter: Number(values.quarter),
          parsed: parseIftaTaxRateCsv(text, {
            jurisdictions,
            year: values.year,
            quarter: Number(values.quarter),
          }),
        });
      } catch (error) {
        setReadError(error instanceof Error ? error.message : "That file could not be read.");
      }
    },
    [form, jurisdictions],
  );

  const errorCount = result?.rows.filter((row) => row.error !== null).length ?? 0;
  const validCount = result?.valid.length ?? 0;
  const reviewing = result !== null && result.fileErrors.length === 0;

  return (
    <Dialog open onOpenChange={onOpenChange}>
      <DialogContent className="max-h-[85vh] overflow-y-auto sm:max-w-3xl">
        <DialogHeader>
          <DialogTitle>{t("Import IFTA Tax Rates")}</DialogTitle>
          <DialogDescription>
            {t("Read a CSV of the quarter's matrix in the browser, check every row, then publish the good ones. Rows with problems are shown but never sent.")}
          </DialogDescription>
        </DialogHeader>

        <FormProvider {...form}>
          <Form onSubmit={(event) => event.preventDefault()}>
            <FormGroup cols={2}>
              <FormControl>
                <NumberField
                  control={form.control}
                  name="year"
                  label={t("Year")}
                  placeholder="2026"
                  min={IFTA_MIN_YEAR}
                  max={IFTA_MAX_YEAR}
                  rules={{ required: true }}
                  description={t("Every row in the file is published for this year.")}
                />
              </FormControl>
              <FormControl>
                <SelectField
                  control={form.control}
                  name="quarter"
                  label={t("Quarter")}
                  options={iftaQuarterChoices}
                  rules={{ required: true }}
                  placeholder={t("Select a quarter")}
                  description={t("And this quarter. Changing either clears the preview.")}
                />
              </FormControl>
            </FormGroup>
          </Form>
        </FormProvider>

        {!reviewing ? (
          <DocumentUploadZone
            accept=".csv,.txt"
            disabled={jurisdictionsLoading}
            onFilesSelected={(files) => {
              const file = files[0];
              if (file) void readFile(file);
            }}
            onFilesRejected={() => setReadError("That file is too large to read here.")}
            className="min-h-32"
          />
        ) : null}

        {readError ? (
          <Alert variant="destructive">
            <CircleAlertIcon className="size-4" />
            <AlertDescription>{readError}</AlertDescription>
          </Alert>
        ) : null}

        {result && result.fileErrors.length > 0 ? (
          <Alert variant="destructive">
            <CircleAlertIcon className="size-4" />
            <AlertTitle>{t("{0} could not be read", fileName ?? "This file")}</AlertTitle>
            <AlertDescription>
              <ul className="mt-1 list-inside list-disc space-y-0.5">
                {result.fileErrors.map((problem) => (
                  <li key={problem}>{problem}</li>
                ))}
              </ul>
              <p className="mt-1.5 text-xs">
                {t("The columns are jurisdiction code, fuel type, rate per gallon and surcharge per gallon, matched by name. Start from the template if in doubt.")}
              </p>
            </AlertDescription>
          </Alert>
        ) : null}

        {reviewing && result ? (
          <div className="flex flex-col gap-3">
            <div className="bg-muted/30 rounded-lg border p-3">
              <div className="mb-1.5 flex flex-wrap items-center gap-2">
                <Badge variant={errorCount > 0 ? "warning" : "secondary"}>
                  {errorCount > 0 ? "Needs attention" : "Ready"}
                </Badge>
                {fileName ? <span className="font-mono text-xs">{fileName}</span> : null}
                <span className="text-muted-foreground text-xs">
                  {result.rows.length} {pluralize("row", result.rows.length)}
                </span>
              </div>
              <p className="text-sm">
                {validCount > 0
                  ? `Publishing would set ${validCount} ${pluralize("rate", validCount)} for Q${quarter} ${year}, replacing any already published for the same jurisdiction and fuel.`
                  : "No row in this file can be published as it stands."}
                {errorCount > 0
                  ? ` ${errorCount} ${pluralize("row", errorCount)} will be left out.`
                  : ""}
              </p>
            </div>

            <div className="overflow-hidden rounded-lg border">
              <div className="max-h-80 overflow-auto">
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead className="w-12 text-xs">{t("Line")}</TableHead>
                      <TableHead className="text-xs">{t("Jurisdiction")}</TableHead>
                      <TableHead className="text-xs">{t("Fuel")}</TableHead>
                      <TableHead className="text-right text-xs">{t("Rate")}</TableHead>
                      <TableHead className="text-right text-xs">{t("Surcharge")}</TableHead>
                      <TableHead className="text-xs">{t("Problem")}</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {result.rows.map((row) => (
                      <TableRow key={row.line} data-invalid={row.error ? "true" : undefined}>
                        <TableCell className="font-mono text-xs tabular-nums">{row.line}</TableCell>
                        <TableCell className="text-xs">{row.jurisdictionCode || "—"}</TableCell>
                        <TableCell className="text-xs">{row.fuelType || "—"}</TableCell>
                        <TableCell className="text-right text-xs tabular-nums">
                          {row.ratePerGallon || "—"}
                        </TableCell>
                        <TableCell className="text-right text-xs tabular-nums">
                          {row.surchargeRatePerGallon ?? "—"}
                        </TableCell>
                        <TableCell
                          className={cn(
                            "text-xs",
                            row.error ? "text-destructive" : "text-muted-foreground",
                          )}
                        >
                          {row.error ?? "OK"}
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </div>
            </div>
          </div>
        ) : null}

        {!reviewing ? (
          <div className="bg-muted/30 flex items-center justify-between rounded-lg border px-3 py-2">
            <p className="text-muted-foreground text-xs">
              {t("Not sure how to lay out the file? Start from the template.")}
            </p>
            <Button
              type="button"
              variant="outline"
              size="sm"
              className="gap-1.5"
              onClick={() =>
                downloadCsv(
                  IFTA_TAX_RATE_TEMPLATE_CSV,
                  iftaTaxRateTemplateFileName(year, Number(quarter)),
                )
              }
            >
              <DownloadIcon className="size-3.5" />
              {t("Download template")}
            </Button>
          </div>
        ) : null}

        <DialogFooter>
          {reviewing ? (
            <>
              <Button
                type="button"
                variant="outline"
                onClick={() => setPreview(null)}
                disabled={isPending}
              >
                {t("Choose another file")}
              </Button>
              <Button
                type="button"
                isLoading={isPending}
                loadingText={t("Publishing...")}
                disabled={validCount === 0}
                onClick={() => result && importRates(result.valid)}
              >
                {t("Publish {0} {1}", validCount, pluralize("rate", validCount))}
              </Button>
            </>
          ) : (
            <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
              {t("Close")}
            </Button>
          )}
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
