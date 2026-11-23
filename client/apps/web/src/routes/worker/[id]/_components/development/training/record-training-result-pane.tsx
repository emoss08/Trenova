import { TrainingCourseAutocompleteField } from "@/components/autocomplete-fields";
import { FileDropField, type AttachedFile } from "@/components/documents/file-drop-field";
import { FieldNote } from "@/components/fields/field-components";
import { AutoCompleteDateField } from "@/components/fields/date-field/date-field";
import { InputField } from "@/components/fields/input-field";
import { TextareaField } from "@/components/fields/textarea-field";
import {
  PaneFormLoading,
  WorkPane,
  WorkPaneBody,
  WorkPaneCancel,
  WorkPaneDirtyTracker,
  WorkPaneFooter,
} from "@/components/work-pane";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { useSelectOption } from "@/hooks/use-select-option";
import { useSingleFlight } from "@/hooks/use-single-flight";
import type { SelectOption } from "@/lib/graphql/select-options";
import {
  attachWorkerTrainingDocument,
  completeWorkerTraining,
  type WorkerTrainingRecordRow,
} from "@/lib/graphql/worker-training";
import { selectOptionMetaString } from "@/lib/select-option-meta";
import { zodResolver } from "@hookform/resolvers/zod";
import { Button } from "@trenova/shared/components/ui/button";
import { Form, FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import { toast } from "@trenova/shared/components/ui/toast";
import { useT } from "@trenova/shared/i18n/use-t";
import { getTodayDate } from "@trenova/shared/lib/date";
import {
  completeTrainingFormSchema,
  type CompleteTrainingFormValues,
} from "@trenova/shared/types/worker-training";
import { useState } from "react";
import {
  Controller,
  FormProvider,
  useForm,
  useWatch,
  type Control,
  type Resolver,
} from "react-hook-form";
import { useWorkerInvalidation } from "../../use-worker-invalidation";
import { projectScore, scoreLine, scoreTone } from "./training-score";

type ResultValues = CompleteTrainingFormValues & { certificate: AttachedFile | null };

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

/** A course's passing mark, from the assignment or the course picked; null when unscored. */
function passingOf(
  record: WorkerTrainingRecordRow | null,
  course: SelectOption | null,
): string | null {
  return (
    record?.course?.passingScore ??
    (course ? selectOptionMetaString(course, "passingScore") || null : null)
  );
}

/**
 * Record a result beside the courses table: the date, the score against the course's passing
 * mark (said live, before saving), the certificate and notes. A fail keeps the assignment
 * open for a retake.
 *
 * Mounted fresh at each opening (`lazyWorkPane`), so what it was opened on is read once at
 * mount: a refetch of the assignment while it is open never touches the form, and the title
 * stays while the pane folds away after a save.
 */
export function RecordTrainingResultPane({
  open,
  record: requested,
  presetCourseId: requestedCourseId,
  workerId,
  onClose,
}: RecordTrainingResultPaneProps) {
  const t = useT();
  const [record] = useState(requested);
  const [presetCourseId] = useState(requestedCourseId);
  const preset = useSelectOption("TRAINING_COURSE", record ? null : presetCourseId);
  // The course picked in the form, once one is; until then the preset.
  const [picked, setPicked] = useState<SelectOption | null | undefined>(undefined);
  const course = picked === undefined ? preset.option : picked;
  const name = record?.course?.name ?? course?.label ?? null;

  return (
    <WorkPane
      open={open}
      onOpenChange={(next) => {
        if (!next) onClose();
      }}
      title={name ? t("Record result: {0}", name) : t("Record result")}
      description={t("Any expiry is counted from the completion date.")}
    >
      {preset.isLoading ? (
        <PaneFormLoading />
      ) : (
        <ResultForm
          workerId={workerId}
          record={record}
          courseId={record?.courseId ?? presetCourseId ?? ""}
          passing={passingOf(record, course)}
          onPick={setPicked}
          onClose={onClose}
        />
      )}
    </WorkPane>
  );
}

function ResultForm({
  workerId,
  record,
  courseId,
  passing,
  onPick,
  onClose,
}: {
  workerId: string;
  record: WorkerTrainingRecordRow | null;
  courseId: string;
  passing: string | null;
  onPick: (course: SelectOption | null) => void;
  onClose: () => void;
}) {
  const t = useT();
  const invalidate = useWorkerInvalidation(workerId, "training");
  const [uploading, setUploading] = useState(false);
  const form = useForm<ResultValues>({
    resolver: zodResolver(completeTrainingFormSchema) as unknown as Resolver<ResultValues>,
    defaultValues: {
      courseId,
      completedAt: getTodayDate(),
      score: null,
      notes: null,
      requiresScore: passing != null,
      certificate: null,
    },
  });
  const { control, handleSubmit, setValue, getValues } = form;
  const scored = passing != null;

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
      // The schema strips the certificate from the submitted values; it is read off the form.
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
    onSuccess: async ({ certificateFailed }) => {
      if (certificateFailed) {
        toast.warning(t("The result is recorded, but the certificate was not attached"), {
          description: t("It is in the worker's documents; attach it from there."),
        });
      }
      await invalidate();
      onClose();
    },
  });
  const submit = useSingleFlight((values: ResultValues) => save.mutateAsync(values));

  return (
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
                onOptionChange={(option) => {
                  // The course decides whether a score is needed and the mark it is read against.
                  onPick(option);
                  setValue("requiresScore", passingOf(null, option) != null);
                }}
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
              <ScoreField control={control} passing={passing} required={scored} />
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
          <SubmitButton
            control={control}
            passing={passing}
            pending={submit.pending}
            uploading={uploading}
          />
        </WorkPaneFooter>
      </Form>
    </FormProvider>
  );
}

/** The score, with the line under it saying where the typed figure lands against the mark. */
function ScoreField({
  control,
  passing,
  required,
}: {
  control: Control<ResultValues>;
  passing: string | null;
  required: boolean;
}) {
  const t = useT();
  const score = useWatch({ control, name: "score" });
  const projection = projectScore(score, passing);
  return (
    <InputField<ResultValues>
      control={control}
      name="score"
      label={t("Score")}
      placeholder={t("e.g. 92")}
      sideText="%"
      inputMode="decimal"
      required={required}
      description={
        <FieldNote live tone={scoreTone(projection)}>
          {scoreLine(projection, t)}
        </FieldNote>
      }
    />
  );
}

/** Says a fail before it is saved: the button turns destructive while the score would fail. */
function SubmitButton({
  control,
  passing,
  pending,
  uploading,
}: {
  control: Control<ResultValues>;
  passing: string | null;
  pending: boolean;
  uploading: boolean;
}) {
  const t = useT();
  const score = useWatch({ control, name: "score" });
  const failing = projectScore(score, passing).kind === "fail";
  return (
    <Button
      type="submit"
      size="sm"
      variant={failing ? "destructive" : "default"}
      isLoading={pending}
      disabled={uploading}
    >
      {failing ? t("Record fail") : t("Record result")}
    </Button>
  );
}
