import { useT } from "@trenova/shared/i18n/use-t";
import { TrainingCourseAutocompleteField } from "@/components/autocomplete-fields";
import { AutoCompleteDateField } from "@/components/fields/date-field/date-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { useSelectOption } from "@/hooks/use-select-option";
import type { SelectOption } from "@/lib/graphql/select-options";
import { selectOptionMetaNumber } from "@/lib/select-option-meta";
import { assignWorkerTraining, type WorkerTrainingRecordRow } from "@/lib/graphql/worker-training";
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
import {
  assignTrainingFormSchema,
  type AssignTrainingFormValues,
} from "@trenova/shared/types/worker-training";
import { useCallback, useEffect } from "react";
import { FormProvider, useForm, useWatch, type Resolver } from "react-hook-form";
import { toast } from "sonner";
import { useTrainingInvalidation } from "./use-training-invalidation";

const DAY = 86_400;

export type AssignTrainingDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  workerId: string;
  /** Pre-select a course, e.g. from a slot card. */
  courseId?: string | null;
  /** Courses that already have an open assignment and cannot be picked. */
  openCourseIds: ReadonlySet<string>;
};

export function AssignTrainingDialog({
  open,
  onOpenChange,
  workerId,
  courseId,
  openCourseIds,
}: AssignTrainingDialogProps) {
  const t = useT();

  const invalidate = useTrainingInvalidation(workerId);
  const form = useForm<AssignTrainingFormValues>({
    resolver: zodResolver(assignTrainingFormSchema) as Resolver<AssignTrainingFormValues>,
    defaultValues: { courseId: courseId ?? "", dueAt: null, notes: null },
  });
  const { control, handleSubmit, reset, setValue } = form;

  useEffect(() => {
    if (open) reset({ courseId: courseId ?? "", dueAt: null, notes: null });
  }, [open, courseId, reset]);

  const selectedId = useWatch({ control, name: "courseId" });
  const { option: selected } = useSelectOption("TRAINING_COURSE", selectedId);
  const dueDaysAfterAssignment = selected
    ? (selectOptionMetaNumber(selected, "dueDaysAfterAssignment") ?? 0)
    : 0;

  // Courses the worker already has open are hidden rather than rejected on
  // submit: assigning one twice is the mistake this dialog is most likely to
  // invite.
  const hideAssignedCourses = useCallback(
    (option: SelectOption) => !openCourseIds.has(option.id),
    [openCourseIds],
  );

  useEffect(() => {
    if (!selected) return;
    setValue(
      "dueAt",
      dueDaysAfterAssignment > 0 ? getTodayDate() + dueDaysAfterAssignment * DAY : null,
    );
  }, [selected, dueDaysAfterAssignment, setValue]);

  const { mutateAsync, isPending } = useApiMutation<
    WorkerTrainingRecordRow,
    AssignTrainingFormValues,
    unknown,
    AssignTrainingFormValues
  >({
    form,
    resourceName: "Training",
    mutationFn: (values) =>
      assignWorkerTraining({
        workerId,
        courseId: values.courseId,
        dueAt: values.dueAt ?? undefined,
        notes: values.notes ?? undefined,
      }),
    onSuccess: (record) => {
      toast.success(`${record.course?.name ?? "Course"} assigned`, {
        description: record.dueAt
          ? "The driver will see it in Dash with the due date."
          : "The driver will see it in Dash.",
      });
      void invalidate();
      onOpenChange(false);
    },
  });

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{t("Assign a course")}</DialogTitle>
          <DialogDescription>
            {t(
              "The course appears on the driver's Training list in Dash. Self-serve courses complete when the driver acknowledges them; scored and in-person courses wait for you to record the result.",
            )}
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
            <FormGroup className="pb-2" cols={2}>
              <FormControl cols="full">
                <TrainingCourseAutocompleteField<AssignTrainingFormValues>
                  control={control}
                  name="courseId"
                  label={t("Course")}
                  placeholder={t("Select a course")}
                  rules={{ required: true }}
                  filterOption={hideAssignedCourses}
                  noResultsMessage={t("Every active course is already assigned to this worker.")}
                  description={
                    selected?.description || t("Courses with an open assignment are not listed.")
                  }
                />
              </FormControl>
              <FormControl cols="full">
                <AutoCompleteDateField<AssignTrainingFormValues>
                  control={control}
                  name="dueAt"
                  label={t("Due")}
                  placeholder={t("No due date")}
                  description={
                    dueDaysAfterAssignment > 0
                      ? t("Defaults to {0} days from today.", String(dueDaysAfterAssignment))
                      : t("Leave empty for no deadline; a date in the past is rejected.")
                  }
                />
              </FormControl>
              <FormControl cols="full">
                <TextareaField<AssignTrainingFormValues>
                  control={control}
                  name="notes"
                  label={t("Notes")}
                  placeholder={t("Anything the driver or the recorder should know")}
                  maxLength={1000}
                  description={t("Saved on the training record.")}
                />
              </FormControl>
            </FormGroup>
            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
                {t("Cancel")}
              </Button>
              <Button type="submit" isLoading={isPending} loadingText={t("Assigning...")}>
                {t("Assign")}
              </Button>
            </DialogFooter>
          </Form>
        </FormProvider>
      </DialogContent>
    </Dialog>
  );
}
