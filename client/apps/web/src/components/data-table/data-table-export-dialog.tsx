import { ChoiceButton } from "@/components/choice-button";
import { EmailChipsField } from "@/components/fields/email-chips-field";
import { InputField } from "@/components/fields/input-field";
import { SegmentedField } from "@/components/fields/segmented-field";
import { useApiMutation } from "@/hooks/use-api-mutation";
import {
  buildCsv,
  buildExportColumns,
  buildTableExportView,
  downloadCsv,
  EXPORT_MAX_ROWS,
  exportFilename,
  fetchAllRows,
} from "@/lib/data-table-export";
import { buildPrintDocument, PRINT_MAX_ROWS, printDocument } from "@/lib/data-table-print";
import { formatToUserTimezone } from "@trenova/shared/lib/date";
import {
  exportTableView,
  scheduleTableView,
  type TableExportOutcome,
} from "@/lib/graphql/table-export";
import { zodResolver } from "@hookform/resolvers/zod";
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
import { Label } from "@trenova/shared/components/ui/label";
import { translate } from "@trenova/shared/i18n/runtime";
import { useT } from "@trenova/shared/i18n/use-t";
import type {
  DataTableGraphQLSource,
  DataTableQueryOptions,
  Table,
} from "@trenova/shared/types/data-table";
import { useEffect, useRef, useState } from "react";
import { Controller, FormProvider, useForm, useWatch } from "react-hook-form";
import { toast } from "sonner";
import { z } from "zod";

/** Scheduled exports run at seven in the morning, in the scheduler's time zone. */
const SCHEDULE_CRON = {
  daily: "0 7 * * *",
  weekly: "0 7 * * 1",
  monthly: "0 7 1 * *",
} as const;

const MAX_RECIPIENTS = 50;

const exportFormSchema = z
  .object({
    destination: z.enum(["download", "print", "report", "schedule"]),
    scope: z.enum(["all", "page"]),
    columnsMode: z.enum(["visible", "all"]),
    format: z.enum(["xlsx", "csv", "pdf"]),
    name: z
      .string()
      .trim()
      .max(200, { error: () => translate("Keep the name under 200 characters") }),
    frequency: z.enum(["daily", "weekly", "monthly"]),
    recipients: z
      .array(z.string())
      .max(MAX_RECIPIENTS, { error: () => translate("Send to at most 50 people") }),
  })
  .superRefine((values, ctx) => {
    if ((values.destination === "report" || values.destination === "schedule") && values.name === "") {
      ctx.addIssue({
        code: "custom",
        path: ["name"],
        message: translate("Give the report a name"),
      });
    }
  });

type ExportFormValues = z.infer<typeof exportFormSchema>;

function exportDefaults(resource: string): ExportFormValues {
  return {
    destination: "download",
    scope: "all",
    columnsMode: "visible",
    format: "xlsx",
    name: translate("{0} export", resource),
    frequency: "weekly",
    recipients: [],
  };
}

type DataTableExportDialogProps<TData extends Record<string, any>> = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  /** The table's name, used for the file and the report. */
  resource: string;
  /** The table's permission resource. Without one the table can only download. */
  permissionResource?: string;
  table: Table<TData>;
  graphql: DataTableGraphQLSource<TData>;
  queryOptions: Omit<DataTableQueryOptions, "cursor">;
  currentPageRows: TData[];
  totalCount: number | null;
};

/**
 * Exports what the table is showing. A download is a CSV built in the browser, up to
 * its row cap. A report is built on the server in Excel, CSV or PDF with every
 * matching row, once or on a schedule, and is kept in the person's reports.
 */
