import { AutocompleteField } from "@/components/fields/autocomplete/autocomplete";
import { SelectField } from "@/components/fields/select-field";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { startBulkEdit, type BulkEditJob } from "@/lib/graphql/bulk-edit";
import type { SelectOption as RecordOption } from "@/lib/graphql/select-options";
import { queries } from "@/lib/queries";
import { recordOptionLabel } from "@/lib/select-option-meta";
import { zodResolver } from "@hookform/resolvers/zod";
import type {
  BulkEditSelectionInput,
  SelectOptionResource,
} from "@trenova/graphql/generated/graphql";
import { useQuery } from "@tanstack/react-query";
import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
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
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { translate } from "@trenova/shared/i18n/runtime";
import { useT } from "@trenova/shared/i18n/use-t";
import { useEffect } from "react";
import { FormProvider, useForm, useWatch } from "react-hook-form";
import { z } from "zod";

const bulkEditFormSchema = z.object({
  field: z.string().min(1, { error: () => translate("Choose a field to change") }),
  value: z.string().min(1, { error: () => translate("Choose the new value") }),
});

type BulkEditFormValues = z.infer<typeof bulkEditFormSchema>;

const NO_FIELDS: never[] = [];

type BulkEditDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  /** The table's permission resource, such as shipment. */
  resource: string;
  /** The rows chosen one by one, or every row matching the table's filters. */
  selection: BulkEditSelectionInput;
  onStarted: (job: BulkEditJob) => void;
};

/**
 * Changes one field on many rows at once. It says how many rows the change reaches
 * before it starts, runs it in the background, and the change can be undone for a
 * day after it finishes.
 */
export function BulkEditDialog({
  open,
  onOpenChange,
  resource,
  selection,
  onStarted,
}: BulkEditDialogProps) {
  const t = useT();
  const form = useForm<BulkEditFormValues>({
    resolver: zodResolver(bulkEditFormSchema),
    defaultValues: { field: "", value: "" },
  });
  const { control, handleSubmit, reset, setValue } = form;
  const fieldName = useWatch({ control, name: "field" });

  useEffect(() => {
    if (open) reset({ field: "", value: "" });
  }, [open, reset]);

  const fieldsQuery = useQuery({
    ...queries.bulkEdit.fields(resource),
    enabled: open,
    staleTime: Infinity,
  });
  const fields = fieldsQuery.data ?? NO_FIELDS;
  const field = fields.find((entry) => entry.name === fieldName);

  useEffect(() => {
    if (!fieldName && fields.length === 1) setValue("field", fields[0].name);
  }, [fieldName, fields, setValue]);

  const previewQuery = useQuery({
    ...queries.bulkEdit.preview({ resource, field: "", value: "", selection }),
    enabled: open,
    staleTime: 10_000,
  });
  const preview = previewQuery.data;

  const { mutateAsync } = useApiMutation<
    BulkEditJob,
    BulkEditFormValues,
    unknown,
    BulkEditFormValues
  >({
    mutationFn: (values) =>
      startBulkEdit({ resource, field: values.field, value: values.value, selection }),
    resourceName: "Bulk edit",
    form,
    onSuccess: (job) => {
      onStarted(job);
      onOpenChange(false);
    },
  });

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent size="sm">
        <DialogHeader>
          <DialogTitle>{t("Edit rows")}</DialogTitle>
          <DialogDescription>
            {t("Set one field to the same value on every row below. You can undo it for a day.")}
          </DialogDescription>
        </DialogHeader>
        <FormProvider {...form}>
        <Form onSubmit={handleSubmit((values) => mutateAsync(values))}>
          <FormGroup cols={1} className="pb-4">
            <FormControl cols="full">
              <SelectField
                name="field"
                control={control}
                label={t("Field")}
                placeholder={fieldsQuery.isPending ? t("Loading fields...") : t("Choose a field")}
                options={fields.map((entry) => ({ value: entry.name, label: t(entry.label) }))}
                rules={{ required: true }}
                onValueChange={() => setValue("value", "")}
              />
            </FormControl>
            {field ? (
              <FormControl cols="full">
                {field.kind === "record" && field.record ? (
                  <AutocompleteField<RecordOption, BulkEditFormValues>
                    name="value"
                    control={control}
                    label={t("New value")}
                    placeholder={t("Choose a record")}
                    graphql={{ resource: field.record as SelectOptionResource }}
                    getOptionValue={(option) => option.id}
                    getDisplayValue={recordOptionLabel}
                    renderOption={(option) => (
                      <span className="truncate">{recordOptionLabel(option)}</span>
                    )}
                    rules={{ required: true }}
                  />
                ) : (
                  <SelectField
                    name="value"
                    control={control}
                    label={t("New value")}
                    placeholder={t("Choose a value")}
                    options={field.options.map((option) => ({
                      value: option.value,
                      label: t(option.label),
                    }))}
                    rules={{ required: true }}
                  />
                )}
              </FormControl>
            ) : null}
            <FormControl cols="full">
              {previewQuery.isPending ? (
                <Skeleton className="h-4 w-40" />
              ) : preview?.tooMany ? (
                <Alert size="sm" variant="destructive">
                  <AlertDescription>
                    {t(
                      "{0} rows match, more than one bulk edit can change. Narrow the filters first.",
                      preview.count.toLocaleString(),
                    )}
                  </AlertDescription>
                </Alert>
              ) : preview ? (
                <p className="text-muted-foreground text-sm">
                  {t(
                    "{0, plural, one {This changes # row.} other {This changes # rows.}}",
                    preview.count,
                  )}
                </p>
              ) : previewQuery.isError ? (
                <Alert size="sm" variant="destructive">
                  <AlertDescription>{t("The rows could not be counted.")}</AlertDescription>
                </Alert>
              ) : null}
            </FormControl>
          </FormGroup>
          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
              {t("Cancel")}
            </Button>
            <Button
              type="submit"
              disabled={!preview || preview.tooMany || preview.count === 0}
              isLoading={form.formState.isSubmitting}
              loadingText={t("Starting...")}
            >
              {preview && !preview.tooMany
                ? t("{0, plural, one {Change # row} other {Change # rows}}", preview.count)
                : t("Change rows")}
            </Button>
          </DialogFooter>
        </Form>
        </FormProvider>
      </DialogContent>
    </Dialog>
  );
}
