import { useT } from "@trenova/shared/i18n/use-t";
import { AutoCompleteDateField } from "@/components/fields/date-field/date-field";
import { NumberField } from "@/components/fields/number-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { recordLeaveDay, type LeaveCaseRow } from "@/lib/graphql/worker-leave";
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
import { getTodayDate } from "@trenova/shared/lib/date";
import { leaveDayFormSchema, type LeaveDayFormValues } from "@trenova/shared/types/worker-leave";
import { FormProvider, useForm, type Resolver } from "react-hook-form";
import { useWorkerInvalidation } from "../../use-worker-invalidation";

export type LeaveDayDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  workerId: string;
  leaveCase: LeaveCaseRow | null;
};

function emptyDay(): LeaveDayFormValues {
  return { usedOn: getTodayDate(), hours: 8, notes: null };
}

/**
 * Recording a day of leave against an approved case. The form lives inside the dialog's content,
 * which mounts at each opening, so every opening starts from an empty day.
 */
export function LeaveDayDialog({ open, onOpenChange, workerId, leaveCase }: LeaveDayDialogProps) {
  const t = useT();
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent size="md">
        <DialogHeader>
          <DialogTitle>{t("Record a day of leave")}</DialogTitle>
          <DialogDescription>
            {t(
              "Hours rather than days, so intermittent leave can be taken in the increment the organisation actually uses.",
            )}
          </DialogDescription>
        </DialogHeader>
        {leaveCase ? (
          <LeaveDayForm
            workerId={workerId}
            leaveCase={leaveCase}
            onDone={() => onOpenChange(false)}
          />
        ) : null}
      </DialogContent>
    </Dialog>
  );
}

function LeaveDayForm({
  workerId,
  leaveCase,
  onDone,
}: {
  workerId: string;
  leaveCase: LeaveCaseRow;
  onDone: () => void;
}) {
  const t = useT();
  const invalidate = useWorkerInvalidation(workerId, "leave");
  const form = useForm<LeaveDayFormValues>({
    resolver: zodResolver(leaveDayFormSchema) as Resolver<LeaveDayFormValues>,
    defaultValues: emptyDay(),
  });
  const { control, handleSubmit } = form;

  const { mutateAsync, isPending } = useApiMutation<
    { id: string; countsAgainstEntitlement: boolean },
    LeaveDayFormValues,
    unknown,
    LeaveDayFormValues
  >({
    form,
    resourceName: "Day",
    scope: { id: `worker-leave-day-record:${leaveCase.id}` },
    mutationFn: (values) =>
      recordLeaveDay({
        caseId: leaveCase.id,
        usedOn: values.usedOn,
        hours: String(values.hours),
        notes: values.notes ?? undefined,
      }),
    onSuccess: async () => {
      await invalidate();
      onDone();
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
          <FormControl>
            <AutoCompleteDateField<LeaveDayFormValues>
              control={control}
              name="usedOn"
              label={t("Day")}
              placeholder={t("Day the leave was taken")}
              description={t("The calendar day these hours belong to.")}
              required
            />
          </FormControl>
          <FormControl>
            <NumberField<LeaveDayFormValues>
              control={control}
              name="hours"
              label={t("Hours")}
              placeholder={t("e.g. 8")}
              description={t("Hours of leave taken that day, more than zero and up to 24.")}
              required
            />
          </FormControl>
          <FormControl cols="full">
            <Alert variant={leaveCase.fmlaDesignated ? "info" : "warning"}>
              <AlertDescription>
                {leaveCase.fmlaDesignated
                  ? t("This case is designated as FMLA, so the day draws the entitlement down.")
                  : t(
                      "This case is not designated as FMLA. The day is recorded but draws nothing down.",
                    )}
              </AlertDescription>
            </Alert>
          </FormControl>
          <FormControl cols="full">
            <TextareaField<LeaveDayFormValues>
              control={control}
              name="notes"
              label={t("Notes")}
              placeholder={t("e.g. Half day for a medical appointment")}
              description={t("Kept on the entry and shown in the case's list of days.")}
              maxLength={2000}
            />
          </FormControl>
        </FormGroup>
        <DialogFooter>
          <Button type="button" size="sm" variant="outline" onClick={onDone}>
            {t("Cancel")}
          </Button>
          <Button type="submit" size="sm" isLoading={isPending}>
            {t("Record day")}
          </Button>
        </DialogFooter>
      </Form>
    </FormProvider>
  );
}