export default function DataTableExportDialog<TData extends Record<string, any>>({
  open,
  onOpenChange,
  resource,
  permissionResource,
  table,
  graphql,
  queryOptions,
  currentPageRows,
  totalCount,
}: DataTableExportDialogProps<TData>) {
  const t = useT();
  const [progress, setProgress] = useState<string | null>(null);
  const cancelledRef = useRef(false);

  const form = useForm<ExportFormValues>({
    resolver: zodResolver(exportFormSchema),
    defaultValues: exportDefaults(resource),
  });
  const { control, handleSubmit, reset, setError } = form;
  const destination = useWatch({ control, name: "destination" });

  useEffect(() => {
    if (open) reset(exportDefaults(resource));
  }, [open, reset, resource]);

  const localCap = destination === "print" ? PRINT_MAX_ROWS : EXPORT_MAX_ROWS;
  const overCap = totalCount != null && totalCount > localCap;

  const handleOpenChange = (nextOpen: boolean) => {
    if (!nextOpen) cancelledRef.current = true;
    onOpenChange(nextOpen);
  };

  const { mutateAsync: sendToServer } = useApiMutation<
    TableExportOutcome,
    ExportFormValues,
    unknown,
    ExportFormValues
  >({
    mutationFn: (values) => {
      const view = buildTableExportView(
        table.getAllLeafColumns(),
        values.columnsMode === "visible",
        permissionResource ?? "",
        queryOptions,
      );
      if (values.destination === "schedule") {
        return scheduleTableView({
          name: values.name,
          view,
          cronExpression: SCHEDULE_CRON[values.frequency],
          timezone: Intl.DateTimeFormat().resolvedOptions().timeZone,
          formats: [values.format],
          emailRecipients: values.recipients,
          emailAttach: values.recipients.length > 0,
        });
      }
      return exportTableView({ name: values.name, view, format: values.format });
    },
    resourceName: "Export",
    form,
    onSuccess: (outcome, values) => {
      const skipped =
        outcome.skippedColumns.length > 0
          ? t("These columns could not be included: {0}.", outcome.skippedColumns.join(", "))
          : null;
      if (values.destination === "schedule") {
        toast.success(t("Export scheduled"), {
          description: skipped ?? t("Change or stop it from Reports."),
        });
      } else {
        toast.success(t("Your export is being built"), {
          description: skipped ?? t("You will be told when the file is ready. It is kept in Reports."),
        });
      }
      onOpenChange(false);
    },
  });

  const collectRows = async (values: ExportFormValues, maxRows: number) => {
    if (values.scope === "page") return currentPageRows;
    return fetchAllRows<TData>({
      graphql,
      options: queryOptions,
      maxRows,
      onProgress: ({ fetched, total }) =>
        setProgress(
          total != null
            ? t("{0} of {1}", fetched.toLocaleString(), Math.min(total, maxRows).toLocaleString())
            : fetched.toLocaleString(),
        ),
      isCancelled: () => cancelledRef.current,
    });
  };

  const buildLocally = async (values: ExportFormValues) => {
    const exportColumns = buildExportColumns(
      table.getAllLeafColumns(),
      values.columnsMode === "visible",
    );
    if (exportColumns.length === 0) {
      setError("root", { message: t("No exportable columns are available.") });
      return;
    }

    const printing = values.destination === "print";
    cancelledRef.current = false;
    setProgress(null);
    try {
      const rows = await collectRows(values, printing ? PRINT_MAX_ROWS : EXPORT_MAX_ROWS);
      if (cancelledRef.current) return;

      if (printing) {
        onOpenChange(false);
        await printDocument(
          buildPrintDocument({
            title: resource,
            subtitle: t(
              "{0, plural, one {# row} other {# rows}} · {1}",
              rows.length,
              formatToUserTimezone(Math.floor(Date.now() / 1000), { showSeconds: false }),
            ),
            columns: exportColumns,
            rows,
          }),
        );
        return;
      }

      downloadCsv(buildCsv(rows, exportColumns), exportFilename(resource));
      toast.success(t("Export complete"), {
        description: t(
          "{0, plural, one {Exported # row to CSV.} other {Exported # rows to CSV.}}",
          rows.length,
        ),
      });
      onOpenChange(false);
    } catch (error) {
      setError("root", {
        message:
          printing && error instanceof Error
            ? error.message
            : t("The rows could not be fetched. Try again."),
      });
    } finally {
      setProgress(null);
    }
  };

  const onSubmit = async (values: ExportFormValues) => {
    if (values.destination === "download" || values.destination === "print") {
      await buildLocally(values);
    } else {
      await sendToServer(values);
    }
  };

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent size="md">
        <DialogHeader>
          <DialogTitle>{t("Export")}</DialogTitle>
          <DialogDescription>
            {t("Exports respect the current filters, sorting, and column layout.")}
          </DialogDescription>
        </DialogHeader>
        <FormProvider {...form}>
          <Form onSubmit={handleSubmit(onSubmit)}>
            <FormGroup cols={1} className="pb-4">
              <FormControl cols="full">
                <SegmentedField
                  name="destination"
                  control={control}
                  label={t("Destination")}
                  options={[
                    { value: "download", label: t("CSV") },
                    { value: "print", label: t("Print") },
                    ...(permissionResource
                      ? [
                          { value: "report" as const, label: t("Report") },
                          { value: "schedule" as const, label: t("Schedule") },
                        ]
                      : []),
                  ]}
                  description={
                    destination === "download"
                      ? t(
                          "A CSV built in your browser, up to {0} rows.",
                          EXPORT_MAX_ROWS.toLocaleString(),
                        )
                      : destination === "print"
                        ? t(
                            "Opens your browser's print dialog, up to {0} rows. Choose Save as PDF there for a file.",
                            PRINT_MAX_ROWS.toLocaleString(),
                          )
                        : destination === "report"
                          ? t(
                              "Built on the server in Excel, CSV or PDF with every matching row. You are told when it is ready.",
                            )
                          : t("Built on the server with every matching row, each time it runs.")
                  }
                />
              </FormControl>

              {destination === "download" || destination === "print" ? (
                <FormControl cols="full">
                  <Label className="text-muted-foreground text-xs font-medium">{t("Rows")}</Label>
                  <Controller
                    control={control}
                    name="scope"
                    render={({ field }) => (
                      <div
                        className="mt-2 flex gap-2"
                        role="radiogroup"
                        aria-label={t("Export scope")}
                      >
                        <ChoiceButton
                          selected={field.value === "all"}
                          onClick={() => field.onChange("all")}
                        >
                          <span className="font-medium">{t("All matching")}</span>
                          {totalCount != null ? (
                            <span className="text-muted-foreground text-xs">
                              {t("{0} rows", Math.min(totalCount, localCap).toLocaleString())}
                            </span>
                          ) : null}
                        </ChoiceButton>
                        <ChoiceButton
                          selected={field.value === "page"}
                          onClick={() => field.onChange("page")}
                        >
                          <span className="font-medium">{t("Current page")}</span>
                          <span className="text-muted-foreground text-xs">
                            {t("{0} rows", currentPageRows.length)}
                          </span>
                        </ChoiceButton>
                      </div>
                    )}
                  />
                  {overCap ? (
                    <p className="text-muted-foreground mt-2 text-xs">
                      {permissionResource
                        ? t(
                            "This stops at {0} rows. Export as a report to get every row.",
                            localCap.toLocaleString(),
                          )
                        : t(
                            "This stops at {0} rows. Narrow your filters to a specific slice.",
                            localCap.toLocaleString(),
                          )}
                    </p>
                  ) : null}
                </FormControl>
              ) : (
                <>
                  <FormControl cols="full">
                    <InputField
                      name="name"
                      control={control}
                      label={t("Report name")}
                      rules={{ required: true }}
                    />
                  </FormControl>
                  <FormControl cols="full">
                    <SegmentedField
                      name="format"
                      control={control}
                      label={t("Format")}
                      options={[
                        { value: "xlsx", label: t("Excel") },
                        { value: "csv", label: t("CSV") },
                        { value: "pdf", label: t("PDF") },
                      ]}
                    />
                  </FormControl>
                </>
              )}

              {destination === "schedule" ? (
                <>
                  <FormControl cols="full">
                    <SegmentedField
                      name="frequency"
                      control={control}
                      label={t("Runs")}
                      description={t("At 7:00 in your time zone.")}
                      options={[
                        { value: "daily", label: t("Every day") },
                        { value: "weekly", label: t("Every Monday") },
                        { value: "monthly", label: t("On the 1st") },
                      ]}
                    />
                  </FormControl>
                  <FormControl cols="full">
                    <EmailChipsField
                      name="recipients"
                      control={control}
                      label={t("Email to")}
                      description={t("Leave empty to keep each run in Reports only.")}
                      placeholder={t("name@company.com")}
                    />
                  </FormControl>
                </>
              ) : null}

              <FormControl cols="full">
                <Label className="text-muted-foreground text-xs font-medium">{t("Columns")}</Label>
                <Controller
                  control={control}
                  name="columnsMode"
                  render={({ field }) => (
                    <div
                      className="mt-2 flex gap-2"
                      role="radiogroup"
                      aria-label={t("Export columns")}
                    >
                      <ChoiceButton
                        selected={field.value === "visible"}
                        onClick={() => field.onChange("visible")}
                      >
                        <span className="font-medium">{t("Visible columns")}</span>
                        <span className="text-muted-foreground text-xs">
                          {t("Matches the table layout")}
                        </span>
                      </ChoiceButton>
                      <ChoiceButton
                        selected={field.value === "all"}
                        onClick={() => field.onChange("all")}
                      >
                        <span className="font-medium">{t("All columns")}</span>
                        <span className="text-muted-foreground text-xs">
                          {t("Every exportable field")}
                        </span>
                      </ChoiceButton>
                    </div>
                  )}
                />
              </FormControl>
            </FormGroup>
            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => handleOpenChange(false)}>
                {t("Cancel")}
              </Button>
              <Button
                type="submit"
                isLoading={form.formState.isSubmitting}
                loadingText={progress ?? t("Exporting...")}
              >
                {destination === "schedule"
                  ? t("Schedule")
                  : destination === "print"
                    ? t("Print")
                    : t("Export")}
              </Button>
            </DialogFooter>
          </Form>
        </FormProvider>
      </DialogContent>
    </Dialog>
  );
}
