import { SelectField } from "@/components/fields/select-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { detentionWaiverReasonChoices } from "@/lib/choices";
import { apiService } from "@/services/api";
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
import { translate } from "@trenova/shared/i18n/runtime";
import { useT } from "@trenova/shared/i18n/use-t";
import { waiverReasonSchema, type DetentionOccurrence } from "@trenova/shared/types/detention";
import { FormProvider, useForm } from "react-hook-form";
import { z } from "zod";

/*
Waiving and disputing a detention charge, each a dialog that asks for the
reason the server records with it. Their own module, so the detention desk's
sheet and the Desk's detention card share them without one bringing the other.
*/

const waiveFormSchema = z.object({
  reason: waiverReasonSchema,
  note: z.string().min(1, { error: () => translate("Explain why this charge is being forgiven") }),
});
type WaiveFormValues = z.infer<typeof waiveFormSchema>;

const disputeFormSchema = z.object({
  note: z.string().min(1, { error: () => translate("Record what the customer is disputing") }),
});
type DisputeFormValues = z.infer<typeof disputeFormSchema>;

export type ActionDialogProps = {
  occurrenceId: string;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onDone: () => void;
};

export function WaiveDialog({ occurrenceId, open, onOpenChange, onDone }: ActionDialogProps) {
  const t = useT();

  const form = useForm<WaiveFormValues>({
    resolver: zodResolver(waiveFormSchema),
    defaultValues: { reason: "CustomerGoodwill", note: "" },
  });
  const { control, handleSubmit, reset } = form;

  const { mutateAsync, isPending } = useApiMutation<
    DetentionOccurrence,
    WaiveFormValues,
    unknown,
    WaiveFormValues
  >({
    mutationFn: (values) => apiService.detentionService.waive(occurrenceId, values),
    onSuccess: () => {
      reset();
      onOpenChange(false);
      onDone();
    },
    form,
    resourceName: "Detention Occurrence",
  });

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{t("Waive this charge")}</DialogTitle>
          <DialogDescription>
            {t(
              "Waiving forgives the money but keeps the record, so discretionary revenue loss stays measurable instead of disappearing into free text.",
            )}
          </DialogDescription>
        </DialogHeader>
        <FormProvider {...form}>
          <Form
            className="flex flex-col gap-4"
            onSubmit={handleSubmit((values) => mutateAsync(values))}
          >
            <FormGroup cols={1}>
              <FormControl>
                <SelectField
                  control={control}
                  name="reason"
                  label={t("Reason")}
                  placeholder={t("Select a coded reason")}
                  rules={{ required: true }}
                  options={detentionWaiverReasonChoices}
                  description={t("The coded reason waiver leakage is reported against.")}
                />
              </FormControl>
              <FormControl>
                <TextareaField
                  control={control}
                  name="note"
                  label={t("Note")}
                  placeholder={t(
                    "Dock crew was short-staffed; customer asked for a one-time concession",
                  )}
                  rules={{ required: true }}
                  description={t("Context the next person reviewing this waiver will need.")}
                />
              </FormControl>
            </FormGroup>
            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
                {t("Cancel")}
              </Button>
              <Button type="submit" isLoading={isPending} loadingText={t("Waiving")}>
                {t("Waive charge")}
              </Button>
            </DialogFooter>
          </Form>
        </FormProvider>
      </DialogContent>
    </Dialog>
  );
}

export function DisputeDialog({ occurrenceId, open, onOpenChange, onDone }: ActionDialogProps) {
  const t = useT();

  const form = useForm<DisputeFormValues>({
    resolver: zodResolver(disputeFormSchema),
    defaultValues: { note: "" },
  });
  const { control, handleSubmit, reset } = form;

  const { mutateAsync, isPending } = useApiMutation<
    DetentionOccurrence,
    DisputeFormValues,
    unknown,
    DisputeFormValues
  >({
    mutationFn: (values) => apiService.detentionService.dispute(occurrenceId, values),
    onSuccess: () => {
      reset();
      onOpenChange(false);
      onDone();
    },
    form,
    resourceName: "Detention Occurrence",
  });

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{t("Record a dispute")}</DialogTitle>
          <DialogDescription>
            {t(
              "Recording the customer's rejection keeps the original computation intact — exactly what working the claim requires.",
            )}
          </DialogDescription>
        </DialogHeader>
        <FormProvider {...form}>
          <Form
            className="flex flex-col gap-4"
            onSubmit={handleSubmit((values) => mutateAsync(values))}
          >
            <FormGroup cols={1}>
              <FormControl>
                <TextareaField
                  control={control}
                  name="note"
                  label={t("What is the customer disputing?")}
                  placeholder={t(
                    "Customer claims the driver arrived 40 minutes later than our records show",
                  )}
                  rules={{ required: true }}
                  description={t(
                    "Their claim, verbatim where possible — it decides which evidence matters.",
                  )}
                />
              </FormControl>
            </FormGroup>
            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
                {t("Cancel")}
              </Button>
              <Button type="submit" isLoading={isPending} loadingText={t("Recording")}>
                {t("Record dispute")}
              </Button>
            </DialogFooter>
          </Form>
        </FormProvider>
      </DialogContent>
    </Dialog>
  );
}
