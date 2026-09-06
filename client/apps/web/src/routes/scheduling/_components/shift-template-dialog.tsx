import { ColorField } from "@/components/fields/color-field";
import { InputField } from "@/components/fields/input-field";
import { NumberField } from "@/components/fields/number-field";
import { SelectField } from "@/components/fields/select-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { useApiMutation } from "@/hooks/use-api-mutation";
import {
  createShiftTemplate,
  ROTA_KEY,
  SHIFT_TEMPLATES_KEY,
  updateShiftTemplate,
  type ShiftTemplateRow,
} from "@/lib/graphql/scheduling";
import { zodResolver } from "@hookform/resolvers/zod";
import { useQueryClient } from "@tanstack/react-query";
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
import { Label } from "@trenova/shared/components/ui/label";
import {
  clockToMinutes,
  DAY_LABELS,
  DAY_MASK_PRESETS,
  dayMaskToDays,
  daysToDayMask,
  describeShiftPattern,
  formatShiftWindow,
  minutesToClock,
  weeklyShiftMinutes,
} from "@trenova/shared/lib/scheduling";
import { formatHours } from "@trenova/shared/lib/timesheet";
import { cn } from "@trenova/shared/lib/utils";
import {
  shiftTemplateFormSchema,
  type ShiftTemplateFormValues,
} from "@trenova/shared/types/scheduling";
import { CalendarDaysIcon, ClockIcon, RepeatIcon } from "lucide-react";
import { useEffect } from "react";
import { FormProvider, useForm, useWatch, type Resolver } from "react-hook-form";
import { toast } from "sonner";

const STATUS_OPTIONS = [
  { value: "Active", label: "Active" },
  { value: "Inactive", label: "Retired" },
];

export type ShiftTemplateDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  template: ShiftTemplateRow | null;
};

function defaultsFor(template: ShiftTemplateRow | null): ShiftTemplateFormValues {
  if (!template) {
    return {
      code: "",
      name: "",
      description: null,
      color: null,
      daysOfWeek: "0111110",
      startTime: "06:00",
      durationHours: 10,
      cycleWeeks: 1,
      status: "Active",
    };
  }
  return {
    code: template.code,
    name: template.name,
    description: template.description ?? null,
    color: template.color ?? null,
    daysOfWeek: template.daysOfWeek,
    startTime: minutesToClock(template.startMinute),
    durationHours: template.durationMinutes / 60,
    cycleWeeks: template.cycleWeeks,
    status: template.status === "Active" ? "Active" : "Inactive",
  };
}

/**
 * A shift is described the way it is said out loud — which days, from when,
 * for how long — and the preview beside the form is the same card the board
 * draws, so what is being built is never a surprise.
 */
