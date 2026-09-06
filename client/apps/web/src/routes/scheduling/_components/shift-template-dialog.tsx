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
  dayMaskToDays,
  daysToDayMask,
  formatShiftWindow,
  minutesToClock,
} from "@trenova/shared/lib/scheduling";
import { cn } from "@trenova/shared/lib/utils";
import {
  shiftTemplateFormSchema,
  type ShiftTemplateFormValues,
} from "@trenova/shared/types/scheduling";
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

  const daysOfWeek = useWatch({ control, name: "daysOfWeek" });
  const startTime = useWatch({ control, name: "startTime" });
  const durationHours = useWatch({ control, name: "durationHours" });
  const selectedDays = dayMaskToDays(daysOfWeek ?? "");
  const startMinute = clockToMinutes(startTime ?? "");
  const durationMinutes = Math.round((durationHours ?? 0) * 60);

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

  function toggleDay(day: number) {
    const next = selectedDays.includes(day)
      ? selectedDays.filter((value) => value !== day)
      : [...selectedDays, day].sort((a, b) => a - b);
    setValue("daysOfWeek", daysToDayMask(next), { shouldDirty: true, shouldValidate: true });
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-2xl">
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
                <div className="flex flex-col gap-1.5">
                  <Label>Working days</Label>
                  <div className="flex flex-wrap gap-1">
                    {DAY_LABELS.map((label, day) => (
                      <button
                        key={label}
                        type="button"
                        onClick={() => toggleDay(day)}
                        aria-pressed={selectedDays.includes(day)}
                        className={cn(
                          "h-8 w-12 rounded-md border text-xs font-medium transition-colors",
                          selectedDays.includes(day)
                            ? "border-primary bg-primary text-primary-foreground"
                            : "text-muted-foreground hover:bg-muted",
                        )}
                      >
                        {label}
                      </button>
                    ))}
                  </div>
                  <p className="text-muted-foreground text-xs">
                    Indexed from Sunday, the same way the rota reads it.
                  </p>
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
                  rules={{ required: true }}
                  description={
                    startMinute >= 0 && durationMinutes > 0
                      ? formatShiftWindow(startMinute, durationMinutes)
                      : "How long the shift runs"
                  }
                />
              </FormControl>
              <FormControl>
                <NumberField<ShiftTemplateFormValues>
                  control={control}
                  name="cycleWeeks"
                  label="Rotation (weeks)"
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
                  description="Retiring a shift is refused while anybody is still on it."
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
