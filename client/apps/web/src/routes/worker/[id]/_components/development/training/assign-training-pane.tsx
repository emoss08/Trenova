import { TrainingCourseAutocompleteField } from "@/components/autocomplete-fields";
import { AutoCompleteDateField } from "@/components/fields/date-field/date-field";
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
import { assignWorkerTraining, type WorkerTrainingRecordRow } from "@/lib/graphql/worker-training";
import { selectOptionMetaNumber } from "@/lib/select-option-meta";
import { SECONDS_IN_DAY } from "@/lib/time-units";
import { zodResolver } from "@hookform/resolvers/zod";
import { useQuery } from "@tanstack/react-query";
import { Button } from "@trenova/shared/components/ui/button";
import { Form, FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import { useT } from "@trenova/shared/i18n/use-t";
import { getTodayDate } from "@trenova/shared/lib/date";
import {
  assignTrainingFormSchema,
  type AssignTrainingFormValues,
} from "@trenova/shared/types/worker-training";
import { useState } from "react";
import { FormProvider, useForm, type Resolver } from "react-hook-form";
import { useWorkerInvalidation } from "../../use-worker-invalidation";
import { trainingSummaryQuery } from "../development-reads";
import { openCourseIdsOf } from "./training-model";

export type AssignTrainingPaneProps = {
  open: boolean;
  /** A course to preselect, e.g. a required course nobody has assigned. */
  presetCourseId: string | null;
  workerId: string;
  onClose: () => void;
};

/** The days a course gives a driver to finish it once assigned; 0 for no deadline. */
function dueDaysOf(course: SelectOption | null): number {
  return course ? (selectOptionMetaNumber(course, "dueDaysAfterAssignment") ?? 0) : 0;
}

/** The due date a course's own window sets, counted from today. */
function dueAtFor(course: SelectOption | null): number | null {
  const days = dueDaysOf(course);
  return days > 0 ? getTodayDate() + days * SECONDS_IN_DAY : null;
}

/**
 * Assign a course beside the courses table. The driver sees it in Dash with its due date, and
 * it joins the table. A preset course is read before the form mounts, so its due window is the
 * form's default rather than a value written in afterwards.
 */
export function AssignTrainingPane({
  open,
  presetCourseId,
  workerId,
  onClose,
}: AssignTrainingPaneProps) {
  const t = useT();
  const preset = useSelectOption("TRAINING_COURSE", presetCourseId);

  return (
    <WorkPane
      open={open}
      onOpenChange={(next) => {
        if (!next) onClose();
      }}
      title={t("Assign course")}
      description={t("The driver sees it in Dash with the due date, and it joins this table.")}
    >
      {preset.isLoading ? (
        <PaneFormLoading fields={3} />
      ) : (
        <AssignTrainingForm
          workerId={workerId}
          preset={presetCourseId ? preset.option : null}
          presetCourseId={presetCourseId}
          onClose={onClose}
        />
      )}
    </WorkPane>
  );
}

function AssignTrainingForm({
  workerId,
  preset,
  presetCourseId,
  onClose,
}: {
  workerId: string;
  preset: SelectOption | null;
  presetCourseId: string | null;
  onClose: () => void;
}) {
  const t = useT();
  const invalidate = useWorkerInvalidation(workerId, "training");
  const [selected, setSelected] = useState<SelectOption | null>(preset);
  const form = useForm<AssignTrainingFormValues>({
    resolver: zodResolver(assignTrainingFormSchema) as Resolver<AssignTrainingFormValues>,
    defaultValues: { courseId: presetCourseId ?? "", dueAt: dueAtFor(preset), notes: null },
  });
  const { control, handleSubmit, setValue } = form;

  // Courses with an open assignment are not offered again; the view has read the summary.
  const summaryQuery = useQuery(trainingSummaryQuery(workerId));
  const openCourseIds = openCourseIdsOf(summaryQuery.data?.items ?? []);
  const dueDays = dueDaysOf(selected);

  const save = useApiMutation<
    WorkerTrainingRecordRow,
    AssignTrainingFormValues,
    unknown,
    AssignTrainingFormValues
  >({
    form,
    resourceName: "Training",
    scope: { id: `worker-training-assign:${workerId}` },
    mutationFn: (values) =>
      assignWorkerTraining({
        workerId,
        courseId: values.courseId,
        dueAt: values.dueAt ?? undefined,
        notes: values.notes ?? undefined,
      }),
    onSuccess: async () => {
      await invalidate();
      onClose();
    },
  });
  const submit = useSingleFlight((values: AssignTrainingFormValues) => save.mutateAsync(values));

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
          <FormGroup cols={1}>
            <FormControl>
              <TrainingCourseAutocompleteField<AssignTrainingFormValues>
                control={control}
                name="courseId"
                label={t("Course")}
                placeholder={t("Select a course")}
                required
                filterOption={(option) => !openCourseIds.has(option.id)}
                noResultsMessage={t("Every active course is already assigned to this worker.")}
                onOptionChange={(option) => {
                  // Picking a course sets the due date its own window gives.
                  setSelected(option);
                  setValue("dueAt", dueAtFor(option), { shouldDirty: true });
                }}
                description={
                  selected?.description || t("Courses with an open assignment are not listed.")
                }
              />
            </FormControl>
            <FormControl>
              <AutoCompleteDateField<AssignTrainingFormValues>
                control={control}
                name="dueAt"
                label={t("Due")}
                placeholder={t("No due date")}
                description={
                  dueDays > 0
                    ? t("Defaults to {0, plural, one {# day} other {# days}} from today.", dueDays)
                    : t("Leave empty for no deadline; a date in the past is rejected.")
                }
              />
            </FormControl>
            <FormControl>
              <TextareaField<AssignTrainingFormValues>
                control={control}
                name="notes"
                label={t("Notes")}
                placeholder={t("Anything the driver or the recorder should know")}
                maxLength={1000}
              />
            </FormControl>
          </FormGroup>
        </WorkPaneBody>
        <WorkPaneFooter>
          <WorkPaneCancel />
          <Button type="submit" size="sm" isLoading={submit.pending}>
            {t("Assign course")}
          </Button>
        </WorkPaneFooter>
      </Form>
    </FormProvider>
  );
}