export function ShiftTemplateDialog({ open, onOpenChange, template }: ShiftTemplateDialogProps) {
  const queryClient = useQueryClient();
  const isEdit = Boolean(template);
  const form = useForm<ShiftTemplateFormValues>({
    resolver: zodResolver(shiftTemplateFormSchema) as Resolver<ShiftTemplateFormValues>,
    defaultValues: defaultsFor(template),
  });
  const { control, handleSubmit, reset, setValue, formState } = form;

  useEffect(() => {
    if (!open) return;
    reset(defaultsFor(template));
  }, [open, template, reset]);

  const watched = useWatch({ control });
  const daysOfWeek = watched.daysOfWeek ?? "";
  const selectedDays = dayMaskToDays(daysOfWeek);
  const startMinute = clockToMinutes(watched.startTime ?? "");
  const durationMinutes = Math.round((watched.durationHours ?? 0) * 60);
  const cycleWeeks = watched.cycleWeeks ?? 1;
  const validWindow = startMinute >= 0 && durationMinutes > 0;

  const { mutateAsync, isPending } = useApiMutation<
    { id: string },
    ShiftTemplateFormValues,
    unknown,
    ShiftTemplateFormValues
  >({
    form,
    resourceName: "Shift",
    mutationFn: (values) => {
      const input = {
        code: values.code,
        name: values.name,
        description: values.description ?? undefined,
        color: values.color ?? undefined,
        daysOfWeek: values.daysOfWeek,
        startMinute: clockToMinutes(values.startTime),
        durationMinutes: Math.round(values.durationHours * 60),
        cycleWeeks: values.cycleWeeks,
        status: values.status,
      };
      return template ? updateShiftTemplate(template.id, input) : createShiftTemplate(input);
    },
    onSuccess: () => {
      toast.success(isEdit ? "Shift updated" : "Shift added");
      void queryClient.invalidateQueries({ queryKey: [SHIFT_TEMPLATES_KEY] });
      void queryClient.invalidateQueries({ queryKey: [ROTA_KEY] });
      onOpenChange(false);
    },
  });

  function setDays(mask: string) {
    setValue("daysOfWeek", mask, { shouldDirty: true, shouldValidate: true });
  }

  function toggleDay(day: number) {
    const next = selectedDays.includes(day)
      ? selectedDays.filter((value) => value !== day)
      : [...selectedDays, day].sort((a, b) => a - b);
    setDays(daysToDayMask(next));
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-3xl">
        <DialogHeader>
          <DialogTitle>{isEdit ? "Edit the shift" : "Add a shift"}</DialogTitle>
          <DialogDescription>
            A shift is the days somebody works and when. A rotation longer than a week makes an A/B
            pair one shift with two assignments rather than two near-identical shifts.
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
            <div className="grid gap-5 md:grid-cols-[minmax(0,1fr)_240px]">
              <FormGroup className="pb-2" cols={2}>
                <FormControl>
                  <InputField<ShiftTemplateFormValues>
                    control={control}
                    name="code"
                    label="Code"
                    placeholder="e.g. DAY-A"
                    rules={{ required: true }}
                  />
                </FormControl>
                <FormControl>
                  <InputField<ShiftTemplateFormValues>
                    control={control}
                    name="name"
                    label="Name"
                    placeholder="e.g. Weekday days"
                    rules={{ required: true }}
                  />
                </FormControl>

                <FormControl cols="full">
                  <div className="flex flex-col gap-2">
                    <div className="flex flex-wrap items-center justify-between gap-2">
                      <Label>Working days</Label>
                      <div className="flex flex-wrap gap-1">
                        {DAY_MASK_PRESETS.map((preset) => (
                          <button
                            key={preset.mask}
                            type="button"
                            onClick={() => setDays(preset.mask)}
                            aria-pressed={daysOfWeek === preset.mask}
                            className={cn(
                              "text-muted-foreground hover:text-foreground hover:bg-muted rounded-full border border-transparent px-2 py-0.5 text-[11px] transition-colors",
                              daysOfWeek === preset.mask &&
                                "border-border bg-muted text-foreground",
                            )}
                          >
                            {preset.label}
                          </button>
                        ))}
                      </div>
                    </div>
                    <div className="grid grid-cols-7 gap-1">
                      {DAY_LABELS.map((label, day) => {
                        const on = selectedDays.includes(day);
                        return (
                          <button
                            key={label}
                            type="button"
                            onClick={() => toggleDay(day)}
                            aria-pressed={on}
                            className={cn(
                              "h-8 rounded-md border text-xs font-medium transition-colors",
                              on
                                ? "border-primary bg-primary text-primary-foreground"
                                : "text-muted-foreground hover:bg-muted hover:text-foreground",
                            )}
                          >
                            {label}
                          </button>
                        );
                      })}
                    </div>
                    {formState.errors.daysOfWeek ? (
                      <p className="text-destructive text-xs">
                        {formState.errors.daysOfWeek.message}
                      </p>
                    ) : null}
                  </div>
                </FormControl>

                <FormControl>
                  <InputField<ShiftTemplateFormValues>
                    control={control}
                    name="startTime"
                    label="Starts at"
                    type="time"
                    step={300}
                    rules={{ required: true }}
                  />
                </FormControl>
                <FormControl>
                  <NumberField<ShiftTemplateFormValues>
                    control={control}
                    name="durationHours"
                    label="Length (hours)"
                    step={0.25}
                    min={0.25}
                    max={24}
                    rules={{ required: true }}
                  />
                </FormControl>
                <FormControl>
                  <NumberField<ShiftTemplateFormValues>
                    control={control}
                    name="cycleWeeks"
                    label="Rotation (weeks)"
                    min={1}
                    max={8}
                    rules={{ required: true }}
                    description="1 works every week. 2 alternates, which is how an A/B pair is built."
                  />
                </FormControl>
                <FormControl>
                  <SelectField<ShiftTemplateFormValues>
                    control={control}
                    name="status"
                    label="Status"
                    options={STATUS_OPTIONS}
                    rules={{ required: true }}
                    description="Retiring is refused while anybody is still on it."
                  />
                </FormControl>
                <FormControl>
                  <ColorField<ShiftTemplateFormValues>
                    control={control}
                    name="color"
                    label="Colour"
                    description="How the shift reads on the board."
                  />
                </FormControl>
                <FormControl cols="full">
                  <TextareaField<ShiftTemplateFormValues>
                    control={control}
                    name="description"
                    label="Description"
                    placeholder="What this shift covers"
                  />
                </FormControl>
              </FormGroup>

              <ShiftPreview
                code={watched.code ?? ""}
                name={watched.name ?? ""}
                color={watched.color ?? null}
                selectedDays={selectedDays}
                daysOfWeek={daysOfWeek}
                window={validWindow ? formatShiftWindow(startMinute, durationMinutes) : null}
                weeklyMinutes={validWindow ? weeklyShiftMinutes(daysOfWeek, durationMinutes) : 0}
                cycleWeeks={cycleWeeks}
                retired={watched.status === "Inactive"}
              />
            </div>

            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
                Cancel
              </Button>
              <Button type="submit" isLoading={isPending}>
                {isEdit ? "Save" : "Add shift"}
              </Button>
            </DialogFooter>
          </Form>
        </FormProvider>
      </DialogContent>
    </Dialog>
  );
}

