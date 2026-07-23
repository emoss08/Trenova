import { InputField } from "@/components/fields/input-field";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@trenova/shared/components/ui/select";
import {
  buildCron,
  DEFAULT_CRON_PARTS,
  formatTimeOfDay,
  ordinalDay,
  parseCron,
  type CronFrequency,
  type CronParts,
} from "@/lib/cron";
import { cn } from "@trenova/shared/lib/utils";
import { CheckIcon } from "lucide-react";
import { useState, type ReactNode } from "react";
import { useController, type Control, type FieldPath, type FieldValues } from "react-hook-form";

type CadenceTab = CronFrequency | "custom";

const FREQUENCY_OPTIONS: { value: CadenceTab; label: string }[] = [
  { value: "daily", label: "every day" },
  { value: "weekly", label: "every week" },
  { value: "monthly", label: "every month" },
  { value: "custom", label: "a custom schedule" },
];

const WEEKDAY_CHIPS: { value: number; label: string }[] = [
  { value: 1, label: "Mon" },
  { value: 2, label: "Tue" },
  { value: 3, label: "Wed" },
  { value: 4, label: "Thu" },
  { value: 5, label: "Fri" },
  { value: 6, label: "Sat" },
  { value: 0, label: "Sun" },
];

const TIME_STEP_MINUTES = 30;

const DAY_OF_MONTH_OPTIONS: { value: string; label: string }[] = Array.from(
  { length: 31 },
  (_, i) => ({ value: String(i + 1), label: ordinalDay(i + 1) }),
);

const TIME_OPTIONS: { value: string; label: string }[] = Array.from(
  { length: (24 * 60) / TIME_STEP_MINUTES },
  (_, i) => {
    const total = i * TIME_STEP_MINUTES;
    return {
      value: String(total),
      label: formatTimeOfDay(Math.floor(total / 60), total % 60),
    };
  },
);

const pillTrigger =
  "h-7 w-auto gap-1 border-transparent bg-transparent px-1.5 font-medium text-foreground hover:bg-muted data-popup-open:bg-muted";

function timeOptions(hour: number, minute: number): { value: string; label: string }[] {
  const current = hour * 60 + minute;
  if (current % TIME_STEP_MINUTES === 0) return TIME_OPTIONS;
  return [...TIME_OPTIONS, { value: String(current), label: formatTimeOfDay(hour, minute) }].sort(
    (a, b) => Number(a.value) - Number(b.value),
  );
}

function TimeSelect({
  hour,
  minute,
  onChange,
}: {
  hour: number;
  minute: number;
  onChange: (next: { hour: number; minute: number }) => void;
}) {
  const options = timeOptions(hour, minute);

  return (
    <Select
      items={options}
      value={String(hour * 60 + minute)}
      onValueChange={(value) => {
        if (value === null) return;
        const total = Number(value);
        onChange({ hour: Math.floor(total / 60), minute: total % 60 });
      }}
    >
      <SelectTrigger size="sm" className={pillTrigger} aria-label="Time of day">
        <SelectValue />
      </SelectTrigger>
      <SelectContent className="max-h-64" alignItemWithTrigger={false}>
        {options.map((option) => (
          <SelectItem key={option.value} value={option.value}>
            {option.label}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  );
}

export function CadenceToggleChip({
  active,
  onClick,
  children,
}: {
  active: boolean;
  onClick: () => void;
  children: ReactNode;
}) {
  return (
    <button
      type="button"
      aria-pressed={active}
      onClick={onClick}
      className={cn(
        "inline-flex h-6 items-center gap-1 rounded-md border px-2 text-xs font-medium",
        "transition-[border-color,background-color,color] duration-150",
        "focus-visible:ring-2 focus-visible:ring-ring focus-visible:outline-none",
        active
          ? "border-blue-600 bg-blue-600/10 text-blue-600 dark:text-blue-400"
          : "border-input text-muted-foreground hover:border-border hover:bg-muted hover:text-foreground",
      )}
    >
      {active && <CheckIcon className="size-3" />}
      {children}
    </button>
  );
}

type CronCadenceFieldProps<T extends FieldValues> = {
  control: Control<T>;
  name: FieldPath<T>;
  verb?: string;
};

export function CronCadenceField<T extends FieldValues>({
  control,
  name,
  verb = "Run",
}: CronCadenceFieldProps<T>) {
  const { field } = useController({
    control,
    name,
    rules: { required: "A schedule is required" },
  });

  const [parts, setParts] = useState<CronParts>(() => parseCron(field.value) ?? DEFAULT_CRON_PARTS);
  const [tab, setTab] = useState<CadenceTab>(() => parseCron(field.value)?.frequency ?? "custom");

  const applyParts = (next: CronParts) => {
    setParts(next);
    field.onChange(buildCron(next));
  };

  const handleFrequencyChange = (value: CadenceTab | null) => {
    if (value === null) return;
    setTab(value);
    if (value === "custom") return;
    const base = parseCron(field.value) ?? parts;
    applyParts({ ...base, frequency: value });
  };

  const toggleWeekday = (day: number) => {
    const weekdays = parts.weekdays.includes(day)
      ? parts.weekdays.filter((value) => value !== day)
      : [...parts.weekdays, day];
    if (weekdays.length === 0) return;
    applyParts({ ...parts, weekdays });
  };

  return (
    <div className="flex flex-col gap-3 rounded-lg border border-border bg-muted/30 p-3">
      <div className="flex flex-wrap items-center gap-x-1.5 gap-y-2 text-sm">
        <span className="text-muted-foreground">{verb}</span>
        <Select items={FREQUENCY_OPTIONS} value={tab} onValueChange={handleFrequencyChange}>
          <SelectTrigger size="sm" className={pillTrigger} aria-label="Frequency">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {FREQUENCY_OPTIONS.map((option) => (
              <SelectItem key={option.value} value={option.value}>
                {option.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>

        {tab === "monthly" && (
          <>
            <span className="text-muted-foreground">on the</span>
            <Select
              items={DAY_OF_MONTH_OPTIONS}
              value={String(parts.dayOfMonth)}
              onValueChange={(value) => {
                if (value === null) return;
                applyParts({ ...parts, dayOfMonth: Number(value) });
              }}
            >
              <SelectTrigger size="sm" className={pillTrigger} aria-label="Day of month">
                <SelectValue />
              </SelectTrigger>
              <SelectContent className="max-h-64" alignItemWithTrigger={false}>
                {DAY_OF_MONTH_OPTIONS.map((option) => (
                  <SelectItem key={option.value} value={option.value}>
                    {option.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </>
        )}

        {tab !== "custom" && (
          <>
            <span className="text-muted-foreground">at</span>
            <TimeSelect
              hour={parts.hour}
              minute={parts.minute}
              onChange={({ hour, minute }) => applyParts({ ...parts, hour, minute })}
            />
          </>
        )}
      </div>

      {tab === "weekly" && (
        <div className="flex flex-wrap gap-1.5">
          {WEEKDAY_CHIPS.map((chip) => (
            <CadenceToggleChip
              key={chip.value}
              active={parts.weekdays.includes(chip.value)}
              onClick={() => toggleWeekday(chip.value)}
            >
              {chip.label}
            </CadenceToggleChip>
          ))}
        </div>
      )}

      {tab === "custom" && (
        <InputField
          control={control}
          name={name}
          placeholder="0 8 * * 1"
          description="Five fields: minute, hour, day-of-month, month, day-of-week."
          rules={{ required: "A schedule is required" }}
        />
      )}
    </div>
  );
}
