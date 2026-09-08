import { AutoCompleteDateField } from "@/components/fields/date-field/date-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { useApiMutation } from "@/hooks/use-api-mutation";
import {
  bulkAssignTraining,
  fetchActiveTrainingCourses,
  type BulkAssignTrainingResult,
} from "@/lib/graphql/worker-training";
import { zodResolver } from "@hookform/resolvers/zod";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { Checkbox } from "@trenova/shared/components/ui/checkbox";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@trenova/shared/components/ui/dialog";
import { Form, FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { TRAINING_CATEGORY_LABELS } from "@trenova/shared/types/worker-training";
import type { WorkerRow } from "@/lib/graphql/worker-table";
import { InfoIcon } from "lucide-react";
import { useEffect, useMemo } from "react";
import { FormProvider, useForm, useWatch, type Resolver } from "react-hook-form";
import { toast } from "sonner";
import { z } from "zod";

const formSchema = z.object({
  courseIds: z.array(z.string()).min(1, "Choose at least one course"),
  dueAt: z.number().int().nullable(),
  notes: z.string().nullable(),
});

type FormValues = z.infer<typeof formSchema>;

export type BulkAssignTrainingDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  /** The rows selected on the roster. */
  workers: WorkerRow[];
};

/**
 * Rolls a course out to everyone currently selected on the roster. Pairing it
 * with the roster's saved views is the point: filter to "Training overdue",
 * select all, assign the renewal.
 */
export function BulkAssignTrainingDialog({
  open,
  onOpenChange,
  workers,
}: BulkAssignTrainingDialogProps) {
  const queryClient = useQueryClient();
  const { data: courses = [], isLoading } = useQuery({
    queryKey: ["training-courses", "active"],
    queryFn: ({ signal }) => fetchActiveTrainingCourses({ signal }),
    enabled: open,
  });

  const form = useForm<FormValues>({
    resolver: zodResolver(formSchema) as Resolver<FormValues>,
    defaultValues: { courseIds: [], dueAt: null, notes: null },
  });
  const { control, handleSubmit, reset, setValue } = form;
  const selectedCourseIds = useWatch({ control, name: "courseIds" });

  useEffect(() => {
    if (open) reset({ courseIds: [], dueAt: null, notes: null });
  }, [open, reset]);

  const workerIds = useMemo(() => workers.map((worker) => worker.id), [workers]);

  const { mutateAsync, isPending } = useApiMutation<
    BulkAssignTrainingResult,
    FormValues,
    unknown,
    FormValues
  >({
    form,
    resourceName: "Training assignment",
    mutationFn: (values) =>
      bulkAssignTraining({
        workerIds,
        courseIds: values.courseIds,
        dueAt: values.dueAt ?? undefined,
        notes: values.notes ?? undefined,
      }),
    onSuccess: (result) => {
      toast.success(describeBulkAssign(result));
      void queryClient.invalidateQueries({ queryKey: ["worker-list"] });
      void queryClient.invalidateQueries({ queryKey: ["worker-training-summary"] });
      onOpenChange(false);
    },
  });

  const toggleCourse = (courseId: string, checked: boolean) => {
    setValue(
      "courseIds",
      checked
        ? [...selectedCourseIds, courseId]
        : selectedCourseIds.filter((id) => id !== courseId),
      { shouldValidate: true, shouldDirty: true },
    );
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-xl">
        <DialogHeader>
          <DialogTitle>Assign training</DialogTitle>
          <DialogDescription>
            Opens the courses you choose for {workers.length} selected{" "}
            {workers.length === 1 ? "worker" : "workers"}. Anyone who already has a course open
            keeps the assignment they have.
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
              <FormControl cols="full">
                <div className="flex flex-col gap-1.5">
                  <span className="text-sm font-medium">Courses</span>
                  <ScrollArea className="max-h-56 rounded-md border p-2">
                    {isLoading ? (
                      <p className="text-muted-foreground p-2 text-sm">Loading courses...</p>
                    ) : courses.length === 0 ? (
                      <p className="text-muted-foreground p-2 text-sm">
                        No active courses to assign.
                      </p>
                    ) : (
                      <div className="flex flex-col gap-1">
                        {courses.map((course) => (
                          <label
                            key={course.id}
                            className="hover:bg-muted/50 flex cursor-pointer items-center gap-2 rounded-md p-1.5"
                          >
                            <Checkbox
                              checked={selectedCourseIds.includes(course.id)}
                              onCheckedChange={(checked) =>
                                toggleCourse(course.id, checked === true)
                              }
                              aria-label={course.name}
                            />
                            <span className="flex-1 truncate text-sm">{course.name}</span>
                            <Badge variant="outline" className="shrink-0">
                              {TRAINING_CATEGORY_LABELS[
                                course.category as keyof typeof TRAINING_CATEGORY_LABELS
                              ] ?? course.category}
                            </Badge>
                          </label>
                        ))}
                      </div>
                    )}
                  </ScrollArea>
                </div>
              </FormControl>
              <FormControl cols="full">
                <AutoCompleteDateField<FormValues>
                  control={control}
                  name="dueAt"
                  label="Due date"
                  placeholder="Each course's own default"
                  description="Leave empty to use the due-days each course already sets."
                />
              </FormControl>
              <FormControl cols="full">
                <TextareaField<FormValues>
                  control={control}
                  name="notes"
                  label="Notes"
                  placeholder="e.g. Annual hazmat refresher"
                  description="Copied onto every training record this run creates."
                  maxLength={4000}
                />
              </FormControl>
              {workers.length > 50 ? (
                <FormControl cols="full">
                  <Alert className="py-2">
                    <InfoIcon className="size-4" />
                    <AlertDescription>
                      This opens up to {workers.length * Math.max(selectedCourseIds.length, 1)}{" "}
                      assignments and notifies every driver affected.
                    </AlertDescription>
                  </Alert>
                </FormControl>
              ) : null}
            </FormGroup>
            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
                Cancel
              </Button>
              <Button type="submit" isLoading={isPending} loadingText="Assigning...">
                Assign to {workers.length}
              </Button>
            </DialogFooter>
          </Form>
        </FormProvider>
      </DialogContent>
    </Dialog>
  );
}

/**
 * Reads the run back in the office's terms. Skipped assignments are reported
 * separately from failures, because "already enrolled" is not a problem.
 */
export function describeBulkAssign(result: BulkAssignTrainingResult): string {
  const parts = [`${result.assignedCount} assigned`];
  if (result.skippedCount > 0) parts.push(`${result.skippedCount} already open`);
  if (result.failedCount > 0) parts.push(`${result.failedCount} failed`);
  return parts.join(", ");
}
