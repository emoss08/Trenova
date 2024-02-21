/*
 * COPYRIGHT(c) 2024 Trenova
 *
 * This file is part of Trenova.
 *
 * The Trenova software is licensed under the Business Source License 1.1. You are granted the right
 * to copy, modify, and redistribute the software, but only for non-production use or with a total
 * of less than three server instances. Starting from the Change Date (November 16, 2026), the
 * software will be made available under version 2 or later of the GNU General Public License.
 * If you use the software in violation of this license, your rights under the license will be
 * terminated automatically. The software is provided "as is," and the Licensor disclaims all
 * warranties and conditions. If you use this license's text or the "Business Source License" name
 * and trademark, you must comply with the Licensor's covenants, which include specifying the
 * Change License as the GPL Version 2.0 or a compatible license, specifying an Additional Use
 * Grant, and not modifying the license in any other way.
 */

import { Input } from "@/components/common/fields/input";
import { Label } from "@/components/common/fields/label";
import { Calendar } from "@/components/ui/calendar";
import { cn, useClickOutside } from "@/lib/utils";
import { CalendarIcon } from "@radix-ui/react-icons";
import { addDays, format, parseISO } from "date-fns";
import React, { useState } from "react";
import {
  Controller,
  FieldValues,
  UseControllerProps,
  useController,
} from "react-hook-form";
import { FieldErrorMessage } from "./error-message";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "./select";

interface DatepickerFieldProps
  extends React.InputHTMLAttributes<HTMLInputElement> {
  label: string;
  description?: string;
  placeholder?: string;
  initialDate?: Date;
}

const PRESET_VALUES = [
  { value: "0", label: "Today" },
  { value: "1", label: "Tomorrow" },
  { value: "3", label: "In 3 days" },
  { value: "7", label: "In a week" },
  { value: "14", label: "In 2 weeks" },
  { value: "30", label: "In a month" },
];

export function DatepickerField<TFieldValues extends FieldValues>({
  ...props
}: DatepickerFieldProps & UseControllerProps<TFieldValues>) {
  const { field, fieldState } = useController(props);
  const [date] = useState<Date | undefined>(props.initialDate);
  const popoverRef = React.useRef<HTMLDivElement>(null);
  const [isOpen, setIsOpen] = React.useState(false);
  const [stringDate, setStringDate] = React.useState(
    props.initialDate ? format(props.initialDate, "yyyy-MM-dd") : "",
  );

  const handleDateChange = (date: Date | undefined) => {
    if (date) {
      const formattedDate = format(date, "yyyy-MM-dd");
      field.onChange(formattedDate);
    } else {
      field.onChange(""); // Clear the value if the date is removed
    }
    setIsOpen(false);
  };

  const close = () => setIsOpen(false);
  useClickOutside(popoverRef, close);

  const formattedDate = field.value ? format(parseISO(field.value), "PPP") : "";

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
      <div className="relative w-full">
        <Controller
          name={props.name}
          control={props.control}
          render={({ field }) => (
            <Input
              {...field}
              onClick={() => setIsOpen(true)}
              aria-invalid={fieldState.invalid}
              value={formattedDate}
              className={cn(
                "flex h-9 w-full rounded-md border border-border bg-background px-3 py-2 text-sm file:border-0 file:bg-transparent file:text-sm file:font-medium file:text-foreground placeholder:text-muted-foreground focus-visible:outline-none focus:ring-1 focus:ring-inset focus:ring-foreground disabled:cursor-not-allowed disabled:opacity-50 sm:text-sm sm:leading-6",
                fieldState.invalid &&
                  "ring-1 ring-inset ring-red-500 placeholder:text-red-500 focus:ring-red-500",
                props.className,
              )}
              onFocus={() => {
                if (date) setStringDate(format(date, "PPP"));
              }}
              onChange={(e) => setStringDate(e.target.value)}
              {...props}
            />
          )}
        />
        <div
          className={cn(
            "absolute inset-y-0 right-8 my-2 h-6 w-px",
            fieldState.invalid ? "bg-red-500" : "bg-foreground/30",
          )}
        />
        <div className="absolute right-0 top-0 mx-2 my-3">
          {fieldState.invalid ? (
            <></>
          ) : (
            <CalendarIcon className="text-foreground/50 hover:text-foreground" />
          )}
        </div>

        {fieldState.invalid && (
          <FieldErrorMessage formError={fieldState.error?.message} />
        )}
        {props.description && !fieldState.invalid && (
          <p className="text-xs text-foreground/70">{props.description}</p>
        )}
        {isOpen && (
          <div
            ref={popoverRef}
            className="z-1000 absolute bottom-full mb-2 rounded-sm border border-muted bg-background shadow-md"
          >
            <div className="flex w-auto flex-col space-y-2 p-2">
              <Select
                onValueChange={(value) =>
                  handleDateChange(addDays(new Date(), parseInt(value)))
                }
              >
                <SelectTrigger className="h-8">
                  <SelectValue placeholder="Select Preset" />
                </SelectTrigger>
                <SelectContent position="popper">
                  {PRESET_VALUES.map((preset) => (
                    <SelectItem key={preset.value} value={preset.value}>
                      {preset.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
              <Calendar
                mode="single"
                selected={field.value ? parseISO(field.value) : undefined}
                onSelect={handleDateChange}
              />
            </div>
          </div>
        )}
      </div>
    </>
  );
}

// interface DateTimePickerFieldProps
//   extends React.InputHTMLAttributes<HTMLInputElement> {
//   label: string;
//   description?: string;
//   placeholder?: string;
//   initialDate?: Date;
//   initialTime?: string;
// }

// const TIME_PRESET_VALUES = [
//   { value: "00:00", label: "Midnight" },
//   { value: "06:00", label: "Morning" },
//   { value: "12:00", label: "Noon" },
//   { value: "18:00", label: "Evening" },
//   { value: "23:59", label: "Midnight" },
//   { value: "now", label: "Now" },
// ];
