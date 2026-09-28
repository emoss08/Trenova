import { DocumentTypeAutocompleteField } from "@/components/autocomplete-fields";
import { NumberField } from "@/components/fields/number-field";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { captureRecordKindLabel, type CaptureRecordKind } from "@/lib/capture";
import {
  buildCoverSheetPdf,
  openCoverSheetsForPrinting,
  type CoverSheetPrint,
} from "@/lib/capture-cover-sheet";
import {
  MAX_COVER_SHEETS,
  coverSheetFormSchema,
  emptyToNull,
  type CoverSheetFormValues,
} from "@/lib/capture-forms";
import { createCaptureCoverSheets, type IssuedCoverSheet } from "@/lib/graphql/capture";
import { selectOptionMetaString } from "@/lib/select-option-meta";
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
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixDateMedium } from "@trenova/shared/lib/date";
import { useEffect, useRef } from "react";
import { FormProvider, useForm, type Resolver } from "react-hook-form";
import { toast } from "sonner";

const DEFAULT_VALUES: CoverSheetFormValues = { copies: 1, documentTypeId: "" };

/**
 * Prints cover sheets for this record. A sheet on top of a stack of paper
 * files everything after it here, until the next sheet, wherever the stack is
 * scanned. Each sheet's code is issued once; a lost sheet is replaced with a
 * new one, not reprinted.
 */
export function CoverSheetDialog({
  open,
  onOpenChange,
  kind,
  recordId,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  kind: CaptureRecordKind;
  recordId: string;
}) {
  const t = useT();
  const form = useForm<CoverSheetFormValues>({
    resolver: zodResolver(coverSheetFormSchema) as Resolver<CoverSheetFormValues>,
    defaultValues: DEFAULT_VALUES,
  });
  const { control, handleSubmit, reset } = form;
  // The chosen type's name is printed on the sheet; only its id is in the form.
  const documentTypeName = useRef<string | null>(null);

  useEffect(() => {
    if (open) {
      reset(DEFAULT_VALUES);
      documentTypeName.current = null;
    }
  }, [open, reset]);

  const print = useApiMutation<Uint8Array, CoverSheetFormValues, unknown, CoverSheetFormValues>({
    form,
    resourceName: "Cover sheet",
    mutationFn: async (values) => {
      const issued = await createCaptureCoverSheets(
        Array.from({ length: values.copies }, () => ({
          targetType: kind,
          targetId: recordId,
          documentTypeId: emptyToNull(values.documentTypeId),
        })),
      );
      const kindLabel = captureRecordKindLabel(t, kind);
      const sheets: CoverSheetPrint[] = issued.map((sheet: IssuedCoverSheet) => ({
        id: sheet.id,
        modules: sheet.qrCode.modules,
        record: sheet.target
          ? { kind: kindLabel, title: sheet.target.title, subtitle: sheet.target.subtitle }
          : { kind: kindLabel, title: recordId, subtitle: "" },
        documentType: values.documentTypeId === "" ? null : documentTypeName.current,
        expiresOn: formatUnixDateMedium(sheet.expiresAt),
      }));

      return buildCoverSheetPdf(sheets, {
        heading: t("Trenova cover sheet"),
        separatorTitle: t("Separator"),
        documentTypeLabel: t("Document type"),
        instructions: t(
          "Put this sheet on top of the pages that belong here and scan the stack. Every page after it is filed onto this record, until the next cover sheet.",
        ),
        separatorInstructions: t("Put this sheet between two documents to split the stack there."),
        expires: (date) => t("Use by {0}", date),
      });
    },
    onSuccess: (bytes, values) => {
      const how = openCoverSheetsForPrinting(bytes, `cover-sheets-${kind}.pdf`);
      toast.success(
        how === "opened"
          ? t(
              "{0, plural, one {Cover sheet ready to print} other {# cover sheets ready to print}}",
              values.copies,
            )
          : t(
              "{0, plural, one {Cover sheet downloaded; open it to print} other {# cover sheets downloaded; open them to print}}",
              values.copies,
            ),
      );
      onOpenChange(false);
    },
  });

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent size="sm">
        <DialogHeader>
          <DialogTitle>{t("Print cover sheets")}</DialogTitle>
          <DialogDescription>
            {t(
              "Put a cover sheet on top of the paper for this record. Wherever the stack is scanned, the pages after it are filed here. A separator sheet comes with each one, to split one stack into documents.",
            )}
          </DialogDescription>
        </DialogHeader>
        <FormProvider {...form}>
          <Form
            className="flex flex-col gap-4"
            onSubmit={(submitEvent) => {
              submitEvent.preventDefault();
              submitEvent.stopPropagation();
              void handleSubmit((values) => print.mutateAsync(values))(submitEvent);
            }}
          >
            <FormGroup cols={1}>
              <FormControl>
                <NumberField<CoverSheetFormValues>
                  control={control}
                  name="copies"
                  label={t("Cover sheets")}
                  description={t("Up to {0} at a time.", MAX_COVER_SHEETS)}
                  min={1}
                  max={MAX_COVER_SHEETS}
                  step={1}
                  rules={{ required: true }}
                />
              </FormControl>
              <FormControl>
                <DocumentTypeAutocompleteField<CoverSheetFormValues>
                  control={control}
                  name="documentTypeId"
                  label={t("Document type")}
                  placeholder={t("Optional")}
                  onOptionChange={(option) => {
                    documentTypeName.current = option
                      ? selectOptionMetaString(option, "name") || option.label
                      : null;
                  }}
                />
              </FormControl>
            </FormGroup>
            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
                {t("Cancel")}
              </Button>
              <Button type="submit" isLoading={print.isPending} loadingText={t("Preparing")}>
                {t("Print cover sheets")}
              </Button>
            </DialogFooter>
          </Form>
        </FormProvider>
      </DialogContent>
    </Dialog>
  );
}
