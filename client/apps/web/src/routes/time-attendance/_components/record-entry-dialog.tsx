import { AutoCompleteDateTimeField } from "@/components/fields/date-field/datetime-field";
import { InputField } from "@/components/fields/input-field";
import { NumberField } from "@/components/fields/number-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { recordTimeEntry } from "@/lib/graphql/timesheet";
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
import { getTodayDate } from "@trenova/shared/lib/date";
import { formatHours } from "@trenova/shared/lib/timesheet";
import {
  recordTimeEntryFormSchema,
  type RecordTimeEntryFormValues,
} from "@trenova/shared/types/timesheet";
import { useEffect } from "react";
import { FormProvider, useForm, useWatch, type Resolver } from "react-hook-form";
import { toast } from "sonner";

export type RecordEntryDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  workerId: string;
  entryId?: string;
  onRecorded: () => void;
};

/**
 * Recording or correcting a period by hand. The reason is required rather than
 * optional: a wage record altered by somebody with no reason recorded is not a
 * record anybody can defend.
 */
export function RecordEntryDialog({
  open,
  onOpenChange,
  workerId,
  entryId,
  onRecorded,
}: RecordEntryDialogProps) {
  const form = useForm<RecordTimeEntryFormValues>({
    resolver: zodResolver(recordTimeEntryFormSchema) as Resolver<RecordTimeEntryFormValues>,
    defaultValues: {
      clockedInAt: getTodayDate(),
      clockedOutAt: getTodayDate() + 8 * 3600,
      breakMinutes: 0,
      note: null,
      reason: "",
    },
  });
  const { control, handleSubmit, reset } = form;

  useEffect(() => {
    if (!open) return;
    reset({
      clockedInAt: getTodayDate(),
      clockedOutAt: getTodayDate() + 8 * 3600,
      breakMinutes: 0,
      note: null,
      reason: "",
    });
  }, [open, reset]);

  const clockedInAt = useWatch({ control, name: "clockedInAt" });
  const clockedOutAt = useWatch({ control, name: "clockedOutAt" });
  const breakMinutes = useWatch({ control, name: "breakMinutes" });
  const paidMinutes = Math.max(
    0,
    Math.floor(((clockedOutAt ?? 0) - (clockedInAt ?? 0)) / 60) - (breakMinutes ?? 0),
  );

  const { mutateAsync, isPending } = useApiMutation<
    { id: string },
    RecordTimeEntryFormValues,
    unknown,
    RecordTimeEntryFormValues
  >({
    form,
    resourceName: "Entry",
    mutationFn: (values) =>
      recordTimeEntry({
        id: entryId ?? undefined,
        workerId,
        clockedInAt: values.clockedInAt,
        clockedOutAt: values.clockedOutAt,
        breakMinutes: values.breakMinutes,
        note: values.note ?? undefined,
        reason: values.reason,
      }),
    onSuccess: () => {
      toast.success(entryId ? "Entry corrected" : "Hours recorded");
      onRecorded();
      onOpenChange(false);
    },
  });

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-xl">
        <DialogHeader>
          <DialogTitle>{entryId ? "Correct the entry" : "Record hours"}</DialogTitle>
          <DialogDescription>
            A period recorded by hand is marked as such, and the reason is kept with it. Hours can
            only be changed while the week is still open.
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
                <AutoCompleteDateTimeField<RecordTimeEntryFormValues>
                  control={control}
                  name="clockedInAt"
                  label="Started"
                  rules={{ required: true }}
                />
              </FormControl>
              <FormControl>
                <AutoCompleteDateTimeField<RecordTimeEntryFormValues>
                  control={control}
                  name="clockedOutAt"
                  label="Finished"
                  rules={{ required: true }}
                />
              </FormControl>
              <FormControl>
                <NumberField<RecordTimeEntryFormValues>
                  control={control}
                  name="breakMinutes"
                  label="Unpaid break (minutes)"
                  description={`${formatHours(paidMinutes)} would be paid`}
                />
              </FormControl>
              <FormControl>
                <InputField<RecordTimeEntryFormValues>
                  control={control}
                  name="reason"
                  label="Reason"
                  placeholder="e.g. Missed clock-out"
                  rules={{ required: true }}
                  description="Kept with the record."
                />
              </FormControl>
              <FormControl cols="full">
                <TextareaField<RecordTimeEntryFormValues>
                  control={control}
                  name="note"
                  label="Note"
                  placeholder="What this period covers"
                />
              </FormControl>
            </FormGroup>

            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
                Cancel
              </Button>
              <Button type="submit" isLoading={isPending} disabled={!workerId}>
                {entryId ? "Save" : "Record"}
              </Button>
            </DialogFooter>
          </Form>
        </FormProvider>
      </DialogContent>
    </Dialog>
  );
}
