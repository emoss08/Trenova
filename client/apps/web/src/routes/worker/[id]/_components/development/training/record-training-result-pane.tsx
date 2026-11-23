import { TrainingCourseAutocompleteField } from "@/components/autocomplete-fields";
import { FileDropField, type AttachedFile } from "@/components/documents/file-drop-field";
import { AutoCompleteDateField } from "@/components/fields/date-field/date-field";
import { InputField } from "@/components/fields/input-field";
import { TextareaField } from "@/components/fields/textarea-field";
import {
  WorkPane,
  WorkPaneBody,
  WorkPaneCancel,
  WorkPaneDirtyTracker,
  WorkPaneFooter,
} from "@/components/work-pane";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { useSingleFlight } from "@/hooks/use-single-flight";
import { useSelectOption } from "@/hooks/use-select-option";
import { selectOptionMetaString } from "@/lib/select-option-meta";
import {
  attachWorkerTrainingDocument,
  completeWorkerTraining,
  type WorkerTrainingRecordRow,
} from "@/lib/graphql/worker-training";
import { zodResolver } from "@hookform/resolvers/zod";
import { Button } from "@trenova/shared/components/ui/button";
import { Form, FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import { toast } from "@trenova/shared/components/ui/toast";
import { useT } from "@trenova/shared/i18n/use-t";
import { getTodayDate } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import {
  completeTrainingFormSchema,
  type CompleteTrainingFormValues,
} from "@trenova/shared/types/worker-training";
import { useEffect, useRef, useState } from "react";
import { Controller, FormProvider, useForm, useWatch, type Resolver } from "react-hook-form";
import { projectScore, scoreLine } from "./training-score";
import { useTrainingInvalidation } from "./use-training-invalidation";

type ResultValues = CompleteTrainingFormValues & { certificate: AttachedFile | null };

function resultDefaults(courseId: string): ResultValues {
  return {
    courseId,
    completedAt: getTodayDate(),
    score: null,
    notes: null,
    requiresScore: false,
    certificate: null,
  };
}

type Saved = { record: WorkerTrainingRecordRow; certificateFailed: boolean };

export type RecordTrainingResultPaneProps = {
  open: boolean;
  /** The open assignment being closed; its course is locked. */
  record: WorkerTrainingRecordRow | null;
  /** The course to file a result for when there is no open assignment (a retake). */
  presetCourseId: string | null;
  workerId: string;
  onClose: () => void;
};

/**
 * Record a result beside the courses table: the date, the score against the course's passing
 * mark (said live, before saving), the certificate and notes. A fail keeps the assignment
 * open for a retake.
 */
export function RecordTrainingResultPane({
  open,
  record,
  presetCourseId,
  workerId,
  onClose,
}: RecordTrainingResultPaneProps) {
  const t = useT();
  const invalidate = useTrainingInvalidation(workerId);
  const lockedCourseId = record?.courseId ?? presetCourseId ?? "";

  const form = useForm<ResultValues>({
    resolver: zodResolver(completeTrainingFormSchema) as unknown as Resolver<ResultValues>,
    defaultValues: resultDefaults(lockedCourseId),
  });
  const { control, handleSubmit, reset, setValue, getValues } = form;

  const openedFor = useRef<string | null>(null);
  useEffect(() => {
    const session = open ? `${record?.id ?? ""}:${lockedCourseId}` : null;
    if (session !== null && openedFor.current !== session) reset(resultDefaults(lockedCourseId));
    openedFor.current = session;
  }, [open, record, lockedCourseId, reset]);

  const courseId = useWatch({ control, name: "courseId" });
  const { option: course } = useSelectOption("TRAINING_COURSE", record ? null : courseId);
  const passing =
    record?.course?.passingScore ??
    (course ? selectOptionMetaString(course, "passingScore") || null : null);
  const scored = passing != null;
  useEffect(() => {
    setValue("requiresScore", scored);
  }, [scored, setValue]);

  const score = useWatch({ control, name: "score" });
  const projection = projectScore(score, passing);
  const failing = projection.kind === "fail";

  // The title names the course while the pane folds away after a save.
  const [courseName, setCourseName] = useState<string | null>(null);
  const [uploading, setUploading] = useState(false);
  const name = record?.course?.name ?? course?.label ?? null;
  if (open && name !== courseName) setCourseName(name);

  const save = useApiMutation<Saved, ResultValues, unknown, ResultValues>({
    form,
    resourceName: "Training result",
    scope: { id: `worker-training-result:${workerId}` },
    mutationFn: async (values) => {
      const saved = await completeWorkerTraining({
        id: record?.id ?? undefined,
        workerId: record ? undefined : workerId,
        courseId: record ? undefined : values.courseId,
        completedAt: values.completedAt,
        score: values.score ?? undefined,
        notes: values.notes ?? undefined,
        version: record?.version,
      });
      const certificate = getValues("certificate");
      if (!certificate) return { record: saved, certificateFailed: false };
      try {
        const attached = await attachWorkerTrainingDocument({
          id: saved.id,
          documentId: certificate.id,
        });
        return { record: attached, certificateFailed: false };
      } catch {
        return { record: saved, certificateFailed: true };
      }
    },
    onSuccess: ({ certificateFailed }) => {
      if (certificateFailed) {
        toast.warning(t("The result is recorded, but the certificate was not attached"), {
          description: t("It is in the worker's documents; attach it from there."),
        });
      }
      void invalidate();
      onClose();
    },
  });
  const submit = useSingleFlight((values: ResultValues) => save.mutateAsync(values));

  return (
    <WorkPane
      open={open}
      onOpenChange={(next) => {
        if (!next) onClose();
      }}
      title={courseName ? t("Record a result: {0}", courseName) : t("Record a result")}
      description={t("Any expiry is counted from the completion date.")}
      contentKey={record?.id ?? lockedCourseId}
    >
      <FormProvider {...form}>
        <WorkPaneDirtyTracker form={form} />
        <Form
          onSubmit={(event) => {
            event.preventDefault();
            event.stopPropagation();
            void handleSubmit((submitted) => submit.run(submitted))(event);
          }}
        >
          <WorkPaneBody>
            <FormGroup cols={2}>
              <FormControl cols="full">
                <TrainingCourseAutocompleteField<ResultValues>
                  control={control}
                  name="courseId"
                  label={t("Course")}
                  placeholder={t("Select a course")}
                  required
                  disabled={Boolean(record)}
                  description={
                    record
                      ? t("Locked to the assignment being closed.")
                      : t("The course this result is for.")
                  }
                />
              </FormControl>
              <FormControl>
                <AutoCompleteDateField<ResultValues>
                  control={control}
                  name="completedAt"
                  label={t("Completed")}
                  placeholder={t("Today")}
                  required
                />
              </FormControl>
              <FormControl>
                <InputField<ResultValues>
                  control={control}
                  name="score"
                  label={t("Score")}
                  placeholder={t("e.g. 92")}
                  sideText="%"
                  inputMode="decimal"
                  required={scored}
                />
              </FormControl>
              <FormControl cols="full">
                <p
                  data-testid="score-line"
                  className={cn(
                    "-mt-1 text-xs",
                    projection.kind === "pass" && "font-medium text-success-foreground",
                    (projection.kind === "fail" || projection.kind === "invalid") &&
                      "font-medium text-danger-foreground",
                    (projection.kind === "waiting" || projection.kind === "unscored") &&
                      "text-muted-foreground",
                  )}
                >
                  {scoreLine(projection, t)}
                </p>
              </FormControl>
              <FormControl cols="full">
                <TextareaField<ResultValues>
                  control={control}
                  name="notes"
                  label={t("Notes")}
                  placeholder={t("Instructor, session, certificate number")}
                  maxLength={1000}
                />
              </FormControl>
              <FormControl cols="full">
                <Controller
                  control={control}
                  name="certificate"
                  render={({ field }) => (
                    <FileDropField
                      label={t("Certificate")}
                      note={t("optional")}
                      value={field.value}
                      onChange={field.onChange}
                      resourceId={workerId}
                      resourceType="worker"
                      prompt={t("Drop the certificate here")}
                      onUploadingChange={setUploading}
                    />
                  )}
                />
              </FormControl>
            </FormGroup>
          </WorkPaneBody>
          <WorkPaneFooter>
            <WorkPaneCancel />
            <Button
              type="submit"
              size="sm"
              variant={failing ? "destructive" : "default"}
              isLoading={submit.pending}
              disabled={uploading}
            >
              {failing ? t("Record fail") : t("Record result")}
            </Button>
          </WorkPaneFooter>
        </Form>
      </FormProvider>
    </WorkPane>
  );
}
