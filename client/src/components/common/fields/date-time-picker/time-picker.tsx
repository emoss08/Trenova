import React from "react";
import { TimeValue } from "react-aria";
import { TimeFieldStateOptions } from "react-stately";
import { TimeField } from "./time-field";

// CREDIT - https://github.com/uncvrd/shadcn-ui-date-time-picker

const TimePicker = React.forwardRef<
  HTMLDivElement,
  Omit<TimeFieldStateOptions<TimeValue>, "locale">
>((props, forwardedRef) => {
  return <TimeField {...props} {...forwardedRef} />;
});

TimePicker.displayName = "TimePicker";

export { TimePicker };
