import { cn } from "@/lib/utils";
import { createCalendar } from "@internationalized/date";
import { useRef } from "react";
import {
  AriaDatePickerProps,
  DateValue,
  useDateField,
  useLocale,
} from "react-aria";
import { useDateFieldState } from "react-stately";
import { DateSegment } from "./date-components";

// CREDIT - https://github.com/uncvrd/shadcn-ui-date-time-picker

function DateField(props: AriaDatePickerProps<DateValue>) {
  const ref = useRef<HTMLDivElement | null>(null);

  const { locale } = useLocale();
  const state = useDateFieldState({
    ...props,
    locale,
    createCalendar,
  });
  const { fieldProps } = useDateField(props, state, ref);

  return (
    <div
      {...fieldProps}
      ref={ref}
      className={cn(
        "inline-flex h-9 flex-1 items-center rounded-md border w-full border-input bg-transparent py-2 pl-3 text-sm ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2",
        props.isDisabled ? "cursor-not-allowed opacity-50" : "",
      )}
    >
      {state.segments.map((segment, i) => (
        <DateSegment key={i} segment={segment} state={state} />
      ))}
      {state.isInvalid && <span aria-hidden="true">🚫</span>}
    </div>
  );
}

export { DateField };
