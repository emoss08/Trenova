import { ControlledShipmentAutocompleteField } from "@/components/autocomplete-fields";
import { CadenceToggleChip, CronCadenceField } from "@/components/fields/cron-cadence-field";
import { DateField } from "@/components/fields/date-field/date-field";
import { InputField } from "@/components/fields/input-field";
import { NumberField } from "@/components/fields/number-field";
import { SelectField } from "@/components/fields/select-field";
import { SwitchField } from "@/components/fields/switch-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { Button } from "@/components/ui/button";
import { FormControl, FormGroup } from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import { timezoneChoices } from "@/lib/choices";
import type { RecurringShipment } from "@/types/recurring-shipment";
import { PlusIcon, XIcon } from "lucide-react";
import { useState } from "react";
import { useController, useFormContext } from "react-hook-form";

const exceptionPolicyChoices = [
  { value: "Skip", label: "Skip the occurrence" },
  { value: "PreviousBusinessDay", label: "Move to previous business day" },
  { value: "NextBusinessDay", label: "Move to next business day" },
];

function SourceShipmentField() {
  const { control } = useFormContext<RecurringShipment>();
  const { field, fieldState } = useController({
    control,
    name: "sourceShipmentId",
    rules: { required: "Source shipment is required" },
  });

  return (
    <div className="flex flex-col gap-0.5">
      <ControlledShipmentAutocompleteField
        label="Source Shipment"
        placeholder="Search by Pro # or BOL..."
        value={field.value ?? ""}
        onValueChange={field.onChange}
        description="Every generated shipment copies this shipment's stops, commodities, and charges"
      />
      {fieldState.error?.message ? (
        <p className="text-2xs text-red-500">{fieldState.error.message}</p>
      ) : null}
    </div>
  );
}

function BlackoutDatesField() {
  const { control } = useFormContext<RecurringShipment>();
  const { field } = useController({ control, name: "blackoutDates" });
  const [draft, setDraft] = useState("");

  const dates = field.value ?? [];

  const addDate = () => {
    if (!draft) return;
    if (dates.includes(draft)) {
      setDraft("");
      return;
    }
    field.onChange([...dates, draft].sort());
    setDraft("");
  };

  const removeDate = (date: string) => {
    field.onChange(dates.filter((value) => value !== date));
  };

  return (
    <div className="flex flex-col gap-2">
      <div className="flex flex-col gap-0.5">
        <label className="text-sm font-medium">Blackout Dates</label>
        <p className="text-2xs text-muted-foreground">
          Holidays or shutdown days. Occurrences landing on these dates follow the exception policy.
        </p>
      </div>
      <div className="flex items-center gap-2">
        <Input
          type="date"
          value={draft}
          onChange={(event) => setDraft(event.target.value)}
          onKeyDown={(event) => {
            if (event.key === "Enter") {
              event.preventDefault();
              addDate();
            }
          }}
          className="h-7 w-40"
          aria-label="Add blackout date"
        />
        <Button type="button" variant="outline" size="sm" onClick={addDate} disabled={!draft}>
          <PlusIcon className="size-3.5" />
          Add
        </Button>
      </div>
      {dates.length > 0 && (
        <div className="flex flex-wrap gap-1.5">
          {dates.map((date) => (
            <CadenceToggleChip key={date} active onClick={() => removeDate(date)}>
              {date}
              <XIcon className="size-3" />
            </CadenceToggleChip>
          ))}
        </div>
      )}
    </div>
  );
}

export function RecurringShipmentForm({ mode }: { mode: "create" | "edit" }) {
  const { control } = useFormContext<RecurringShipment>();

  return (
    <FormGroup cols={2}>
      <FormControl>
        <InputField
          control={control}
          rules={{ required: true }}
          name="name"
          label="Name"
          placeholder="e.g. Acme Weekly Chicago Run"
          description="A short name dispatchers will recognize"
          maxLength={100}
        />
      </FormControl>
      <FormControl>
        <SourceShipmentField />
      </FormControl>
      <FormControl cols="full">
        <TextareaField
          control={control}
          name="description"
          label="Description"
          placeholder="Description"
          description="Optional notes about this recurring lane"
        />
      </FormControl>
      <FormControl cols="full">
        <CronCadenceField control={control} name="cronExpression" verb="Pick up" />
      </FormControl>
      <FormControl>
        <SelectField
          control={control}
          rules={{ required: true }}
          name="timezone"
          label="Timezone"
          placeholder="Timezone"
          description="Occurrence times are interpreted in this timezone"
          options={timezoneChoices}
        />
      </FormControl>
      <FormControl>
        <NumberField
          control={control}
          name="leadTimeDays"
          label="Lead Time (days)"
          description="How many days before pickup the shipment is created"
          min={0}
          max={60}
        />
      </FormControl>
      <FormControl>
        <DateField
          control={control}
          name="startDate"
          label="Start Date"
          description="First day the series can generate (optional)"
          clearable
        />
      </FormControl>
      <FormControl>
        <DateField
          control={control}
          name="endDate"
          label="End Date"
          description="The series expires after this date (optional)"
          clearable
        />
      </FormControl>
      <FormControl>
        <NumberField
          control={control}
          name="maxOccurrences"
          label="Max Occurrences"
          description="Stop after this many shipments (optional)"
          min={1}
        />
      </FormControl>
      <FormControl>
        <SelectField
          control={control}
          rules={{ required: true }}
          name="exceptionPolicy"
          label="Exception Policy"
          placeholder="Exception policy"
          description="What happens when an occurrence lands on a blocked day"
          options={exceptionPolicyChoices}
        />
      </FormControl>
      <FormControl>
        <SwitchField
          control={control}
          name="skipWeekends"
          label="Skip Weekends"
          description="Treat Saturdays and Sundays as blocked days"
          outlined
        />
      </FormControl>
      <FormControl>
        <SwitchField
          control={control}
          name="autoGenerate"
          label="Auto-Generate"
          description="Generate shipments automatically on schedule. Turn off for on-demand templates."
          outlined
        />
      </FormControl>
      <FormControl cols="full">
        <BlackoutDatesField />
      </FormControl>
      {mode === "edit" && (
        <FormControl cols="full">
          <p className="text-2xs text-muted-foreground">
            Changing the schedule recalculates the next pickup. Already-generated shipments are
            never modified.
          </p>
        </FormControl>
      )}
    </FormGroup>
  );
}
