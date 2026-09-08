import { AutoCompleteDateField } from "@/components/fields/date-field/date-field";
import { InputField } from "@/components/fields/input-field";
import { SelectField } from "@/components/fields/select-field";
import { SwitchField } from "@/components/fields/switch-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { giveWorkerRecognition, type WorkerRecognitionRow } from "@/lib/graphql/worker-safety";
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
import {
  RECOGNITION_KIND_LABELS,
  recognitionFormSchema,
  recognitionKindSchema,
  type RecognitionFormValues,
} from "@trenova/shared/types/worker-safety";
import { zodResolver } from "@hookform/resolvers/zod";
import { useEffect } from "react";
import { FormProvider, useForm, useWatch, type Resolver } from "react-hook-form";
import { toast } from "sonner";
import { useSafetyInvalidation } from "./use-safety-invalidation";

const KIND_OPTIONS = recognitionKindSchema.options.map((value) => ({
  value,
  label: RECOGNITION_KIND_LABELS[value],
}));

export type RecognitionDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  workerId: string;
};

export function RecognitionDialog({ open, onOpenChange, workerId }: RecognitionDialogProps) {
  const invalidate = useSafetyInvalidation(workerId);
  const form = useForm<RecognitionFormValues>({
    resolver: zodResolver(recognitionFormSchema) as Resolver<RecognitionFormValues>,
    defaultValues: {
      kind: "SafetyMilestone",
      title: "",
      message: null,
      occurredAt: getTodayDate(),
      visibleToWorker: true,
    },
  });
  const { control, handleSubmit, reset } = form;

  useEffect(() => {
    if (!open) return;
    reset({
      kind: "SafetyMilestone",
      title: "",
      message: null,
      occurredAt: getTodayDate(),
      visibleToWorker: true,
    });
  }, [open, reset]);

  const visible = useWatch({ control, name: "visibleToWorker" });

  const { mutateAsync, isPending } = useApiMutation<
    WorkerRecognitionRow,
    RecognitionFormValues,
    unknown,
    RecognitionFormValues
  >({
    form,
    resourceName: "Recognition",
    mutationFn: (values) =>
      giveWorkerRecognition({
        workerId,
        kind: values.kind,
        title: values.title,
        message: values.message ?? undefined,
        occurredAt: values.occurredAt,
        visibleToWorker: values.visibleToWorker,
      }),
    onSuccess: (saved) => {
      toast.success("Recognition recorded", {
        description: saved.visibleToWorker
          ? "The driver will see it in Dash."
          : "Kept internal — the driver will not see it.",
      });
      void invalidate();
      onOpenChange(false);
    },
  });

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Record recognition</DialogTitle>
          <DialogDescription>
            The other half of the safety record. Visible recognition reaches the driver in Dash;
            internal notes stay with the office.
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
                <SelectField<RecognitionFormValues>
                  control={control}
                  name="kind"
                  label="Kind"
                  options={KIND_OPTIONS}
                  placeholder="Pick a kind"
                  description="What the recognition is for; it is shown with the title on the safety record."
                  rules={{ required: true }}
                />
              </FormControl>
              <FormControl>
                <AutoCompleteDateField<RecognitionFormValues>
                  control={control}
                  name="occurredAt"
                  placeholder="MM/DD/YYYY"
                  description="The day the achievement happened; it defaults to today."
                  label="When"
                  rules={{ required: true }}
                />
              </FormControl>
              <FormControl cols="full">
                <InputField<RecognitionFormValues>
                  control={control}
                  name="title"
                  label="Title"
                  placeholder="e.g. One year accident-free"
                  description="The headline on the record; it is also the subject of the driver's notification when shared."
                  rules={{ required: true }}
                  maxLength={120}
                />
              </FormControl>
              <FormControl cols="full">
                <TextareaField<RecognitionFormValues>
                  control={control}
                  name="message"
                  label="Message"
                  placeholder="What you want the driver to read"
                  description="Optional detail that goes out with the notification when the recognition is shared."
                  maxLength={2000}
                />
              </FormControl>
              <FormControl cols="full">
                <SwitchField<RecognitionFormValues>
                  control={control}
                  name="visibleToWorker"
                  label="Share with the driver"
                  description={
                    visible
                      ? "Sends a notification and shows on their Dash profile."
                      : "Kept on the office record only."
                  }
                  position="left"
                  outlined
                />
              </FormControl>
            </FormGroup>
            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
                Cancel
              </Button>
              <Button type="submit" isLoading={isPending} loadingText="Saving...">
                Record recognition
              </Button>
            </DialogFooter>
          </Form>
        </FormProvider>
      </DialogContent>
    </Dialog>
  );
}
