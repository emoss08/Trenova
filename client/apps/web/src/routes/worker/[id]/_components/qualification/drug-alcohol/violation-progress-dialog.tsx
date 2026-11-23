import { useT } from "@trenova/shared/i18n/use-t";
import { AutoCompleteDateField } from "@/components/fields/date-field/date-field";
import { InputField } from "@/components/fields/input-field";
import { NumberField } from "@/components/fields/number-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { updateDotViolation, type DotViolationRow } from "@/lib/graphql/worker-drug-alcohol";
import { zodResolver } from "@hookform/resolvers/zod";
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
  violationProgressFormSchema,
  type ViolationProgressFormValues,
} from "@trenova/shared/types/worker-drug-alcohol";
import { FormProvider, useForm, type Resolver } from "react-hook-form";
import { useWorkerInvalidation } from "../../use-worker-invalidation";

export type ViolationProgressDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  workerId: string;
  violation: DotViolationRow | null;
};

function defaultsFor(violation: DotViolationRow): ViolationProgressFormValues {
  return {
    sapName: violation.sapName ?? null,
    sapReferredAt: violation.sapReferredAt ?? null,
    sapEvaluationCompletedAt: violation.sapEvaluationCompletedAt ?? null,
    followUpTestCount: violation.followUpTestCount ?? null,
    followUpEndsAt: violation.followUpEndsAt ?? null,
    reportedToClearinghouseAt: violation.reportedToClearinghouseAt ?? null,
    notes: violation.notes ?? null,
  };
}

/**
 * The return-to-duty process of an open violation, edited in a dialog. Its form lives inside the
 * dialog's content, which is mounted only while it is open, so each opening starts from the
 * violation as it stands and a refetch never replaces what is being typed.
 */
export function ViolationProgressDialog({
  open,
  onOpenChange,
  workerId,
  violation,
}: ViolationProgressDialogProps) {
  const t = useT();
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent size="lg">
        <DialogHeader>
          <DialogTitle>{t("Return-to-duty process")}</DialogTitle>
          <DialogDescription>
            {t(
              "The stage is read off what has been recorded, so it can never drift from the file underneath it.",
            )}
          </DialogDescription>
        </DialogHeader>
        {violation ? (
          <ViolationProgressForm
            workerId={workerId}
            violation={violation}
            onClose={() => onOpenChange(false)}
          />
        ) : null}
      </DialogContent>
    </Dialog>
  );
}

function ViolationProgressForm({
  workerId,
  violation,
  onClose,
}: {
  workerId: string;
  violation: DotViolationRow;
  onClose: () => void;
}) {
  const t = useT();
  const invalidate = useWorkerInvalidation(workerId, "testing");
  const form = useForm<ViolationProgressFormValues>({
    resolver: zodResolver(violationProgressFormSchema) as Resolver<ViolationProgressFormValues>,
    defaultValues: defaultsFor(violation),
  });
  const { control, handleSubmit } = form;

  const { mutateAsync, isPending } = useApiMutation<
    DotViolationRow,
    ViolationProgressFormValues,
    unknown,
    ViolationProgressFormValues
  >({
    form,
    resourceName: "Violation",
    mutationFn: (values) =>
      updateDotViolation({
        violationId: violation.id,
        sapName: values.sapName ?? undefined,
        sapReferredAt: values.sapReferredAt ?? undefined,
        sapEvaluationCompletedAt: values.sapEvaluationCompletedAt ?? undefined,
        followUpTestCount: values.followUpTestCount ?? undefined,
        followUpEndsAt: values.followUpEndsAt ?? undefined,
        reportedToClearinghouseAt: values.reportedToClearinghouseAt ?? undefined,
        notes: values.notes ?? undefined,
      }),
    onSuccess: () => {
      void invalidate();
      onClose();
    },
  });

  return (
    <FormProvider {...form}>
      <Form
        className="flex flex-col gap-4"
        onSubmit={(submitEvent) => {
          submitEvent.preventDefault();
          submitEvent.stopPropagation();
          void handleSubmit((values) => mutateAsync(values))(submitEvent);
        }}
      >
        <FormGroup cols={2}>
          <FormControl cols="full">
            <Alert size="sm">
              <AlertDescription>
                {t(
                  "The driver returns to duty once a return-to-duty test is recorded and passed — that is done from the test list, not here. Follow-up testing then runs with the driver back at work.",
                )}
              </AlertDescription>
            </Alert>
          </FormControl>
          <FormControl>
            <InputField<ViolationProgressFormValues>
              control={control}
              name="sapName"
              label={t("Substance abuse professional")}
              placeholder={t("Name of the SAP")}
              description={t(
                "The substance abuse professional handling the evaluation and follow-up plan.",
              )}
            />
          </FormControl>
          <FormControl>
            <AutoCompleteDateField<ViolationProgressFormValues>
              control={control}
              name="sapReferredAt"
              label={t("Referred on")}
              placeholder={t("MM/DD/YYYY")}
              description={t("When the driver was given the SAP referral.")}
            />
          </FormControl>
          <FormControl>
            <AutoCompleteDateField<ViolationProgressFormValues>
              control={control}
              name="sapEvaluationCompletedAt"
              label={t("Evaluation completed")}
              placeholder={t("MM/DD/YYYY")}
              description={t(
                "A passed return-to-duty test only releases the driver once this date is set.",
              )}
            />
          </FormControl>
          <FormControl>
            <AutoCompleteDateField<ViolationProgressFormValues>
              control={control}
              name="reportedToClearinghouseAt"
              label={t("Reported to the clearinghouse")}
              placeholder={t("MM/DD/YYYY")}
              description={t("When the violation was reported to the Clearinghouse.")}
            />
          </FormControl>
          <FormControl>
            <NumberField<ViolationProgressFormValues>
              control={control}
              name="followUpTestCount"
              placeholder={t("e.g. 6")}
              label={t("Follow-up tests required")}
              description={t("At least six in the first twelve months (49 CFR 382.311).")}
            />
          </FormControl>
          <FormControl>
            <AutoCompleteDateField<ViolationProgressFormValues>
              control={control}
              name="followUpEndsAt"
              label={t("Follow-up program ends")}
              placeholder={t("MM/DD/YYYY")}
              description={t(
                "When the SAP's follow-up plan ends; it may run up to five years after the return to duty.",
              )}
            />
          </FormControl>
          <FormControl cols="full">
            <TextareaField<ViolationProgressFormValues>
              control={control}
              name="notes"
              placeholder={t("e.g. Driver began treatment on 3/12")}
              description={t("Internal notes kept with the violation.")}
              label={t("Notes")}
              maxLength={2000}
            />
          </FormControl>
        </FormGroup>
        <DialogFooter>
          <Button type="button" size="sm" variant="outline" onClick={onClose}>
            {t("Cancel")}
          </Button>
          <Button type="submit" size="sm" isLoading={isPending}>
            {t("Save changes")}
          </Button>
        </DialogFooter>
      </Form>
    </FormProvider>
  );
}
