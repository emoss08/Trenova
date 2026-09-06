import { AutoCompleteDateField } from "@/components/fields/date-field/date-field";
import { SelectField } from "@/components/fields/select-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { useApiMutation } from "@/hooks/use-api-mutation";
import {
  assignWorkerTraining,
  fetchActiveTrainingCourses,
  TRAINING_COURSES_KEY,
  type TrainingCourseRow,
  type WorkerTrainingRecordRow,
} from "@/lib/graphql/worker-training";
import { zodResolver } from "@hookform/resolvers/zod";
import { useQuery } from "@tanstack/react-query";
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
  TRAINING_DELIVERY_LABELS,
  type AssignTrainingFormValues,
  type TrainingDelivery,
} from "@trenova/shared/types/worker-training";
import { useEffect, useMemo } from "react";
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
  const invalidate = useTrainingInvalidation(workerId);
  const coursesQuery = useQuery({
    queryKey: [TRAINING_COURSES_KEY],
    queryFn: ({ signal }) => fetchActiveTrainingCourses({ signal }),
    enabled: open,
    staleTime: 5 * 60 * 1000,
  });

  const form = useForm<AssignTrainingFormValues>({
    resolver: zodResolver(assignTrainingFormSchema) as Resolver<AssignTrainingFormValues>,
    defaultValues: { courseId: courseId ?? "", dueAt: null, notes: null },
  });
  const { control, handleSubmit, reset, setValue } = form;

  useEffect(() => {
    if (open) reset({ courseId: courseId ?? "", dueAt: null, notes: null });
  }, [open, courseId, reset]);

  const selectedId = useWatch({ control, name: "courseId" });
  const courses = useMemo(
    () => (coursesQuery.data ?? []).filter((course) => !openCourseIds.has(course.id)),
    [coursesQuery.data, openCourseIds],
  );
  const selected = courses.find((course) => course.id === selectedId);

  useEffect(() => {
    if (!selected) return;
    setValue(
      "dueAt",
      selected.dueDaysAfterAssignment > 0
        ? getTodayDate() + selected.dueDaysAfterAssignment * DAY
        : null,
    );
  }, [selected, setValue]);

  const options = useMemo(
    () =>
      courses.map((course) => ({
        value: course.id,
        label: course.name,
        description: describeCourse(course),
      })),
    [courses],
  );

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
          <DialogTitle>Assign a course</DialogTitle>
          <DialogDescription>
            The course appears on the driver&apos;s Training list in Dash. Self-serve courses
            complete when the driver acknowledges them; scored and in-person courses wait for you to
            record the result.
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
                <SelectField<AssignTrainingFormValues>
                  control={control}
                  name="courseId"
                  label="Course"
                  placeholder="Select a course"
                  options={options}
                  rules={{ required: true }}
                  description={
                    selected?.description ??
                    (courses.length === 0 && !coursesQuery.isLoading
                      ? "Every active course is already assigned to this worker."
                      : undefined)
                  }
                />
              </FormControl>
              <FormControl cols="full">
                <AutoCompleteDateField<AssignTrainingFormValues>
                  control={control}
                  name="dueAt"
                  label="Due"
                  placeholder="No due date"
                  description={
                    selected && selected.dueDaysAfterAssignment > 0
                      ? `Defaults to ${selected.dueDaysAfterAssignment} days from today.`
                      : undefined
                  }
                />
              </FormControl>
              <FormControl cols="full">
                <TextareaField<AssignTrainingFormValues>
                  control={control}
                  name="notes"
                  label="Notes"
                  placeholder="Anything the driver or the recorder should know"
                  maxLength={1000}
                />
              </FormControl>
            </FormGroup>
            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
                Cancel
              </Button>
              <Button
                type="submit"
                isLoading={isPending}
                loadingText="Assigning..."
                disabled={courses.length === 0}
              >
                Assign
              </Button>
            </DialogFooter>
          </Form>
        </FormProvider>
      </DialogContent>
    </Dialog>
  );
}

function describeCourse(course: TrainingCourseRow): string {
  const parts = [TRAINING_DELIVERY_LABELS[course.delivery as TrainingDelivery]];
  if (course.durationMinutes > 0) parts.push(`${course.durationMinutes} min`);
  if (course.passingScore) parts.push(`pass ≥ ${Number(course.passingScore).toFixed(0)}%`);
  if (course.validityMonths) parts.push(`valid ${course.validityMonths} mo`);
  return parts.join(" · ");
}
