import { useT } from "@trenova/shared/i18n/use-t";
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
  timezoneGroupedChoices,
} from "@/lib/choices";
import { describeCron } from "@/lib/cron";
import type { RecurringShipment } from "@/types/recurring-shipment";
import { FormControl, FormGroup, FormSection } from "@trenova/shared/components/ui/form";
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
  const t = useT();

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
              <span className="font-medium">{cadence}</span> {t("in {0}", timezoneLabel(timezone))}
            </>
          ) : (
            <span className="text-muted-foreground">
              {t("Custom schedule — occurrences follow the raw cron expression in {0}.", timezoneLabel(timezone))}
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
  const t = useT();

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
        title={t("Series")}
        description={t("What this recurring lane is called and which shipment it copies")}
      >
        <FormGroup cols={2}>
          <FormControl>
            <InputField
              control={control}
              name="name"
              label={t("Name")}
              placeholder={t("Acme Weekly Chicago Run")}
              rules={{ required: true }}
              maxLength={100}
              description={t("A short name dispatchers will recognize in lists and history.")}
            />
          </FormControl>
          <FormControl>
            <SelectField
              control={control}
              name="status"
              label={t("Status")}
              placeholder={t("Select status")}
              rules={{ required: true }}
              options={statusChoices}
              description={t("Only Active series generate. Pause to hold the lane without losing its history.")}
            />
          </FormControl>
          <FormControl cols="full">
            <ShipmentAutocompleteField
              control={control}
              name="sourceShipmentId"
              label={t("Source Shipment")}
              placeholder={t("Search by Pro # or BOL...")}
              rules={{ required: "Source shipment is required" }}
              description={t("Every generated shipment copies this one's stops, commodities, and charges. Changing it does not touch shipments already generated.")}
            />
          </FormControl>
          <FormControl cols="full">
            <TextareaField
              control={control}
              name="description"
              label={t("Description")}
              placeholder={t("Weekly dry van out of the Elk Grove DC, per the 2026 Acme contract")}
              description={t("Context for the next dispatcher who has to understand why this lane exists.")}
            />
          </FormControl>
        </FormGroup>
      </FormSection>

      <FormSection
        title={t("Schedule")}
        description={t("When occurrences land, and how far ahead the shipment is booked")}
      >
        <FormGroup cols={2}>
          <FormControl cols="full">
            <CronCadenceField control={control} name="cronExpression" verb="Pick up" />
          </FormControl>
          <FormControl>
            <SelectField
              control={control}
              name="timezone"
              label={t("Timezone")}
              placeholder={t("Select timezone")}
              rules={{ required: true }}
              groups={timezoneGroupedChoices}
              renderOption={(option) => (
                <span className="flex w-full items-center justify-between gap-3">
                  <span>{t(option.label)}</span>
                  {option.description && (
                    <span className="text-muted-foreground text-xs">{t(option.description)}</span>
                  )}
                </span>
              )}
              description={t("Occurrence times are interpreted here, so the schedule holds across daylight saving shifts.")}
            />
          </FormControl>
          <FormControl>
            <NumberField
              control={control}
              name="leadTimeDays"
              label={t("Lead Time")}
              placeholder="1"
              sideText="days"
              rules={{ required: true }}
              min={0}
              max={60}
              description={t("How many days before pickup the shipment is created. Give dispatch enough runway to assign a truck.")}
            />
          </FormControl>
          <FormControl cols="full">
            <SchedulePreview />
          </FormControl>
        </FormGroup>
      </FormSection>

      <FormSection
        title={t("Series window")}
        description={t("The boundaries that stop the series on their own. Leave them empty for a lane that runs indefinitely.")}
      >
        <FormGroup cols={2}>
          <FormControl>
            <AutoCompleteDateField
              control={control}
              name="startDate"
              label={t("Start Date")}
              placeholder={t("Starts immediately")}
              description={t("The first day the series may generate. Occurrences before it are ignored.")}
              clearable
            />
          </FormControl>
          <FormControl>
            <AutoCompleteDateField
              control={control}
              name="endDate"
              label={t("End Date")}
              placeholder={t("No end date")}
              description={t("The series expires after this day. Must fall after the start date.")}
              clearable
            />
          </FormControl>
          <FormControl cols="full">
            <NumberField
              control={control}
              name="maxOccurrences"
              label={t("Max Occurrences")}
              placeholder={t("Unlimited")}
              min={1}
              description={t("The series expires once it has generated this many shipments.")}
            />
          </FormControl>
        </FormGroup>
      </FormSection>

      <FormSection
        title={t("Blocked days")}
        description={t("Days the lane cannot run, and what happens to an occurrence that lands on one")}
      >
        <FormGroup cols={1}>
          <FormControl>
            <SwitchField
              control={control}
              name="skipWeekends"
              label={t("Skip weekends")}
              description={t("Treats Saturday and Sunday as blocked, the same as a blackout date.")}
              outlined
              position="left"
            />
          </FormControl>
          <FormControl>
            <SelectField
              control={control}
              name="exceptionPolicy"
              label={t("Exception Policy")}
              placeholder={t("Select exception policy")}
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
        title={t("Generation")}
        description={t("Whether the schedule books freight on its own or waits for a dispatcher")}
      >
        <FormGroup cols={1}>
          <FormControl>
            <SwitchField
              control={control}
              name="autoGenerate"
              label={t("Generate shipments automatically")}
              description={t("Turn this off to keep the series as an on-demand template — occurrences are only created when someone runs Generate Now.")}
              outlined
              position="left"
            />
          </FormControl>
          {mode === "edit" && (
            <FormControl>
              <p className="text-2xs text-muted-foreground">
                {t("Changing the schedule recalculates the next pickup from the next future occurrence. Shipments that have already been generated are never modified.")}
              </p>
            </FormControl>
          )}
        </FormGroup>
      </FormSection>
    </div>
  );
}
