import { useT } from "@trenova/shared/i18n/use-t";
import { AutoCompleteDateField } from "@/components/fields/date-field/date-field";
import { InputField } from "@/components/fields/input-field";
import { SelectField } from "@/components/fields/select-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { useApiMutation } from "@/hooks/use-api-mutation";
import {
  completeWorkerTraining,
  fetchActiveTrainingCourses,
  TRAINING_COURSES_KEY,
  type WorkerTrainingRecordRow,
} from "@/lib/graphql/worker-training";
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
  completeTrainingFormSchema,
  type CompleteTrainingFormValues,
} from "@trenova/shared/types/worker-training";
import { zodResolver } from "@hookform/resolvers/zod";
import { useQuery } from "@tanstack/react-query";
import { useEffect, useMemo } from "react";
import { FormProvider, useForm, useWatch, type Resolver } from "react-hook-form";
import { toast } from "sonner";
import { useTrainingInvalidation } from "./use-training-invalidation";

export type CompleteTrainingDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  workerId: string;
  /** The open record being closed, when there is one. */
  record?: WorkerTrainingRecordRow | null;
  /** Pre-select the course when filing a completion without an open record. */
  courseId?: string | null;
};

export function CompleteTrainingDialog({
  open,
  onOpenChange,
  workerId,
  record,
  courseId,
}: CompleteTrainingDialogProps) {
  const t = useT();

  const invalidate = useTrainingInvalidation(workerId);
  const coursesQuery = useQuery({
    queryKey: [TRAINING_COURSES_KEY],
    queryFn: ({ signal }) => fetchActiveTrainingCourses({ signal }),
    enabled: open,
    staleTime: 5 * 60 * 1000,
  });
  const courses = useMemo(() => coursesQuery.data ?? [], [coursesQuery.data]);
  const lockedCourseId = record?.courseId ?? courseId ?? "";

  const form = useForm<CompleteTrainingFormValues>({
    resolver: zodResolver(completeTrainingFormSchema) as Resolver<CompleteTrainingFormValues>,
    defaultValues: {
      courseId: lockedCourseId,
      completedAt: getTodayDate(),
      score: null,
      notes: null,
      requiresScore: false,
    },
  });
  const { control, handleSubmit, reset, setValue } = form;

  useEffect(() => {
    if (!open) return;
    reset({
      courseId: lockedCourseId,
      completedAt: getTodayDate(),
      score: null,
      notes: null,
      requiresScore: false,
    });
  }, [open, lockedCourseId, reset]);

  const selectedId = useWatch({ control, name: "courseId" });
  const selected = record?.course ?? courses.find((course) => course.id === selectedId);
  const scored = Boolean(selected?.passingScore);

  useEffect(() => {
    setValue("requiresScore", scored);
  }, [scored, setValue]);

  const score = useWatch({ control, name: "score" });
  const projected = useMemo(() => {
    if (!selected?.passingScore || !score) return null;
    return Number(score) >= Number(selected.passingScore) ? "pass" : "fail";
  }, [score, selected]);

  const { mutateAsync, isPending } = useApiMutation<
    WorkerTrainingRecordRow,
    CompleteTrainingFormValues,
    unknown,
    CompleteTrainingFormValues
  >({
    form,
    resourceName: "Training result",
    mutationFn: (values) =>
      completeWorkerTraining({
        id: record?.id ?? undefined,
        workerId: record ? undefined : workerId,
        courseId: record ? undefined : values.courseId,
        completedAt: values.completedAt,
        score: values.score ?? undefined,
        notes: values.notes ?? undefined,
        version: record?.version,
      }),
    onSuccess: (saved) => {
      const passed = saved.status === "Completed";
      toast[passed ? "success" : "warning"](
        passed
          ? `${saved.course?.name ?? "Course"} completed`
          : `${saved.course?.name ?? "Course"} failed`,
        {
          description: passed
            ? saved.expiresAt
              ? "The certification is on file with its expiry date."
              : "The course is on file."
            : "Assign the course again when the worker is ready to retake it.",
        },
      );
      void invalidate();
      onOpenChange(false);
    },
  });

  const courseOptions = useMemo(
    () => courses.map((course) => ({ value: course.id, label: course.name })),
    [courses],
  );

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{t("Record a result")}</DialogTitle>
          <DialogDescription>
            {scored
              ? `Scored course — ${Number(selected?.passingScore).toFixed(0)}% or better passes. A fail closes the assignment; assign it again for a retake.`
              : "Marks the course complete on the date given. Recurring courses get their expiry from the course's validity."}
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
                <SelectField<CompleteTrainingFormValues>
                  control={control}
                  name="courseId"
                  label={t("Course")}
                  placeholder={t("Select a course")}
                  options={
                    record?.course
                      ? [{ value: record.course.id, label: record.course.name }]
                      : courseOptions
                  }
                  rules={{ required: true }}
                  isReadOnly={Boolean(record)}
                  description={
                    record
                      ? "Locked to the assignment being closed."
                      : "The course this result is for."
                  }
                />
              </FormControl>
              <FormControl>
                <AutoCompleteDateField<CompleteTrainingFormValues>
                  control={control}
                  name="completedAt"
                  label={t("Completed")}
                  placeholder={t("Today")}
                  rules={{ required: true }}
                  description={t("The day the course was finished; any expiry is counted from it.")}
                />
              </FormControl>
              <FormControl>
                <InputField<CompleteTrainingFormValues>
                  control={control}
                  name="score"
                  label={t("Score")}
                  placeholder={scored ? "e.g. 92" : "Optional"}
                  sideText="%"
                  rules={{ required: scored }}
                  description={
                    projected === "pass"
                      ? "Passes."
                      : projected === "fail"
                        ? "Below the passing mark — this will be recorded as a fail."
                        : scored
                          ? "Compared with the passing mark to decide pass or fail."
                          : "Optional for a course without a passing mark."
                  }
                />
              </FormControl>
              <FormControl cols="full">
                <TextareaField<CompleteTrainingFormValues>
                  control={control}
                  name="notes"
                  label={t("Notes")}
                  placeholder={t("Instructor, session, certificate number")}
                  maxLength={1000}
                  description={t("Kept on the record with the result.")}
                />
              </FormControl>
            </FormGroup>
            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
                {t("Cancel")}
              </Button>
              <Button
                type="submit"
                variant={projected === "fail" ? "destructive" : "default"}
                isLoading={isPending}
                loadingText={t("Saving...")}
              >
                {projected === "fail" ? "Record fail" : "Record result"}
              </Button>
            </DialogFooter>
          </Form>
        </FormProvider>
      </DialogContent>
    </Dialog>
  );
}
