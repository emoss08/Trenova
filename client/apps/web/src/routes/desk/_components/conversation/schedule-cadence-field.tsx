import {
  CadenceTimeSelect,
  CadenceToggleChip,
  cadencePillTrigger,
} from "@/components/fields/cron-cadence-field";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@trenova/shared/components/ui/select";
import { useT } from "@trenova/shared/i18n/use-t";
import { useController, type Control } from "react-hook-form";
import type { ScheduleEditValues, ScheduleRepeat } from "./schedule-edit-form";

type Frequency = "day" | "weekday" | "week";

const FREQUENCY_OPTIONS: { value: Frequency; label: string }[] = [
  { value: "day", label: "every day" },
  { value: "weekday", label: "every weekday" },
  { value: "week", label: "every week" },
];

const DAY_CHIPS: { value: ScheduleRepeat; label: string }[] = [
  { value: "monday", label: "Mon" },
  { value: "tuesday", label: "Tue" },
  { value: "wednesday", label: "Wed" },
  { value: "thursday", label: "Thu" },
  { value: "friday", label: "Fri" },
  { value: "saturday", label: "Sat" },
  { value: "sunday", label: "Sun" },
];

/** The frequency a repeat is read as: one day of the week is "every week". */
function frequencyOf(repeat: ScheduleRepeat): Frequency {
  return repeat === "day" || repeat === "weekday" ? repeat : "week";
}

/** A 24-hour "HH:MM" as the hour and minute the time select takes, 8:00 when unreadable. */
function clockOf(time: string): { hour: number; minute: number } {
  const match = /^(\d{1,2}):(\d{2})$/.exec(time);
  return match ? { hour: Number(match[1]), minute: Number(match[2]) } : { hour: 8, minute: 0 };
}

const pad = (value: number) => String(value).padStart(2, "0");

/**
 * When a schedule runs, written the way the report library's cadence is:
 * "Run [every week] at [8:00 AM]" over the day it runs on. A schedule runs
 * every day, every weekday, or on one day of the week, so a week picks one
 * day rather than several.
 */
export function ScheduleCadenceField({ control }: { control: Control<ScheduleEditValues> }) {
  const t = useT();
  const { field: repeat, fieldState: repeatState } = useController({ control, name: "repeat" });
  const { field: time, fieldState: timeState } = useController({ control, name: "time" });
  const error = repeatState.error?.message ?? timeState.error?.message;
  const frequency = frequencyOf(repeat.value);
  const clock = clockOf(time.value);

  return (
    <div className="flex flex-col gap-1.5">
      <span className="text-xs font-medium text-muted-foreground">{t("Cadence")}</span>
      <div className="border-border bg-muted/30 flex flex-col gap-3 rounded-lg border p-3">
        <div className="flex flex-wrap items-center gap-x-1.5 gap-y-2 text-sm">
          <span className="text-muted-foreground">{t("Run")}</span>
          <Select
            items={FREQUENCY_OPTIONS.map((option) => ({ ...option, label: t(option.label) }))}
            value={frequency}
            onValueChange={(value) => {
              if (value === null || value === frequency) return;
              repeat.onChange(value === "week" ? "monday" : value);
            }}
          >
            <SelectTrigger size="sm" className={cadencePillTrigger} aria-label={t("Frequency")}>
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {FREQUENCY_OPTIONS.map((option) => (
                <SelectItem key={option.value} value={option.value}>
                  {t(option.label)}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
          <span className="text-muted-foreground">{t("at")}</span>
          <CadenceTimeSelect
            hour={clock.hour}
            minute={clock.minute}
            onChange={({ hour, minute }) => time.onChange(`${pad(hour)}:${pad(minute)}`)}
          />
        </div>

        {frequency === "week" && (
          <div className="flex flex-wrap gap-1.5" role="group" aria-label={t("Day of the week")}>
            {DAY_CHIPS.map((chip) => (
              <CadenceToggleChip
                key={chip.value}
                active={repeat.value === chip.value}
                onClick={() => repeat.onChange(chip.value)}
              >
                {t(chip.label)}
              </CadenceToggleChip>
            ))}
          </div>
        )}
      </div>
      {error && (
        <p className="text-2xs text-destructive" role="alert">
          {error}
        </p>
      )}
    </div>
  );
}
