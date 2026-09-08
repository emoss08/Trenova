import { ShiftTemplateAutocompleteField } from "@/components/autocomplete-fields";
import { AutoCompleteDateField } from "@/components/fields/date-field/date-field";
import { NumberField } from "@/components/fields/number-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { assignWorkerShift } from "@/lib/graphql/scheduling";
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
  DAY_LABELS,
  dayMaskToDays,
  describeShiftPattern,
  formatShiftWindow,
} from "@trenova/shared/lib/scheduling";
import { cn } from "@trenova/shared/lib/utils";
import type { SelectOption as GraphQLSelectOption } from "@/lib/graphql/select-options";
import {
  assignShiftFormSchema,
  type AssignShiftFormValues,
} from "@trenova/shared/types/scheduling";
import { useEffect, useState } from "react";
import { FormProvider, useForm, type Resolver } from "react-hook-form";
import { toast } from "sonner";

export type AssignShiftDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  workerId: string;
  onAssigned: () => void;
};

type ShiftPreview = {
  daysOfWeek: string;
  startMinute: number;
  durationMinutes: number;
  cycleWeeks: number;
  color: string;
};

function metaString(meta: Record<string, unknown>, key: string): string {
  const value = meta[key];
  return typeof value === "string" ? value : "";
}

function metaNumber(meta: Record<string, unknown>, key: string, fallback: number): number {
  const value = Number(meta[key]);
  return Number.isFinite(value) ? value : fallback;
}

function previewOf(option: GraphQLSelectOption | null): ShiftPreview | null {
  const meta = option?.meta as Record<string, unknown> | null | undefined;
  if (!meta) return null;
  return {
    daysOfWeek: metaString(meta, "daysOfWeek"),
    startMinute: metaNumber(meta, "startMinute", 0),
    durationMinutes: metaNumber(meta, "durationMinutes", 0),
    cycleWeeks: metaNumber(meta, "cycleWeeks", 1),
    color: metaString(meta, "color"),
  };
}

function defaults(workerId: string): AssignShiftFormValues {
  return {
    workerId,
    shiftTemplateId: "",
    effectiveFrom: getTodayDate(),
    cycleOffsetWeeks: 0,
    notes: null,
  };
}

/**
 * Putting a worker on a pattern. The shift is picked from the same autocomplete
 * the rest of the product uses, and the pattern it carries is previewed before
 * anybody commits — a wrong shift on a rota is a week of somebody's life.
 */
export function AssignShiftDialog({
  open,
  onOpenChange,
  workerId,
  onAssigned,
}: AssignShiftDialogProps) {
  const [preview, setPreview] = useState<ShiftPreview | null>(null);
  const form = useForm<AssignShiftFormValues>({
    resolver: zodResolver(assignShiftFormSchema) as Resolver<AssignShiftFormValues>,
    defaultValues: defaults(workerId),
  });
  const { control, handleSubmit, reset } = form;

  useEffect(() => {
    if (!open) return;
    reset(defaults(workerId));
  }, [open, workerId, reset]);

  const { mutateAsync, isPending } = useApiMutation<
    { id: string },
    AssignShiftFormValues,
    unknown,
    AssignShiftFormValues
  >({
    form,
    resourceName: "Assignment",
    mutationFn: (values) =>
      assignWorkerShift({
        workerId: values.workerId,
        shiftTemplateId: values.shiftTemplateId,
        effectiveFrom: values.effectiveFrom,
        cycleOffsetWeeks: values.cycleOffsetWeeks,
        notes: values.notes ?? undefined,
      }),
    onSuccess: () => {
      toast.success("Put on the shift");
      onAssigned();
      onOpenChange(false);
    },
  });

  const rotates = (preview?.cycleWeeks ?? 1) > 1;

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-xl">
        <DialogHeader>
          <DialogTitle>Put on a shift</DialogTitle>
          <DialogDescription>
            The shift in force is ended the day before this one starts, so the worker is never on
            two patterns at once.
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
                <ShiftTemplateAutocompleteField<AssignShiftFormValues>
                  control={control}
                  name="shiftTemplateId"
                  label="Shift"
                  placeholder="Search shifts"
                  rules={{ required: true }}
                  onOptionChange={(option) => setPreview(previewOf(option))}
                  description="Only shifts in force can be assigned."
                />
              </FormControl>

              {preview ? (
                <FormControl cols="full">
                  <div className="bg-muted/30 flex flex-col gap-2 rounded-lg border p-3 text-xs">
                    <div className="flex items-center justify-between">
                      <span className="font-medium">
                        {describeShiftPattern(preview.daysOfWeek, preview.cycleWeeks)}
                      </span>
                      <span className="text-muted-foreground tabular-nums">
                        {formatShiftWindow(preview.startMinute, preview.durationMinutes)}
                      </span>
                    </div>
                    <div className="flex items-center gap-1">
                      {DAY_LABELS.map((label, index) => {
                        const on = dayMaskToDays(preview.daysOfWeek).includes(index);
                        return (
                          <span
                            key={label}
                            className={cn(
                              "grid h-6 flex-1 place-items-center rounded-md text-[10px] font-medium",
                              on
                                ? "bg-primary text-primary-foreground"
                                : "bg-muted text-muted-foreground",
                            )}
                          >
                            {label[0]}
                          </span>
                        );
                      })}
                    </div>
                  </div>
                </FormControl>
              ) : null}

              <FormControl>
                <AutoCompleteDateField<AssignShiftFormValues>
                  control={control}
                  name="effectiveFrom"
                  label="Effective from"
                  placeholder="Today"
                  rules={{ required: true }}
                  description="The first day the worker is on this pattern."
                />
              </FormControl>
              {rotates ? (
                <FormControl>
                  <NumberField<AssignShiftFormValues>
                    control={control}
                    name="cycleOffsetWeeks"
                    label="Rotation offset (weeks)"
                    placeholder="0"
                    description={`0 to ${(preview?.cycleWeeks ?? 1) - 1}. Two workers on the same shift at different offsets alternate.`}
                  />
                </FormControl>
              ) : null}
              <FormControl cols="full">
                <TextareaField<AssignShiftFormValues>
                  control={control}
                  name="notes"
                  label="Notes"
                  placeholder="Why they are on this shift"
                  description="Kept on the assignment record."
                />
              </FormControl>
            </FormGroup>

            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
                Cancel
              </Button>
              <Button type="submit" isLoading={isPending}>
                Assign
              </Button>
            </DialogFooter>
          </Form>
        </FormProvider>
      </DialogContent>
    </Dialog>
  );
}
