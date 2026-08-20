import { ShipmentAutocompleteField } from "@/components/autocomplete-fields";
import { CronCadenceField } from "@/components/fields/cron-cadence-field";
import { AutoCompleteDateField } from "@/components/fields/date-field/date-field";
import { InputField } from "@/components/fields/input-field";
import { NumberField } from "@/components/fields/number-field";
import { SelectField } from "@/components/fields/select-field";
import { SwitchField } from "@/components/fields/switch-field";
import { TextareaField } from "@/components/fields/textarea-field";
import {
  recurringShipmentExceptionPolicyChoices,
  recurringShipmentStatusChoices,
  timezoneChoices,
} from "@/lib/choices";
import { describeCron } from "@/lib/cron";
import { FormControl, FormGroup, FormSection } from "@trenova/shared/components/ui/form";
import type { RecurringShipment } from "@/types/recurring-shipment";
import { CalendarClockIcon } from "lucide-react";
import { useMemo } from "react";
import { useFormContext, useWatch } from "react-hook-form";
import { BlackoutDatesField } from "./blackout-dates-field";

function timezoneLabel(timezone: string | undefined): string {
  if (!timezone) return "the series timezone";
  return timezoneChoices.find((choice) => choice.value === timezone)?.label ?? timezone;
}

/**
 * Restates the cron expression, timezone, and lead time as one sentence.
 *
 * The cadence builder speaks in cron, but a dispatcher signs off on "when does
 * the truck actually get booked" — the gap between those two is where
 * mis-scheduled series come from.
 */
function SchedulePreview() {
  const { control } = useFormContext<RecurringShipment>();
  const cronExpression = useWatch({ control, name: "cronExpression" });
  const timezone = useWatch({ control, name: "timezone" });
  const leadTimeDays = useWatch({ control, name: "leadTimeDays" });
  const autoGenerate = useWatch({ control, name: "autoGenerate" });

  const cadence = describeCron(cronExpression ?? "");

  const lead =
    leadTimeDays && leadTimeDays > 0
      ? `created ${leadTimeDays} day${leadTimeDays === 1 ? "" : "s"} ahead of pickup`
      : "created the same day as pickup";

  return (
    <div className="border-border bg-muted/30 flex items-start gap-2 rounded-lg border p-3">
      <CalendarClockIcon className="text-muted-foreground mt-0.5 size-4 shrink-0" />
      <div className="flex flex-col gap-0.5 text-sm">
        <span>
          {cadence ? (
            <>
              <span className="font-medium">{cadence}</span> in {timezoneLabel(timezone)}
            </>
          ) : (
            <span className="text-muted-foreground">
              Custom schedule — occurrences follow the raw cron expression in{" "}
              {timezoneLabel(timezone)}.
            </span>
          )}
        </span>
        <span className="text-2xs text-muted-foreground">
          {autoGenerate
            ? `Shipments are ${lead}.`
            : `Auto-generation is off — occurrences wait for someone to generate them, and would be ${lead}.`}
        </span>
      </div>
    </div>
  );
}

