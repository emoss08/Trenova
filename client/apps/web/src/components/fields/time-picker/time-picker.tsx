import { useT } from "@trenova/shared/i18n/use-t";
import * as React from "react";

import { Label } from "@trenova/shared/components/ui/label";
import { TimePeriodSelect } from "./period-select";
import { TimePickerInput } from "./time-picker-input";
import type { Period } from "./time-picker-utils";

interface TimePickerProps {
  date: Date | undefined;
  setDate: (date: Date | undefined) => void;
  /** 12 shows hours 1 to 12 with AM and PM; 24 shows hours 0 to 23 and no period. */
  hourCycle?: 12 | 24;
}

function calcCurrentPeriod(date: Date | undefined) {
  if (!date) return "AM";
  return new Date(date).getHours() >= 12 ? "PM" : "AM";
}

export function TimePicker({ date, setDate, hourCycle = 12 }: TimePickerProps) {
  const t = useT();

  const currentPeriod = calcCurrentPeriod(date);
  const [period, setPeriod] = React.useState<Period>(currentPeriod);
  const [prevDate, setPrevDate] = React.useState(date);

  if (date !== prevDate) {
    setPrevDate(date);
    if (date) {
      setPeriod(currentPeriod);
    }
  }

  const id = React.useId();
  const minuteRef = React.useRef<HTMLInputElement>(null);
  const hourRef = React.useRef<HTMLInputElement>(null);
  const periodRef = React.useRef<HTMLButtonElement>(null);

  return (
    <div className="flex justify-center gap-2">
      <div className="grid gap-1 text-center">
        <Label htmlFor={`${id}-hours`} className="text-xs">
          {t("Hours")}
        </Label>
        <TimePickerInput
          id={`${id}-hours`}
          picker={hourCycle === 12 ? "12hours" : "hours"}
          period={period}
          date={date}
          setDate={setDate}
          ref={hourRef}
          onRightFocus={() => minuteRef.current?.focus()}
        />
      </div>
      <div className="grid gap-1 text-center">
        <Label htmlFor={`${id}-minutes`} className="text-xs">
          {t("Minutes")}
        </Label>
        <TimePickerInput
          picker="minutes"
          id={`${id}-minutes`}
          date={date}
          setDate={setDate}
          ref={minuteRef}
          onLeftFocus={() => hourRef.current?.focus()}
          onRightFocus={hourCycle === 12 ? () => periodRef.current?.focus() : undefined}
        />
      </div>
      {hourCycle === 12 ? (
        <div className="grid gap-1 text-center">
          <Label className="text-xs">{t("Period")}</Label>
          <TimePeriodSelect
            period={period}
            setPeriod={setPeriod}
            date={date}
            setDate={setDate}
            onLeftFocus={() => minuteRef.current?.focus()}
          />
        </div>
      ) : null}
    </div>
  );
}
