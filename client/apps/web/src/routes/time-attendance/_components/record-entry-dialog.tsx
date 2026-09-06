import { AutoCompleteDateTimeField } from "@/components/fields/date-field/datetime-field";
import { InputField } from "@/components/fields/input-field";
import { NumberField } from "@/components/fields/number-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { recordTimeEntry } from "@/lib/graphql/timesheet";
import { zodResolver } from "@hookform/resolvers/zod";
import { Badge } from "@trenova/shared/components/ui/badge";
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
import { formatUnixInUserTimezone, getTodayDate } from "@trenova/shared/lib/date";
import {
  formatHours,
  manualEntryDefaults,
  paidMinutesFor,
  type ManualEntryLike,
} from "@trenova/shared/lib/timesheet";
import { cn } from "@trenova/shared/lib/utils";
import {
  recordTimeEntryFormSchema,
  type RecordTimeEntryFormValues,
} from "@trenova/shared/types/timesheet";
import { ArrowRightIcon, CoffeeIcon } from "lucide-react";
import { useEffect, useState } from "react";
import { FormProvider, useForm, useWatch, type Resolver } from "react-hook-form";
import { toast } from "sonner";

export type EditableTimeEntry = ManualEntryLike & { id: string; source: string };

export type RecordEntryDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  workerId: string;
  /** The punch being corrected; absent when hours are being added by hand. */
  entry?: EditableTimeEntry | null;
  onRecorded: () => void;
};

const BREAK_PRESETS = [0, 15, 30, 45, 60];

function formatPunchTime(unixSeconds: number): string {
  return formatUnixInUserTimezone(unixSeconds, { hour: "numeric", minute: "2-digit" });
}

/**
 * Recording or correcting a period by hand. The reason is required rather than
 * optional: a wage record altered by somebody with no reason recorded is not a
 * record anybody can defend. The strip above the fields keeps saying what the
 * entry would pay, so the figure is checked before it is saved rather than
 * after.
 */
