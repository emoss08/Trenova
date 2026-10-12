import { useT } from "@trenova/shared/i18n/use-t";
import { Button } from "@trenova/shared/components/ui/button";
import { Calendar } from "@trenova/shared/components/ui/calendar";
import { Popover, PopoverContent, PopoverTrigger } from "@trenova/shared/components/ui/popover";
import { CalendarIcon } from "@trenova/shared/components/icons";
import { fromUserWallClock, toUserWallClock } from "@trenova/shared/lib/date";
import { fieldInvalidClass } from "@trenova/shared/lib/variants/field";
import { cn } from "@trenova/shared/lib/utils";
import type { FormControlProps } from "@trenova/shared/types/fields";
import { format, startOfDay } from "date-fns";
import type React from "react";
import { useState } from "react";
import { Controller, type FieldValues } from "react-hook-form";
import { FieldWrapper, type FieldLayout } from "./field-components";

type DatePickerFieldProps<T extends FieldValues> = FormControlProps<T> & {
  label: string;
  description?: React.ReactNode;
  placeholder?: string;
  layout?: FieldLayout;
  className?: string;
  /** Days before this one cannot be picked. */
  fromDate?: Date;
};

/**
 * A single day picked from a calendar, bound to a form field holding Unix seconds at the
 * start of that day in the person's own time zone. The trigger reads the day in full.
 */
export function DatePickerField<T extends FieldValues>({
  name,
  control,
  rules,
  label,
  description,
  placeholder,
  layout,
  className,
  fromDate,
}: DatePickerFieldProps<T>) {
  const t = useT();
  const [open, setOpen] = useState(false);
  const fieldId = `date-${name}`;

  return (
    <Controller
      name={name}
      control={control}
      rules={rules}
      render={({ field, fieldState }) => {
        const selected = toUserWallClock(typeof field.value === "number" ? field.value : null);

        return (
          <FieldWrapper
            name={name}
            label={label}
            description={description}
            required={!!rules?.required}
            error={fieldState.error?.message}
            descriptionId={`${fieldId}-description`}
            errorId={`${fieldId}-error`}
            className={className}
            layout={layout}
          >
            <Popover open={open} onOpenChange={setOpen}>
              <PopoverTrigger
                render={
                  <Button
                    id={fieldId}
                    type="button"
                    variant="outline"
                    data-empty={!selected}
                    aria-invalid={fieldState.invalid}
                    disabled={field.disabled}
                    className={cn(
                      "data-[empty=true]:text-muted-foreground w-full justify-start text-left font-normal",
                      fieldState.invalid && fieldInvalidClass,
                    )}
                  />
                }
              >
                <CalendarIcon className="size-3.5" />
                {selected ? (
                  format(selected, "PPP")
                ) : (
                  <span>{placeholder ?? t("Pick a date")}</span>
                )}
              </PopoverTrigger>
              <PopoverContent className="w-auto p-0">
                <Calendar
                  mode="single"
                  selected={selected}
                  disabled={fromDate ? { before: fromDate } : undefined}
                  onSelect={(day: Date | undefined) => {
                    field.onChange(day ? (fromUserWallClock(startOfDay(day)) ?? null) : null);
                    setOpen(false);
                  }}
                />
              </PopoverContent>
            </Popover>
          </FieldWrapper>
        );
      }}
    />
  );
}
