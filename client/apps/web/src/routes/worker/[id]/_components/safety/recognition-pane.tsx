import { AutoCompleteDateField } from "@/components/fields/date-field/date-field";
import { InputField } from "@/components/fields/input-field";
import { SelectField } from "@/components/fields/select-field";
import { SwitchField } from "@/components/fields/switch-field";
import { TextareaField } from "@/components/fields/textarea-field";
import {
  WorkPane,
  WorkPaneBody,
  WorkPaneCancel,
  WorkPaneDirtyTracker,
  WorkPaneFooter,
} from "@/components/work-pane";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { giveWorkerRecognition, type WorkerRecognitionRow } from "@/lib/graphql/worker-safety";
import { zodResolver } from "@hookform/resolvers/zod";
import { Button } from "@trenova/shared/components/ui/button";
import { Form, FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import { sourceLabels } from "@trenova/shared/i18n/labels";
import { useT } from "@trenova/shared/i18n/use-t";
import { getTodayDate } from "@trenova/shared/lib/date";
import {
  RECOGNITION_KIND_LABELS,
  recognitionFormSchema,
  recognitionKindSchema,
  type RecognitionFormValues,
} from "@trenova/shared/types/worker-safety";
import { FormProvider, useForm, useWatch, type Resolver } from "react-hook-form";
import { useWorkerInvalidation } from "../use-worker-invalidation";

const RECOGNITION_KIND_OPTIONS = recognitionKindSchema.options.map((value) => ({
  value,
  label: sourceLabels(RECOGNITION_KIND_LABELS)[value],
}));

function emptyRecognition(): RecognitionFormValues {
  return {
    kind: "SafetyMilestone",
    title: "",
    message: null,
    occurredAt: getTodayDate(),
    visibleToWorker: true,
  };
}

export type RecognitionPaneProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  workerId: string;
  workerName: string;
  onSaved: () => void;
};

/** Records recognition on the safety record, shared with the driver in Dash or kept internal. */
export function RecognitionPane({
  open,
  onOpenChange,
  workerId,
  workerName,
  onSaved,
}: RecognitionPaneProps) {
  const t = useT();
  const invalidate = useWorkerInvalidation(workerId, "safety");
  const form = useForm<RecognitionFormValues>({
    resolver: zodResolver(recognitionFormSchema) as Resolver<RecognitionFormValues>,
    defaultValues: emptyRecognition(),
  });
  const { control, handleSubmit } = form;

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
    onSuccess: async () => {
      await invalidate();
      onSaved();
    },
  });

  return (
    <WorkPane
      open={open}
      onOpenChange={onOpenChange}
      title={t("Add recognition")}
      description={t("Kept on the safety record. Shared recognition shows in Dash.")}
    >
      <WorkPaneDirtyTracker form={form} />
      <FormProvider {...form}>
        <Form
          onSubmit={(submitEvent) => {
            submitEvent.preventDefault();
            submitEvent.stopPropagation();
            void handleSubmit((values) => mutateAsync(values))(submitEvent);
          }}
        >
          <WorkPaneBody className="text-sm">
            <FormGroup cols={1}>
              <FormControl cols="full">
                <SelectField<RecognitionFormValues>
                  control={control}
                  name="kind"
                  label={t("Kind")}
                  options={RECOGNITION_KIND_OPTIONS}
                  placeholder={t("Pick a kind")}
                  required
                />
              </FormControl>
              <FormControl cols="full">
                <InputField<RecognitionFormValues>
                  control={control}
                  name="title"
                  label={t("Title")}
                  placeholder={t("e.g. One year accident-free")}
                  required
                  maxLength={120}
                />
              </FormControl>
              <FormControl cols="full">
                <AutoCompleteDateField<RecognitionFormValues>
                  control={control}
                  name="occurredAt"
                  label={t("When")}
                  placeholder={t("MM/DD/YYYY")}
                  description={t("The day the achievement happened; it defaults to today.")}
                  required
                />
              </FormControl>
              <FormControl cols="full">
                <TextareaField<RecognitionFormValues>
                  control={control}
                  name="message"
                  label={t("Message")}
                  placeholder={t("What you want the driver to read")}
                  maxLength={2000}
                />
              </FormControl>
              <FormControl cols="full">
                <SwitchField<RecognitionFormValues>
                  control={control}
                  name="visibleToWorker"
                  label={t("Share with the driver")}
                  description={
                    visible
                      ? t("{0} sees it in Dash with your message.", workerName)
                      : t("Kept on the office record only.")
                  }
                  position="left"
                  outlined
                />
              </FormControl>
            </FormGroup>
          </WorkPaneBody>
          <WorkPaneFooter>
            <WorkPaneCancel />
            <Button type="submit" size="sm" isLoading={isPending}>
              {t("Add recognition")}
            </Button>
          </WorkPaneFooter>
        </Form>
      </FormProvider>
    </WorkPane>
  );
}
