import { ShiftTemplateAutocompleteField } from "@/components/autocomplete-fields";
import { AutoCompleteDateField } from "@/components/fields/date-field/date-field";
import { NumberField } from "@/components/fields/number-field";
import { TextareaField } from "@/components/fields/textarea-field";
import {
  WorkPane,
  WorkPaneBody,
  WorkPaneCancel,
  WorkPaneDirtyTracker,
  WorkPaneFooter,
} from "@/components/work-pane";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { assignWorkerShift } from "@/lib/graphql/scheduling";
import type { SelectOption as GraphQLSelectOption } from "@/lib/graphql/select-options";
import { zodResolver } from "@hookform/resolvers/zod";
import { Button } from "@trenova/shared/components/ui/button";
import { Form, FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import { useT } from "@trenova/shared/i18n/use-t";
import { getTodayDate } from "@trenova/shared/lib/date";
import { describeShiftPattern, formatShiftWindow } from "@trenova/shared/lib/scheduling";
import { cn } from "@trenova/shared/lib/utils";
import {
  assignShiftFormSchema,
  type AssignShiftFormValues,
} from "@trenova/shared/types/scheduling";
import { useEffect, useState } from "react";
import { FormProvider, useForm, type Resolver } from "react-hook-form";
import { longWeekday, shiftClashes, type StatedPreference } from "./schedule-model";
import { WeekPattern } from "./week-pattern";

export type AssignShiftPaneProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  workerId: string;
  /** True when the worker is on a shift already, so this moves them. */
  moving: boolean;
  preferences: readonly StatedPreference[];
  onAssigned: () => void;
};

type ShiftPreview = {
  daysOfWeek: string;
  startMinute: number;
  durationMinutes: number;
  cycleWeeks: number;
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
 * Putting a worker on a pattern, beside the schedule it changes. The picked shift's week is drawn
 * before anybody commits, with the days the worker said they can't work marked where it lands.
 */
export function AssignShiftPane({
  open,
  onOpenChange,
  workerId,
  moving,
  preferences,
  onAssigned,
}: AssignShiftPaneProps) {
  const t = useT();
  const [preview, setPreview] = useState<ShiftPreview | null>(null);
  const form = useForm<AssignShiftFormValues>({
    resolver: zodResolver(assignShiftFormSchema) as Resolver<AssignShiftFormValues>,
    defaultValues: defaults(workerId),
  });
  const { control, handleSubmit, reset } = form;

  const [wasOpen, setWasOpen] = useState(open);
  if (open !== wasOpen) {
    setWasOpen(open);
    if (open) setPreview(null);
  }

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
    scope: { id: `worker-assign-shift:${workerId}` },
    mutationFn: (values) =>
      assignWorkerShift({
        workerId: values.workerId,
        shiftTemplateId: values.shiftTemplateId,
        effectiveFrom: values.effectiveFrom,
        cycleOffsetWeeks: values.cycleOffsetWeeks,
        notes: values.notes ?? undefined,
      }),
    onSuccess: () => {
      onAssigned();
      onOpenChange(false);
    },
  });

  const rotates = (preview?.cycleWeeks ?? 1) > 1;
  const clashes = preview ? shiftClashes(preview.daysOfWeek, preferences) : [];

  return (
    <WorkPane
      open={open}
      onOpenChange={onOpenChange}
      title={moving ? t("Move to another shift") : t("Put on a shift")}
      description={t(
        "Dispatch plans them from the shift pattern. The shift in force ends the day before this one starts.",
      )}
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
          <WorkPaneBody>
            <FormGroup cols={1}>
              <FormControl>
                <ShiftTemplateAutocompleteField<AssignShiftFormValues>
                  control={control}
                  name="shiftTemplateId"
                  label={t("Shift")}
                  placeholder={t("Pick a shift")}
                  required
                  onOptionChange={(option) => setPreview(previewOf(option))}
                  description={t("Only shifts in force can be assigned.")}
                />
              </FormControl>
              <FormControl>
                <div className="flex flex-col gap-1.5">
                  <span className="text-xs font-medium">{t("Pattern")}</span>
                  <div className="flex flex-col gap-2 rounded-md border border-border-subtle bg-sunken p-2">
                    {preview ? (
                      <>
                        <p className="flex items-baseline justify-between gap-2 text-xs">
                          <span className="font-medium">
                            {describeShiftPattern(preview.daysOfWeek, preview.cycleWeeks)}
                          </span>
                          <span className="text-muted-foreground tabular-nums">
                            {formatShiftWindow(preview.startMinute, preview.durationMinutes)}
                          </span>
                        </p>
                        <WeekPattern
                          key={`${preview.daysOfWeek}-${preview.startMinute}-${preview.durationMinutes}`}
                          daysOfWeek={preview.daysOfWeek}
                          startMinute={preview.startMinute}
                          durationMinutes={preview.durationMinutes}
                          clashes={clashes}
                        />
                      </>
                    ) : (
                      <p className="text-xs text-muted-foreground">
                        {t("Pick a shift to see its week.")}
                      </p>
                    )}
                  </div>
                  {preview ? (
                    <p
                      className={cn(
                        "text-2xs",
                        clashes.length > 0 ? "text-warning-foreground" : "text-muted-foreground",
                      )}
                    >
                      {clashes.length > 0
                        ? t(
                            "Works {0}, which they said they can't.",
                            clashes.map((day) => longWeekday(day)).join(", "),
                          )
                        : t("Fits the days they said they can work.")}
                    </p>
                  ) : null}
                </div>
              </FormControl>
              <FormControl>
                <AutoCompleteDateField<AssignShiftFormValues>
                  control={control}
                  name="effectiveFrom"
                  label={t("Effective from")}
                  placeholder={t("Today")}
                  required
                  description={t("The first day the worker is on this pattern.")}
                />
              </FormControl>
              {rotates ? (
                <FormControl>
                  <NumberField<AssignShiftFormValues>
                    control={control}
                    name="cycleOffsetWeeks"
                    label={t("Rotation offset (weeks)")}
                    placeholder="0"
                    description={t(
                      "0 to {0}. Two workers on the same shift at different offsets alternate.",
                      (preview?.cycleWeeks ?? 1) - 1,
                    )}
                  />
                </FormControl>
              ) : null}
              <FormControl>
                <TextareaField<AssignShiftFormValues>
                  control={control}
                  name="notes"
                  label={t("Notes")}
                  placeholder={t("Why they are moving to this shift")}
                  description={t("Kept on the assignment record.")}
                />
              </FormControl>
            </FormGroup>
          </WorkPaneBody>
          <WorkPaneFooter>
            <WorkPaneCancel />
            <Button type="submit" size="sm" isLoading={isPending}>
              {moving ? t("Move to shift") : t("Put on shift")}
            </Button>
          </WorkPaneFooter>
        </Form>
      </FormProvider>
    </WorkPane>
  );
}
