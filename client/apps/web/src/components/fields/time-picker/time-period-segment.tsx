import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import React from "react";
import type { Period } from "./time-picker-utils";

/**
 * The AM/PM segment of a time typed in place: A or P sets it, the up and down arrows flip it, and
 * the left arrow goes back to the minutes, as a native time field does.
 */
export const TimePeriodSegment = React.forwardRef<
  HTMLInputElement,
  {
    period: Period;
    onPeriodChange: (period: Period) => void;
    onLeftFocus?: () => void;
    empty?: boolean;
    disabled?: boolean;
    className?: string;
  }
>(function TimePeriodSegment(
  { period, onPeriodChange, onLeftFocus, empty = false, disabled, className },
  ref,
) {
  const t = useT();

  return (
    <input
      ref={ref}
      type="text"
      readOnly
      disabled={disabled}
      aria-label={t("AM or PM")}
      value={empty ? "--" : period === "AM" ? t("AM") : t("PM")}
      onKeyDown={(event) => {
        if (event.key === "Tab") return;
        event.preventDefault();
        const key = event.key.toLowerCase();
        if (key === "a") onPeriodChange("AM");
        if (key === "p") onPeriodChange("PM");
        if (event.key === "ArrowUp" || event.key === "ArrowDown") {
          onPeriodChange(period === "AM" ? "PM" : "AM");
        }
        if (event.key === "ArrowLeft") onLeftFocus?.();
      }}
      className={cn(
        "h-5 w-6 cursor-default rounded-xs bg-transparent p-0 text-center caret-transparent outline-none focus:bg-surface-selected disabled:cursor-not-allowed",
        className,
      )}
    />
  );
});
