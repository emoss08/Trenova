import { Button } from "@/components/ui/button";
import { Calendar } from "@/components/ui/calendar";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import { toDate, toUnixTimeStamp } from "@/lib/date";
import { cn } from "@/lib/utils";
import type { FormControlProps } from "@/types/fields";
import { format } from "date-fns";
import { CalendarIcon, XIcon } from "lucide-react";
import { Controller, type FieldValues } from "react-hook-form";
import { FieldWrapper } from "../field-components";
import { AutoCompleteDatePicker } from "./date-picker";
export type BaseDateFieldProps = {
  label: string;
  description?: string;
  className?: string;
  onKeyDown?: (event: React.KeyboardEvent<HTMLInputElement>) => void;
  placeholder?: string;
  clearable?: boolean;
};

export type DateFieldProps<T extends FieldValues> = BaseDateFieldProps &
  FormControlProps<T>;

const styles = {
  base: "w-full h-7 text-sm justify-start text-left font-normal border border-input bg-muted rounded-md",
  invalid:
    "border-red-500 bg-red-500/20 text-red-500 hover:text-red-500 hover:bg-red-500/20 data-[state=open]:border-red-600 data-[state=open]:outline-hidden data-[state=open]:ring-4 data-[state=open]:ring-red-400/20",
  open: "text-sm data-[state=open]:border-foreground data-[state=open]:outline-hidden data-[state=open]:ring-4 data-[state=open]:ring-foreground/20",
  focusVisible:
    "focus-visible:border-foreground focus-visible:outline-hidden focus-visible:ring-4 focus-visible:ring-foreground/20",
  hover:
    "transition-[border-color,box-shadow] duration-200 ease-in-out hover:bg-none",
  disabled: "text-muted-foreground hover:text-muted-foreground",
};

export function DateField<T extends FieldValues>({
  label,
  description,
  name,
  control,
  rules,
  className,
  placeholder,
  clearable,
}: DateFieldProps<T>) {
  return (
    <Controller<T>
      name={name}
      control={control}
      rules={rules}
      render={({ field, fieldState }) => {
        const dateValue = toDate(field.value);

        return (
          <FieldWrapper
            label={label}
            description={description}
            required={!!rules?.required}
            error={fieldState.error?.message}
            className={className}
          >
            <Popover>
              <PopoverTrigger
                render={
                  <Button
                    variant="outline"
                    className={cn(
                      styles.base,
                      styles.focusVisible,
                      styles.hover,
                      !dateValue && styles.disabled,
                      fieldState.invalid && styles.invalid,
                    )}
                  >
                    <CalendarIcon className="mr-0.5" />
                    {dateValue ? (
                      format(dateValue, "PPP")
                    ) : (
                      <span>{placeholder || "Pick a date"}</span>
                    )}
                    {clearable && dateValue && (
                      <XIcon
                        className="ml-auto h-4 w-4 cursor-pointer"
                        onClick={(e) => {
                          e.stopPropagation();
                          field.onChange(null);
                        }}
                      />
                    )}
                  </Button>
                }
              />

              <PopoverContent className="w-(--radix-popover-trigger-width) p-0">
                <Calendar
                  mode="single"
                  selected={dateValue}
                  onSelect={(date) => {
                    field.onChange(toUnixTimeStamp(date));
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
export interface DatePickerProps
  extends React.InputHTMLAttributes<HTMLInputElement> {
  date: Date | undefined;
  setDate: (date: Date | undefined) => void;
  isInvalid?: boolean;
  placeholder?: string;
  clearable?: boolean;
  label?: string;
  description?: string;
}

export type AutoCompleteDateFieldProps<T extends FieldValues> = Omit<
  DatePickerProps,
  "date" | "setDate"
> &
  FormControlProps<T>;

export function AutoCompleteDateField<T extends FieldValues>({
  name,
  control,
  rules,
  className,
  label,
  description,
  placeholder,
  isInvalid,
  ...props
}: AutoCompleteDateFieldProps<T>) {
  const inputId = `input-${name}`;
  const descriptionId = `${inputId}-description`;
  const errorId = `${inputId}-error`;

  return (
    <Controller<T>
      name={name}
      control={control}
      rules={rules}
      render={({ field, fieldState }) => {
        return (
          <FieldWrapper
            label={label}
            description={description}
            required={!!rules?.required}
            error={fieldState.error?.message}
            className={className}
          >
            <AutoCompleteDatePicker
              {...field}
              {...props}
              name={name}
              id={inputId}
              aria-label={label}
              aria-invalid={isInvalid}
              date={field.value ? toDate(field.value) : undefined}
              placeholder={placeholder}
              setDate={(date) =>
                field.onChange(date ? toUnixTimeStamp(date) : null)
              }
              onBlur={field.onBlur}
              className={className}
              isInvalid={fieldState.invalid}
              autoComplete="off"
              aria-describedby={cn(
                description && descriptionId,
                fieldState.error && errorId,
              )}
            />
          </FieldWrapper>
        );
      }}
    />
  );
}
