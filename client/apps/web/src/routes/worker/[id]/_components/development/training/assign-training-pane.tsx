import { TrainingCourseAutocompleteField } from "@/components/autocomplete-fields";
import { AutoCompleteDateField } from "@/components/fields/date-field/date-field";
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
import type { SelectOption } from "@/lib/graphql/select-options";
import { selectOptionMetaNumber } from "@/lib/select-option-meta";
import { assignWorkerTraining, type WorkerTrainingRecordRow } from "@/lib/graphql/worker-training";
import { zodResolver } from "@hookform/resolvers/zod";
import { Button } from "@trenova/shared/components/ui/button";
import { Form, FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import { useT } from "@trenova/shared/i18n/use-t";
import { getTodayDate } from "@trenova/shared/lib/date";
import {
  assignTrainingFormSchema,
  type AssignTrainingFormValues,
} from "@trenova/shared/types/worker-training";
import { useEffect, useRef } from "react";
import { FormProvider, useForm, useWatch, type Resolver } from "react-hook-form";
import { useTrainingInvalidation } from "./use-training-invalidation";

const DAY = 86_400;

export type AssignTrainingPaneProps = {
  open: boolean;
  /** A course to preselect, e.g. a required course nobody has assigned. */
  presetCourseId: string | null;
  workerId: string;
  /** Courses with an open assignment; they are not offered again. */
  openCourseIds: ReadonlySet<string>;
  onClose: () => void;
};

/**
 * Assign a course beside the courses table. The driver sees it in Dash with its due date, and
 * it joins the table.
 */
export function AssignTrainingPane({
  open,
  presetCourseId,
  workerId,
  openCourseIds,
  onClose,
}: AssignTrainingPaneProps) {
  const t = useT();
  const invalidate = useTrainingInvalidation(workerId);
  const form = useForm<AssignTrainingFormValues>({
    resolver: zodResolver(assignTrainingFormSchema) as Resolver<AssignTrainingFormValues>,
    defaultValues: { courseId: presetCourseId ?? "", dueAt: null, notes: null },
  });
  const { control, handleSubmit, reset, setValue } = form;

  const openedFor = useRef<string | null>(null);
  const filledFor = useRef<string | null>(null);
  useEffect(() => {
    const session = open ? (presetCourseId ?? "") : null;
    if (session !== null && openedFor.current !== session) {
      reset({ courseId: presetCourseId ?? "", dueAt: null, notes: null });
      filledFor.current = null;
    }
    openedFor.current = session;
  }, [open, presetCourseId, reset]);

  const selectedId = useWatch({ control, name: "courseId" });
  const { option: selected } = useSelectOption("TRAINING_COURSE", selectedId);
  const dueDays = selected ? (selectOptionMetaNumber(selected, "dueDaysAfterAssignment") ?? 0) : 0;

  // A course's own due window fills the date when the course is picked.
  useEffect(() => {
    if (!selected || filledFor.current === selected.id) return;
    filledFor.current = selected.id;
    setValue("dueAt", dueDays > 0 ? getTodayDate() + dueDays * DAY : null);
  }, [selected, dueDays, setValue]);

  const hideAssigned = (option: SelectOption) => !openCourseIds.has(option.id);

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
    onSuccess: () => {
      void invalidate();
      onClose();
    },
  });
  const submit = useSingleFlight((values: AssignTrainingFormValues) => save.mutateAsync(values));

  return (
    <WorkPane
      open={open}
      onOpenChange={(next) => {
        if (!next) onClose();
      }}
      title={t("Assign a course")}
      description={t("The driver sees it in Dash with the due date, and it joins this table.")}
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
            <FormGroup cols={1}>
              <FormControl>
                <TrainingCourseAutocompleteField<AssignTrainingFormValues>
                  control={control}
                  name="courseId"
                  label={t("Course")}
                  placeholder={t("Select a course")}
                  required
                  filterOption={hideAssigned}
                  noResultsMessage={t("Every active course is already assigned to this worker.")}
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
                      ? t(
                          "Defaults to {0, plural, one {# day} other {# days}} from today.",
                          dueDays,
                        )
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
              {t("Assign")}
            </Button>
          </WorkPaneFooter>
        </Form>
      </FormProvider>
    </WorkPane>
  );
}