function ShiftPreview({
  code,
  name,
  color,
  selectedDays,
  daysOfWeek,
  window,
  weeklyMinutes,
  cycleWeeks,
  retired,
}: {
  code: string;
  name: string;
  color: string | null;
  selectedDays: number[];
  daysOfWeek: string;
  window: string | null;
  weeklyMinutes: number;
  cycleWeeks: number;
  retired: boolean;
}) {
  return (
    <aside
      className={cn(
        "bg-muted/30 flex h-fit flex-col gap-3 rounded-lg border p-3 md:sticky md:top-0",
        retired && "opacity-70",
      )}
      aria-label="Shift preview"
    >
      <p className="text-muted-foreground text-[11px] font-medium uppercase">On the board</p>
      <div className="flex items-center gap-2">
        <span className="bg-accent grid size-8 shrink-0 place-items-center rounded-md text-xs font-semibold">
          {code.trim().slice(0, 3).toUpperCase() || "—"}
        </span>
        <div className="min-w-0">
          <p className="flex items-center gap-1.5 truncate text-sm font-medium">
            {color ? (
              <span
                className="size-2 shrink-0 rounded-full"
                style={{ backgroundColor: color }}
                aria-hidden
              />
            ) : null}
            {name.trim() || "Unnamed shift"}
          </p>
          <p className="text-muted-foreground truncate text-xs">
            {describeShiftPattern(daysOfWeek, cycleWeeks)}
          </p>
        </div>
      </div>

      <div className="grid grid-cols-7 gap-0.5">
        {DAY_LABELS.map((label, day) => {
          const on = selectedDays.includes(day);
          return (
            <span
              key={label}
              className={cn(
                "grid h-6 place-items-center rounded text-[10px]",
                on ? "bg-primary text-primary-foreground" : "bg-muted text-muted-foreground/70",
              )}
              aria-hidden
            >
              {label[0]}
            </span>
          );
        })}
      </div>

      <dl className="flex flex-col gap-1.5 text-xs">
        <div className="flex items-center gap-2">
          <ClockIcon className="text-muted-foreground size-3.5 shrink-0" />
          <dd className="tabular-nums">{window ?? "Set a start and a length"}</dd>
        </div>
        <div className="flex items-center gap-2">
          <CalendarDaysIcon className="text-muted-foreground size-3.5 shrink-0" />
          <dd className="tabular-nums">
            {selectedDays.length} day{selectedDays.length === 1 ? "" : "s"} ·{" "}
            {formatHours(weeklyMinutes)} a week
          </dd>
        </div>
        <div className="flex items-center gap-2">
          <RepeatIcon className="text-muted-foreground size-3.5 shrink-0" />
          <dd>{cycleWeeks > 1 ? `${cycleWeeks}-week rotation` : "Same every week"}</dd>
        </div>
      </dl>

      {retired ? <Badge variant="inactive">Retired</Badge> : null}
    </aside>
  );
}
