import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { cn } from "@/lib/utils";
import { faCalendarAlt } from "@fortawesome/pro-duotone-svg-icons";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import React, { useRef, useState } from "react";
import { DateValue, useDatePicker, useInteractOutside } from "react-aria";
import {
  FieldValues,
  UseControllerProps,
  useController,
} from "react-hook-form";
import { DatePickerStateOptions, useDatePickerState } from "react-stately";
import { FieldDescription } from "../components";
import { ErrorMessage } from "../error-message";
import { Label } from "../label";
import { Calendar } from "./calendar";
import { DateField } from "./date-field";
import { TimeField } from "./time-field";

// CREDIT - https://github.com/uncvrd/shadcn-ui-date-time-picker

// Utility function to format date and time into a string
function formatDateToString(dateValue: any, timeValue: any = null) {
  if (!dateValue) return null;

  const { year, month, day } = dateValue;
  const hour = timeValue?.hour || 0;
  const minute = timeValue?.minute || 0;
  const second = timeValue?.second || 0;
  const millisecond = timeValue?.millisecond || 0;

  const date = new Date(
    Date.UTC(year, month - 1, day, hour, minute, second, millisecond),
  );
  return `${date.getUTCFullYear()}-${String(date.getUTCMonth() + 1).padStart(
    2,
    "0",
  )}-${String(date.getUTCDate()).padStart(2, "0")} ${String(
    date.getUTCHours(),
  ).padStart(2, "0")}:${String(date.getUTCMinutes()).padStart(2, "0")}:${String(
    date.getUTCSeconds(),
  ).padStart(2, "0")}.${String(date.getUTCMilliseconds()).padStart(3, "0")}+00`;
}

type TDateTimePickerProps<TFieldValues extends FieldValues> = {
  name: string;
} & DatePickerStateOptions<DateValue> &
  UseControllerProps<TFieldValues> &
  Omit<React.InputHTMLAttributes<HTMLInputElement>, "placeholder">;

export function DateTimePicker<TFieldValues extends FieldValues>({
  ...props
}: TDateTimePickerProps<TFieldValues>) {
  const contentRef = useRef<HTMLDivElement | null>(null);
  const groupRef = useRef<HTMLDivElement | null>(null);
  const { field, fieldState } = useController(props);
  const [open, setOpen] = useState(false);

  const state = useDatePickerState({
    ...props,
    onChange: (value) => {
      const formattedValue = formatDateToString(value, state.timeValue);

      field.onChange(formattedValue);
    },
  });

  const { groupProps, fieldProps, dialogProps, calendarProps } = useDatePicker(
    props,
    state,
    contentRef,
  );

  useInteractOutside({
    ref: contentRef,
    onInteractOutside: () => {
      setOpen(false);
    },
  });

  return (
    <>
      {props.label && (
        <Label
          className={cn(
            "text-sm font-medium bg-background border-border",
            props.rules?.required && "required",
          )}
          htmlFor={props.id}
        >
          {props.label}
        </Label>
      )}
      <div
        {...groupProps}
        ref={groupRef}
        className={cn(
          fieldState.invalid &&
            "ring-2 ring-inset ring-red-500 placeholder:text-red-500 focus:ring-red-500 bg-red-500 bg-opacity-20",
          groupProps.className,
          "flex h-9 rounded-md ring-inset ring-offset-background focus-within:ring-1 focus-within:ring-primary focus-within:ring-offset-2",
        )}
      >
        <DateField {...fieldProps} />
        <Popover open={open} onOpenChange={setOpen}>
          <PopoverTrigger asChild>
            <div className="relative">
              <div className="bg-border absolute inset-y-0 right-8 mt-1.5 h-6 w-px" />
              <TooltipProvider delayDuration={100}>
                <Tooltip>
                  <TooltipTrigger asChild>
                    <div className="absolute right-0 mr-2.5 mt-2.5">
                      <FontAwesomeIcon
                        icon={faCalendarAlt}
                        className={cn(
                          "text-muted-foreground hover:text-foreground mb-2.5 size-4 cursor-pointer",
                          fieldState.invalid && "text-red-500",
                        )}
                        onClick={() => setOpen(!open)}
                      />
                    </div>
                  </TooltipTrigger>
                  <TooltipContent sideOffset={10}>
                    <span>Select date and time</span>
                  </TooltipContent>
                </Tooltip>
              </TooltipProvider>
            </div>
          </PopoverTrigger>
          <PopoverContent ref={contentRef} className="w-full">
            <div {...dialogProps} className="space-y-3">
              <Calendar {...calendarProps} />
              {!!state.hasTime && (
                <TimeField
                  value={state.timeValue}
                  onChange={state.setTimeValue}
                />
              )}
            </div>
          </PopoverContent>
        </Popover>
      </div>
      {props.description && !fieldState.invalid && (
        <FieldDescription description={props.description as string} />
      )}
      {fieldState.invalid && (
        <ErrorMessage formError={fieldState.error?.message} />
      )}
    </>
  );
}
