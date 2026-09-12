import { useT } from "@trenova/shared/i18n/use-t";
import { AutoCompleteDateField } from "@/components/fields/date-field/date-field";
import { InputField } from "@/components/fields/input-field";
import { NumberField } from "@/components/fields/number-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { OSHA_LOG_KEY, saveOshaSummary, type OshaSummary } from "@/lib/graphql/worker-injury";
import { zodResolver } from "@hookform/resolvers/zod";
import { useQueryClient } from "@tanstack/react-query";
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
import {
  oshaSummaryFormSchema,
  type OSHASummaryFormValues,
} from "@trenova/shared/types/worker-injury";
import { useEffect } from "react";
import { FormProvider, useForm, type Resolver } from "react-hook-form";
import { toast } from "sonner";

export type OshaSummaryDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  year: number;
  summary: OshaSummary | null;
};

function defaultsFor(summary: OshaSummary | null): OSHASummaryFormValues {
  return {
    naicsCode: summary?.naicsCode ?? null,
    averageEmployees: summary?.averageEmployees ?? 0,
    totalHoursWorked: summary?.totalHoursWorked ?? 0,
    executiveName: summary?.executiveName ?? null,
    executiveTitle: summary?.executiveTitle ?? null,
    executivePhone: summary?.executivePhone ?? null,
    submittedAt: summary?.submittedAt ?? null,
    submissionReference: summary?.submissionReference ?? null,
    notes: summary?.notes ?? null,
  };
}

export function OshaSummaryDialog({ open, onOpenChange, year, summary }: OshaSummaryDialogProps) {
  const t = useT();

  const queryClient = useQueryClient();
  const form = useForm<OSHASummaryFormValues>({
    resolver: zodResolver(oshaSummaryFormSchema) as Resolver<OSHASummaryFormValues>,
    defaultValues: defaultsFor(summary),
  });
  const { control, handleSubmit, reset } = form;

  useEffect(() => {
    if (!open) return;
    reset(defaultsFor(summary));
  }, [open, summary, reset]);

  const { mutateAsync, isPending } = useApiMutation<
    { id: string },
    OSHASummaryFormValues,
    unknown,
    OSHASummaryFormValues
  >({
    form,
    resourceName: "Summary",
    mutationFn: (values) =>
      saveOshaSummary({
        year,
        naicsCode: values.naicsCode ?? undefined,
        averageEmployees: values.averageEmployees,
        totalHoursWorked: values.totalHoursWorked,
        executiveName: values.executiveName ?? undefined,
        executiveTitle: values.executiveTitle ?? undefined,
        executivePhone: values.executivePhone ?? undefined,
        submittedAt: values.submittedAt ?? undefined,
        submissionReference: values.submissionReference ?? undefined,
        notes: values.notes ?? undefined,
      }),
    onSuccess: () => {
      toast.success(t("Summary saved"));
      void queryClient.invalidateQueries({ queryKey: [OSHA_LOG_KEY] });
      onOpenChange(false);
    },
  });

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-xl">
        <DialogHeader>
          <DialogTitle>{t("300A figures for {0}", year)}</DialogTitle>
          <DialogDescription>
            {t("The case totals come from the log. These are the establishment figures it cannot supply: how many people worked here and for how many hours.")}
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
            <FormGroup className="pb-2" cols={2}>
              <FormControl>
                <NumberField<OSHASummaryFormValues>
                  control={control}
                  name="averageEmployees"
                  label={t("Annual average employees")}
                  placeholder="42"
                  description={t("Average number of employees over the year, as OSHA Form 300A asks for it; required before the summary can be certified.")}
                  rules={{ required: true }}
                />
              </FormControl>
              <FormControl>
                <NumberField<OSHASummaryFormValues>
                  control={control}
                  name="totalHoursWorked"
                  label={t("Total hours worked")}
                  placeholder="240000"
                  description={t("Hours every employee worked during the year; the incident and DART rates are calculated from it.")}
                  rules={{ required: true }}
                />
              </FormControl>
              <FormControl>
                <InputField<OSHASummaryFormValues>
                  control={control}
                  name="naicsCode"
                  label={t("NAICS code")}
                  placeholder={t("e.g. 484121")}
                  description={t("The industry code printed on the 300A; 484121 is long-distance truckload freight.")}
                />
              </FormControl>
              <FormControl>
                <InputField<OSHASummaryFormValues>
                  control={control}
                  name="executiveName"
                  label={t("Certifying executive")}
                  placeholder={t("e.g. Jane Doe")}
                  description={t("The company executive who will certify the summary; a name is required before it can be certified.")}
                />
              </FormControl>
              <FormControl>
                <InputField<OSHASummaryFormValues>
                  control={control}
                  name="executiveTitle"
                  label={t("Title")}
                  placeholder={t("e.g. Vice President of Operations")}
                  description={t("The executive's job title, as it appears on the 300A.")}
                />
              </FormControl>
              <FormControl>
                <InputField<OSHASummaryFormValues>
                  control={control}
                  name="executivePhone"
                  label={t("Phone")}
                  placeholder={t("e.g. (555) 123-4567")}
                  description={t("A number where the executive can be reached about the summary.")}
                />
              </FormControl>
              <FormControl>
                <AutoCompleteDateField<OSHASummaryFormValues>
                  control={control}
                  name="submittedAt"
                  label={t("Electronically submitted")}
                  placeholder={t("e.g. Mar 2")}
                  description={t("The day the 300A was filed electronically with OSHA; leave it blank until it has been.")}
                />
              </FormControl>
              <FormControl>
                <InputField<OSHASummaryFormValues>
                  control={control}
                  name="submissionReference"
                  label={t("Submission reference")}
                  placeholder={t("e.g. 2026-0001234")}
                  description={t("The confirmation reference returned when the 300A was filed, so it can be found again.")}
                />
              </FormControl>
              <FormControl cols="full">
                <TextareaField<OSHASummaryFormValues>
                  control={control}
                  name="notes"
                  label={t("Notes")}
                  placeholder={t("e.g. Hours include the yard crew and seasonal drivers")}
                  description={t("Working notes on where the figures came from, for whoever prepares next year's form.")}
                  maxLength={2000}
                />
              </FormControl>
              <FormControl cols="full">
                <Alert>
                  <AlertDescription>
                    {t("Certifying is a separate step, and it is refused while cases are still accruing days — certifying a log that has not finished moving is certifying a number that is about to change.")}
                  </AlertDescription>
                </Alert>
              </FormControl>
            </FormGroup>
            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
                {t("Cancel")}
              </Button>
              <Button type="submit" isLoading={isPending}>
                {t("Save")}
              </Button>
            </DialogFooter>
          </Form>
        </FormProvider>
      </DialogContent>
    </Dialog>
  );
}
