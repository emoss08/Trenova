import { useT } from "@trenova/shared/i18n/use-t";
import { TextareaField } from "@/components/fields/textarea-field";
import { useApiMutation } from "@/hooks/use-api-mutation";
import {
  PAYROLL_EXPORTS_KEY,
  TIMESHEETS_KEY,
  voidPayrollExport,
  type PayrollExportRow,
} from "@/lib/graphql/timesheet";
import { zodResolver } from "@hookform/resolvers/zod";
import { useQueryClient } from "@tanstack/react-query";
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
import { formatShiftDate } from "@trenova/shared/lib/scheduling";
import {
  voidPayrollExportFormSchema,
  type VoidPayrollExportFormValues,
} from "@trenova/shared/types/timesheet";
import { useEffect } from "react";
import { FormProvider, useForm, type Resolver } from "react-hook-form";
import { toast } from "sonner";

export type VoidExportDialogProps = {
  run: PayrollExportRow | null;
  onOpenChange: (open: boolean) => void;
};

/**
 * Taking a payroll run back. Voiding one reopens every week in it, so the
 * reason is required: nobody can explain the reopening afterwards otherwise.
 */
export function VoidExportDialog({ run, onOpenChange }: VoidExportDialogProps) {
  const t = useT();

  const queryClient = useQueryClient();
  const form = useForm<VoidPayrollExportFormValues>({
    resolver: zodResolver(voidPayrollExportFormSchema) as Resolver<VoidPayrollExportFormValues>,
    defaultValues: { reason: "" },
  });
  const { control, handleSubmit, reset } = form;

  useEffect(() => {
    if (run) reset({ reason: "" });
  }, [run, reset]);

  const { mutateAsync, isPending } = useApiMutation<
    { id: string },
    VoidPayrollExportFormValues,
    unknown,
    VoidPayrollExportFormValues
  >({
    form,
    resourceName: "Payroll run",
    mutationFn: (values) => voidPayrollExport({ id: run?.id ?? "", reason: values.reason }),
    onSuccess: () => {
      toast.success(t("Run voided — the weeks in it are back to approved"));
      void queryClient.invalidateQueries({ queryKey: [PAYROLL_EXPORTS_KEY] });
      void queryClient.invalidateQueries({ queryKey: [TIMESHEETS_KEY] });
      onOpenChange(false);
    },
  });

  return (
    <Dialog open={run !== null} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>{t("Void this payroll run")}</DialogTitle>
          <DialogDescription>
            {t(
              "{0} Every week in it goes back to approved so the period can be run again once whatever was wrong is fixed. The run itself is kept, not deleted.",
              run
                ? t(
                    "{0} – {1} · {2, plural, one {# timesheet} other {# timesheets}}.",
                    formatShiftDate(run.periodStart),
                    formatShiftDate(run.periodEnd - 86400),
                    run.timesheetCount,
                  )
                : "",
            )}
          </DialogDescription>
        </DialogHeader>
        <FormProvider {...form}>
          <Form
            onSubmit={(submitEvent) => {
              submitEvent.preventDefault();
              submitEvent.stopPropagation();
              void handleSubmit((values) => mutateAsync(values))(submitEvent);
            }}
          >
            <FormGroup className="pb-2" cols={1}>
              <FormControl cols="full">
                <TextareaField<VoidPayrollExportFormValues>
                  control={control}
                  name="reason"
                  label={t("Why")}
                  placeholder={t("e.g. Wrong period — the Friday sheets were still open")}
                  description={t(
                    "Kept with the voided run so the reopened weeks can be explained later.",
                  )}
                  rules={{ required: true }}
                />
              </FormControl>
            </FormGroup>
            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
                {t("Keep it")}
              </Button>
              <Button type="submit" variant="destructive" isLoading={isPending}>
                {t("Void run")}
              </Button>
            </DialogFooter>
          </Form>
        </FormProvider>
      </DialogContent>
    </Dialog>
  );
}
