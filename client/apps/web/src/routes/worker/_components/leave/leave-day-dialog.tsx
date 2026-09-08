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
import { useEffect } from "react";
import { FormProvider, useForm, type Resolver } from "react-hook-form";
import { toast } from "sonner";
import { useLeaveInvalidation } from "./use-leave-invalidation";

export type LeaveDayDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  workerId: string;
  leaveCase: LeaveCaseRow | null;
};

function emptyDay(): LeaveDayFormValues {
  return { usedOn: getTodayDate(), hours: 8, notes: null };
}

export function LeaveDayDialog({ open, onOpenChange, workerId, leaveCase }: LeaveDayDialogProps) {
  const invalidate = useLeaveInvalidation(workerId);
  const form = useForm<LeaveDayFormValues>({
    resolver: zodResolver(leaveDayFormSchema) as Resolver<LeaveDayFormValues>,
    defaultValues: emptyDay(),
  });
  const { control, handleSubmit, reset } = form;

  useEffect(() => {
    if (!open) return;
    reset(emptyDay());
  }, [open, reset]);

  const { mutateAsync, isPending } = useApiMutation<
    { id: string; countsAgainstEntitlement: boolean },
    LeaveDayFormValues,
    unknown,
    LeaveDayFormValues
  >({
    form,
    resourceName: "Day",
    mutationFn: (values) => {
      if (!leaveCase) throw new Error("No case selected");
      return recordLeaveDay({
        caseId: leaveCase.id,
        usedOn: values.usedOn,
        hours: String(values.hours),
        notes: values.notes ?? undefined,
      });
    },
    onSuccess: (saved) => {
      toast.success("Day recorded", {
        description: saved.countsAgainstEntitlement
          ? "Drawn against the FMLA entitlement."
          : "Recorded, but not drawn against the entitlement — the case is not designated.",
      });
      void invalidate();
      onOpenChange(false);
    },
  });

  const designated = leaveCase?.fmlaDesignated ?? false;

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>Record a day of leave</DialogTitle>
          <DialogDescription>
            Hours rather than days, so intermittent leave can be taken in the increment the
            organisation actually uses.
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
                <AutoCompleteDateField<LeaveDayFormValues>
                  control={control}
                  name="usedOn"
                  label="Day"
                  placeholder="Day the leave was taken"
                  description="The calendar day these hours belong to."
                  rules={{ required: true }}
                />
              </FormControl>
              <FormControl>
                <NumberField<LeaveDayFormValues>
                  control={control}
                  name="hours"
                  label="Hours"
                  placeholder="e.g. 8"
                  description="Hours of leave taken that day, more than zero and up to 24."
                  rules={{ required: true }}
                />
              </FormControl>
              <FormControl cols="full">
                <Alert variant={designated ? "info" : "warning"}>
                  <AlertDescription>
                    {designated
                      ? "This case is designated as FMLA, so the day draws the entitlement down."
                      : "This case is not designated as FMLA. The day is recorded but draws nothing down."}
                  </AlertDescription>
                </Alert>
              </FormControl>
              <FormControl cols="full">
                <TextareaField<LeaveDayFormValues>
                  control={control}
                  name="notes"
                  label="Notes"
                  placeholder="e.g. Half day for a medical appointment"
                  description="Kept on the entry and shown in the case's list of days."
                  maxLength={2000}
                />
              </FormControl>
            </FormGroup>
            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
                Cancel
              </Button>
              <Button type="submit" isLoading={isPending}>
                Record
              </Button>
            </DialogFooter>
          </Form>
        </FormProvider>
      </DialogContent>
    </Dialog>
  );
}