export function RecurringShipmentForm({ mode }: { mode: "create" | "edit" }) {
  const { control } = useFormContext<RecurringShipment>();

  const skipWeekends = useWatch({ control, name: "skipWeekends" });
  const blackoutDates = useWatch({ control, name: "blackoutDates" });
  const hasBlockedDays = skipWeekends || (blackoutDates?.length ?? 0) > 0;

  // Expired is a terminal state the scheduler sets when the window or the
  // occurrence cap runs out — a new series can never start there.
  const statusChoices = useMemo(
    () =>
      mode === "create"
        ? recurringShipmentStatusChoices.filter((choice) => choice.value !== "Expired")
        : recurringShipmentStatusChoices,
    [mode],
  );

  return (
    <div className="flex flex-col gap-2">
      <FormSection
        title="Series"
        description="What this recurring lane is called and which shipment it copies"
      >
        <FormGroup cols={2}>
          <FormControl>
            <InputField
              control={control}
              name="name"
              label="Name"
              placeholder="Acme Weekly Chicago Run"
              rules={{ required: true }}
              maxLength={100}
              description="A short name dispatchers will recognize in lists and history."
            />
          </FormControl>
          <FormControl>
            <SelectField
              control={control}
              name="status"
              label="Status"
              placeholder="Select status"
              rules={{ required: true }}
              options={statusChoices}
              description="Only Active series generate. Pause to hold the lane without losing its history."
            />
          </FormControl>
          <FormControl cols="full">
            <ShipmentAutocompleteField
              control={control}
              name="sourceShipmentId"
              label="Source Shipment"
              placeholder="Search by Pro # or BOL..."
              rules={{ required: "Source shipment is required" }}
              description="Every generated shipment copies this one's stops, commodities, and charges. Changing it does not touch shipments already generated."
            />
          </FormControl>
          <FormControl cols="full">
            <TextareaField
              control={control}
              name="description"
              label="Description"
              placeholder="Weekly dry van out of the Elk Grove DC, per the 2026 Acme contract"
              description="Context for the next dispatcher who has to understand why this lane exists."
            />
          </FormControl>
        </FormGroup>
      </FormSection>

      <FormSection
        title="Schedule"
        description="When occurrences land, and how far ahead the shipment is booked"
      >
        <FormGroup cols={2}>
          <FormControl cols="full">
            <CronCadenceField control={control} name="cronExpression" verb="Pick up" />
          </FormControl>
          <FormControl>
            <SelectField
              control={control}
              name="timezone"
              label="Timezone"
              placeholder="Select timezone"
              rules={{ required: true }}
              options={timezoneChoices}
              description="Occurrence times are interpreted here, so the schedule holds across daylight saving shifts."
            />
          </FormControl>
          <FormControl>
            <NumberField
              control={control}
              name="leadTimeDays"
              label="Lead Time"
              placeholder="1"
              sideText="days"
              rules={{ required: true }}
              min={0}
              max={60}
              description="How many days before pickup the shipment is created. Give dispatch enough runway to assign a truck."
            />
          </FormControl>
          <FormControl cols="full">
            <SchedulePreview />
          </FormControl>
        </FormGroup>
      </FormSection>

      <FormSection
        title="Series window"
        description="The boundaries that stop the series on their own. Leave them empty for a lane that runs indefinitely."
      >
        <FormGroup cols={2}>
          <FormControl>
            <AutoCompleteDateField
              control={control}
              name="startDate"
              label="Start Date"
              placeholder="Starts immediately"
              description="The first day the series may generate. Occurrences before it are ignored."
              clearable
            />
          </FormControl>
          <FormControl>
            <AutoCompleteDateField
              control={control}
              name="endDate"
              label="End Date"
              placeholder="No end date"
              description="The series expires after this day. Must fall after the start date."
              clearable
            />
          </FormControl>
          <FormControl cols="full">
            <NumberField
              control={control}
              name="maxOccurrences"
              label="Max Occurrences"
              placeholder="Unlimited"
              min={1}
              description="The series expires once it has generated this many shipments."
            />
          </FormControl>
        </FormGroup>
      </FormSection>

      <FormSection
        title="Blocked days"
        description="Days the lane cannot run, and what happens to an occurrence that lands on one"
      >
        <FormGroup cols={1}>
          <FormControl>
            <SwitchField
              control={control}
              name="skipWeekends"
              label="Skip weekends"
              description="Treats Saturday and Sunday as blocked, the same as a blackout date."
              outlined
              position="left"
            />
          </FormControl>
          <FormControl>
            <SelectField
              control={control}
              name="exceptionPolicy"
              label="Exception Policy"
              placeholder="Select exception policy"
              rules={{ required: true }}
              options={recurringShipmentExceptionPolicyChoices}
              description={
                hasBlockedDays
                  ? "Applied whenever an occurrence lands on a weekend or blackout date."
                  : "Nothing is blocked yet, so this policy stays dormant until you skip weekends or add a blackout date."
              }
            />
          </FormControl>
          <FormControl cols="full">
            <BlackoutDatesField />
          </FormControl>
        </FormGroup>
      </FormSection>

      <FormSection
        title="Generation"
        description="Whether the schedule books freight on its own or waits for a dispatcher"
      >
        <FormGroup cols={1}>
          <FormControl>
            <SwitchField
              control={control}
              name="autoGenerate"
              label="Generate shipments automatically"
              description="Turn this off to keep the series as an on-demand template — occurrences are only created when someone runs Generate Now."
              outlined
              position="left"
            />
          </FormControl>
          {mode === "edit" && (
            <FormControl>
              <p className="text-2xs text-muted-foreground">
                Changing the schedule recalculates the next pickup from the next future occurrence.
                Shipments that have already been generated are never modified.
              </p>
            </FormControl>
          )}
        </FormGroup>
      </FormSection>
    </div>
  );
}
