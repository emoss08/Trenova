import { ClockIcon } from "@trenova/shared/components/icons";
import { Button } from "@trenova/shared/components/ui/button";
import { Popover, PopoverContent, PopoverTrigger } from "@trenova/shared/components/ui/popover";
import { useT } from "@trenova/shared/i18n/use-t";
import {
  formatTimeOfDay,
  parseTimeOfDay,
  timeOfDayValue,
  usesTwelveHourClock,
} from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import { fieldInvalidClass } from "@trenova/shared/lib/variants/field";
import type { FormControlProps } from "@trenova/shared/types/fields";
import { useRef } from "react";
import { Controller, type FieldValues } from "react-hook-form";
import { FieldWrapper } from "./field-components";
import { TimePeriodSegment } from "./time-picker/time-period-segment";
import { TimePicker } from "./time-picker/time-picker";
import { TimePickerInput } from "./time-picker/time-picker-input";
import type { Period } from "./time-picker/time-picker-utils";

type TimeOfDayFieldProps<T extends FieldValues> = FormControlProps<T> & {
  label?: string;
  description?: string;
  className?: string;
  disabled?: boolean;
};

const EMPTY_SEGMENT = "--";

function shownTime(value: unknown): string {
  const time = parseTimeOfDay(typeof value === "string" ? value : "");
  return time ? formatTimeOfDay(time.hour, time.minute) : "";
}

/**
 * A time of day that belongs to no date, such as when a daily schedule runs, typed the way a
 * native time field takes it: an hour, minutes and AM or PM, each a segment that takes only digits
 * (or A and P), steps with the arrow keys and moves on once full. The clock opens a picker. It
 * follows the reader's 12- or 24-hour clock, and the form holds the 24-hour value ("14:30").
 */
export function TimeOfDayField<T extends FieldValues>({
  name,
  control,
  rules,
  label,
  description,
  className,
  disabled,
}: TimeOfDayFieldProps<T>) {
  const t = useT();
  const inputId = `input-${name}`;
  const descriptionId = `${inputId}-description`;
  const errorId = `${inputId}-error`;
  const hourCycle = usesTwelveHourClock() ? 12 : 24;
  const hourRef = useRef<HTMLInputElement>(null);
  const minuteRef = useRef<HTMLInputElement>(null);
  const periodRef = useRef<HTMLInputElement>(null);

  return (
    <Controller<T>
      name={name}
      control={control}
      rules={rules}
      render={({ field, fieldState }) => {
        const time = parseTimeOfDay(field.value as string | null | undefined);
        const date = time ? new Date(2000, 0, 1, time.hour, time.minute) : undefined;
        const period: Period = time && time.hour >= 12 ? "PM" : "AM";
        const empty = !time;
        const setDate = (next: Date | undefined) => {
          if (next) field.onChange(timeOfDayValue(next.getHours(), next.getMinutes()));
        };
        const setPeriod = (next: Period) => {
          const base = time ?? { hour: 0, minute: 0 };
          const hour = (base.hour % 12) + (next === "PM" ? 12 : 0);
          field.onChange(timeOfDayValue(hour, base.minute));
        };
        const describedBy =
          cn(description && descriptionId, fieldState.error && errorId) || undefined;

        return (
          <FieldWrapper
            name={name}
            label={label}
            description={description}
            required={!!rules?.required}
            error={fieldState.error?.message}
            htmlFor={inputId}
            descriptionId={descriptionId}
            errorId={errorId}
            formatValue={shownTime}
          >
            <div
              role="group"
              aria-label={label}
              aria-describedby={describedBy}
              aria-invalid={fieldState.invalid || undefined}
              onBlur={(event) => {
                if (!event.currentTarget.contains(event.relatedTarget as Node | null)) {
                  field.onBlur();
                }
              }}
              className={cn(
                "ui-field ui-container-focus-ring relative flex h-7 w-full min-w-0 items-center gap-px pr-8 pl-1.5 tabular-nums md:text-xs",
                disabled && "cursor-not-allowed opacity-50",
                fieldState.invalid && fieldInvalidClass,
                className,
              )}
            >
              <TimePickerInput
                bare
                ref={(node) => {
                  hourRef.current = node;
                  field.ref(node);
                }}
                id={inputId}
                picker={hourCycle === 12 ? "12hours" : "hours"}
                period={period}
                date={date}
                setDate={setDate}
                value={empty ? EMPTY_SEGMENT : undefined}
                disabled={disabled}
                aria-label={t("Hour")}
                onRightFocus={() => minuteRef.current?.focus()}
              />
              <span aria-hidden className="text-muted-foreground">
                :
              </span>
              <TimePickerInput
                bare
                ref={minuteRef}
                id={`${inputId}-minutes`}
                picker="minutes"
                date={date}
                setDate={setDate}
                value={empty ? EMPTY_SEGMENT : undefined}
                disabled={disabled}
                aria-label={t("Minute")}
                onLeftFocus={() => hourRef.current?.focus()}
                onRightFocus={hourCycle === 12 ? () => periodRef.current?.focus() : undefined}
              />
              {hourCycle === 12 ? (
                <TimePeriodSegment
                  ref={periodRef}
                  period={period}
                  empty={empty}
                  disabled={disabled}
                  onPeriodChange={setPeriod}
                  onLeftFocus={() => minuteRef.current?.focus()}
                  className="ml-0.5"
                />
              ) : null}
              <Popover>
                <PopoverTrigger
                  render={
                    <Button
                      type="button"
                      size="icon"
                      variant="ghost"
                      disabled={disabled}
                      className="text-muted-foreground absolute top-1/2 right-2 size-5 -translate-y-1/2 [&>svg]:size-3"
                    />
                  }
                >
                  <span className="sr-only">{t("Open time picker")}</span>
                  <ClockIcon className="size-4" />
                </PopoverTrigger>
                <PopoverContent align="end" sideOffset={6} className="w-auto p-3">
                  <TimePicker date={date} hourCycle={hourCycle} setDate={setDate} />
                </PopoverContent>
              </Popover>
            </div>
          </FieldWrapper>
        );
      }}
    />
  );
}