export function RecordEntryDialog({
  open,
  onOpenChange,
  workerId,
  entry,
  onRecorded,
}: RecordEntryDialogProps) {
  const [initialValues] = useState(() =>
    manualEntryDefaults(entry ?? null, {
      dayStart: getTodayDate(),
      now: Math.floor(Date.now() / 1000),
    }),
  );
  const form = useForm<RecordTimeEntryFormValues>({
    resolver: zodResolver(recordTimeEntryFormSchema) as Resolver<RecordTimeEntryFormValues>,
    defaultValues: initialValues,
  });
  const { control, handleSubmit, reset, setValue } = form;

  useEffect(() => {
    if (!open) return;
    reset(
      manualEntryDefaults(entry ?? null, {
        dayStart: getTodayDate(),
        now: Math.floor(Date.now() / 1000),
      }),
    );
  }, [open, entry, reset]);

  const clockedInAt = useWatch({ control, name: "clockedInAt" });
  const clockedOutAt = useWatch({ control, name: "clockedOutAt" });
  const breakMinutes = useWatch({ control, name: "breakMinutes" });
  const spanMinutes = Math.max(0, Math.floor(((clockedOutAt ?? 0) - (clockedInAt ?? 0)) / 60));
  const paidMinutes = paidMinutesFor(clockedInAt ?? 0, clockedOutAt ?? 0, breakMinutes ?? 0);
  const backwards = Boolean(clockedInAt && clockedOutAt && clockedOutAt <= clockedInAt);

  const { mutateAsync, isPending } = useApiMutation<
    { id: string },
    RecordTimeEntryFormValues,
    unknown,
    RecordTimeEntryFormValues
  >({
    form,
    resourceName: "Entry",
    mutationFn: (values) =>
      recordTimeEntry({
        id: entry?.id,
        workerId,
        clockedInAt: values.clockedInAt,
        clockedOutAt: values.clockedOutAt,
        breakMinutes: values.breakMinutes,
        note: values.note ?? undefined,
        reason: values.reason,
      }),
    onSuccess: () => {
      toast.success(entry ? "Entry corrected" : "Hours recorded");
      onRecorded();
      onOpenChange(false);
    },
  });

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-xl">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            {entry ? "Correct the entry" : "Record hours"}
            {entry && entry.source !== "Clock" ? (
              <Badge variant="secondary">{entry.source}</Badge>
            ) : null}
          </DialogTitle>
          <DialogDescription>
            A period recorded by hand is marked as such, and the reason is kept with it. Hours can
            only be changed while the week is still open.
          </DialogDescription>
        </DialogHeader>

        <div
          className={cn(
            "bg-muted/30 flex flex-wrap items-center justify-between gap-3 rounded-lg border px-3 py-2.5 transition-colors",
            backwards && "border-destructive/40 bg-destructive/5",
          )}
          aria-live="polite"
        >
          <div className="flex items-center gap-2 text-sm tabular-nums">
            <span className="font-medium">{clockedInAt ? formatPunchTime(clockedInAt) : "—"}</span>
            <ArrowRightIcon className="text-muted-foreground size-3.5" />
            <span className="font-medium">
              {clockedOutAt ? formatPunchTime(clockedOutAt) : "—"}
            </span>
            <span className="text-muted-foreground text-xs">
              {backwards ? "ends before it begins" : `${formatHours(spanMinutes)} on the clock`}
            </span>
          </div>
          <div className="text-right">
            <p className="text-muted-foreground text-[11px] font-medium uppercase">Would be paid</p>
            <p className="font-mono text-xl leading-none font-semibold tabular-nums">
              {formatHours(paidMinutes)}
            </p>
          </div>
        </div>

        <FormProvider {...form}>
          <Form
            onSubmit={(submitEvent) => {
              submitEvent.preventDefault();
              submitEvent.stopPropagation();
              void handleSubmit((values) => mutateAsync(values))(submitEvent);
            }}
          >
            <FormGroup className="pb-2" cols={2}>
              <FormControl>
                <AutoCompleteDateTimeField<RecordTimeEntryFormValues>
                  control={control}
                  name="clockedInAt"
                  label="Started"
                  rules={{ required: true }}
                />
              </FormControl>
              <FormControl>
                <AutoCompleteDateTimeField<RecordTimeEntryFormValues>
                  control={control}
                  name="clockedOutAt"
                  label="Finished"
                  rules={{ required: true }}
                />
              </FormControl>
              <FormControl cols="full">
                <div className="grid gap-2 sm:grid-cols-[minmax(0,1fr)_auto] sm:items-end">
                  <NumberField<RecordTimeEntryFormValues>
                    control={control}
                    name="breakMinutes"
                    label="Unpaid break (minutes)"
                    min={0}
                    step={5}
                  />
                  <div className="flex flex-wrap gap-1 pb-0.5" aria-label="Common breaks">
                    {BREAK_PRESETS.map((minutes) => (
                      <button
                        key={minutes}
                        type="button"
                        aria-pressed={breakMinutes === minutes}
                        onClick={() =>
                          setValue("breakMinutes", minutes, {
                            shouldDirty: true,
                            shouldValidate: true,
                          })
                        }
                        className={cn(
                          "text-muted-foreground hover:text-foreground hover:bg-muted inline-flex h-7 items-center gap-1 rounded-full border border-transparent px-2 text-[11px] tabular-nums transition-colors",
                          breakMinutes === minutes && "border-border bg-muted text-foreground",
                        )}
                      >
                        {minutes === 0 ? "None" : `${minutes}m`}
                      </button>
                    ))}
                    <CoffeeIcon className="text-muted-foreground/60 ml-1 size-3.5 self-center" />
                  </div>
                </div>
              </FormControl>
              <FormControl cols="full">
                <InputField<RecordTimeEntryFormValues>
                  control={control}
                  name="reason"
                  label={entry ? "Why it is being corrected" : "Why it is being recorded by hand"}
                  placeholder="e.g. Missed clock-out"
                  rules={{ required: true }}
                  description="Kept with the record and shown on the timesheet."
                />
              </FormControl>
              <FormControl cols="full">
                <TextareaField<RecordTimeEntryFormValues>
                  control={control}
                  name="note"
                  label="Note"
                  placeholder="What this period covers"
                />
              </FormControl>
            </FormGroup>

            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
                Cancel
              </Button>
              <Button type="submit" isLoading={isPending} disabled={!workerId}>
                {entry ? "Save correction" : "Record"}
              </Button>
            </DialogFooter>
          </Form>
        </FormProvider>
      </DialogContent>
    </Dialog>
  );
}
