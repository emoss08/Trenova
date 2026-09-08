import { TextareaField } from "@/components/fields/textarea-field";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { waiveWorkerTraining, type WorkerTrainingRecordRow } from "@/lib/graphql/worker-training";
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
  waiveTrainingFormSchema,
  type WaiveTrainingFormValues,
} from "@trenova/shared/types/worker-training";
import { zodResolver } from "@hookform/resolvers/zod";
import { useEffect } from "react";
import { FormProvider, useForm, type Resolver } from "react-hook-form";
import { toast } from "sonner";
import { useTrainingInvalidation } from "./use-training-invalidation";

export type WaiveTrainingDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  workerId: string;
  record?: WorkerTrainingRecordRow | null;
};

export function WaiveTrainingDialog({
  open,
  onOpenChange,
  workerId,
  record,
}: WaiveTrainingDialogProps) {
  const invalidate = useTrainingInvalidation(workerId);
  const form = useForm<WaiveTrainingFormValues>({
    resolver: zodResolver(waiveTrainingFormSchema) as Resolver<WaiveTrainingFormValues>,
    defaultValues: { reason: "" },
  });
  const { control, handleSubmit, reset } = form;

  useEffect(() => {
    if (open) reset({ reason: "" });
  }, [open, reset]);

  const { mutateAsync, isPending } = useApiMutation<
    WorkerTrainingRecordRow,
    WaiveTrainingFormValues,
    unknown,
    WaiveTrainingFormValues
  >({
    form,
    resourceName: "Training",
    mutationFn: (values) => {
      if (!record) throw new Error("Nothing to waive");
      return waiveWorkerTraining({ id: record.id, reason: values.reason, version: record.version });
    },
    onSuccess: (saved) => {
      toast.success(`${saved.course?.name ?? "Course"} waived`, {
        description: "It counts as satisfied; the reason stays on the record.",
      });
      void invalidate();
      onOpenChange(false);
    },
  });

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Waive {record?.course?.name ?? "this course"}</DialogTitle>
          <DialogDescription>
            Use this when the worker already meets the requirement another way — prior experience,
            an equivalent certificate, a grandfathered rule. The waiver satisfies the matrix and is
            kept in the audit log with your reason.
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
            <FormGroup className="pb-2" cols={1}>
              <FormControl cols="full">
                <TextareaField<WaiveTrainingFormValues>
                  control={control}
                  name="reason"
                  label="Reason"
                  placeholder="e.g. Completed at previous carrier; certificate on file"
                  rules={{ required: true }}
                  maxLength={255}
                  description="Kept on the record and in the audit log."
                />
              </FormControl>
            </FormGroup>
            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
                Cancel
              </Button>
              <Button type="submit" isLoading={isPending} loadingText="Waiving...">
                Waive course
              </Button>
            </DialogFooter>
          </Form>
        </FormProvider>
      </DialogContent>
    </Dialog>
  );
}
