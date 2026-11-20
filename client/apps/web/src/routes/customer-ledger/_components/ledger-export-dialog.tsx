import { AutoCompleteDateField } from "@/components/fields/date-field/date-field";
import { InputField } from "@/components/fields/input-field";
import { SegmentedField } from "@/components/fields/segmented-field";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { exportCustomerLedger } from "@/lib/graphql/table-export";
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
import { toast } from "@trenova/shared/components/ui/toast";
import { translate } from "@trenova/shared/i18n/runtime";
import { useT } from "@trenova/shared/i18n/use-t";
import { useEffect } from "react";
import { FormProvider, useForm } from "react-hook-form";
import { z } from "zod";

const SECONDS_IN_DAY = 86_400;

const ledgerExportSchema = z
  .object({
    name: z
      .string()
      .trim()
      .min(1, { error: () => translate("Give the export a name") })
      .max(200, { error: () => translate("Keep the name under 200 characters") }),
    startDate: z.number().int().nullable(),
    endDate: z.number().int().nullable(),
    format: z.enum(["xlsx", "csv", "pdf"]),
  })
  .superRefine((values, ctx) => {
    if (values.startDate != null && values.endDate != null && values.startDate > values.endDate) {
      ctx.addIssue({
        code: "custom",
        path: ["endDate"],
        message: translate("End the export on or after its first day"),
      });
    }
  });

type LedgerExportValues = z.infer<typeof ledgerExportSchema>;

function ledgerExportDefaults(customerName: string | undefined): LedgerExportValues {
  return {
    name: customerName ? translate("{0} ledger", customerName) : translate("Customer ledger"),
    startDate: null,
    endDate: null,
    format: "xlsx",
  };
}

/** The last second of the day a date picker chose, so the export takes in the whole day. */
export function endOfChosenDay(day: number | null): number | undefined {
  return day == null ? undefined : day + SECONDS_IN_DAY - 1;
}

/**
 * Exports a customer's ledger over a window on the server: every entry in it, with its
 * running balance from what the customer owed before the first day. The file is built in
 * the background, kept in Reports, and the person is told when it is ready.
 */
export function LedgerExportDialog({
  open,
  onOpenChange,
  customerId,
  customerName,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  customerId: string;
  customerName: string | undefined;
}) {
  const t = useT();
  const form = useForm<LedgerExportValues>({
    resolver: zodResolver(ledgerExportSchema),
    defaultValues: ledgerExportDefaults(customerName),
  });
  const { control, handleSubmit, reset } = form;

  useEffect(() => {
    if (open) reset(ledgerExportDefaults(customerName));
  }, [open, reset, customerName]);

  const { mutateAsync, isPending } = useApiMutation<
    { definitionId: string },
    LedgerExportValues,
    unknown,
    LedgerExportValues
  >({
    form,
    resourceName: "Export",
    mutationFn: (values) =>
      exportCustomerLedger({
        name: values.name,
        customerId,
        startDate: values.startDate ?? undefined,
        endDate: endOfChosenDay(values.endDate),
        format: values.format,
      }),
    onSuccess: () => {
      toast.success(t("Your export is being built"), {
        description: t("You will be told when the file is ready. It is kept in Reports."),
      });
      onOpenChange(false);
    },
  });

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent size="md">
        <DialogHeader>
          <DialogTitle>{t("Export ledger")}</DialogTitle>
          <DialogDescription>
            {t(
              "Every entry in the window with its running balance, built on the server. Balances start from what the customer owed before the first day.",
            )}
          </DialogDescription>
        </DialogHeader>
        <FormProvider {...form}>
          <Form
            onSubmit={(event) => {
              event.preventDefault();
              event.stopPropagation();
              void handleSubmit((values) => mutateAsync(values))(event);
            }}
          >
            <FormGroup cols={2} className="pb-4">
              <FormControl cols="full">
                <InputField<LedgerExportValues>
                  control={control}
                  name="name"
                  label={t("Name")}
                  required
                  description={t("How the export is named in Reports.")}
                />
              </FormControl>
              <FormControl>
                <AutoCompleteDateField<LedgerExportValues>
                  control={control}
                  name="startDate"
                  label={t("From")}
                  placeholder={t("The first entry")}
                />
              </FormControl>
              <FormControl>
                <AutoCompleteDateField<LedgerExportValues>
                  control={control}
                  name="endDate"
                  label={t("Through")}
                  placeholder={t("Today")}
                />
              </FormControl>
              <FormControl cols="full">
                <SegmentedField<LedgerExportValues, LedgerExportValues["format"]>
                  control={control}
                  name="format"
                  label={t("Format")}
                  options={[
                    { value: "xlsx", label: t("Excel") },
                    { value: "csv", label: t("CSV") },
                    { value: "pdf", label: t("PDF") },
                  ]}
                />
              </FormControl>
            </FormGroup>
            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
                {t("Cancel")}
              </Button>
              <Button type="submit" isLoading={isPending} loadingText={t("Exporting...")}>
                {t("Export")}
              </Button>
            </DialogFooter>
          </Form>
        </FormProvider>
      </DialogContent>
    </Dialog>
  );
}
