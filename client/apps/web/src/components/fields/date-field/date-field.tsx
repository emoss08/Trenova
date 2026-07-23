import { Button } from "@trenova/shared/components/ui/button";
import { Calendar } from "@trenova/shared/components/ui/calendar";
import { Popover, PopoverContent, PopoverTrigger } from "@trenova/shared/components/ui/popover";
import { toDate, toUnixTimeStamp } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import type { FormControlProps } from "@trenova/shared/types/fields";
import { format } from "date-fns";
import { CalendarIcon, XIcon } from "lucide-react";
import { useState } from "react";
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
  disabled?: boolean;
  readOnly?: boolean;
};

export type DateFieldProps<T extends FieldValues> = BaseDateFieldProps & FormControlProps<T>;

const styles = {
  base: "w-full h-7 text-sm justify-start text-left font-normal border border-input bg-muted rounded-md",
  invalid:
    "border-red-500 bg-red-500/20 text-red-500 hover:text-red-500 hover:bg-red-500/20 data-[state=open]:border-red-600 data-[state=open]:outline-hidden data-[state=open]:ring-4 data-[state=open]:ring-red-400/20",
  open: "text-sm data-[state=open]:border-foreground data-[state=open]:outline-hidden data-[state=open]:ring-4 data-[state=open]:ring-foreground/20",
  focusVisible:
    "focus-visible:border-foreground focus-visible:outline-hidden focus-visible:ring-4 focus-visible:ring-foreground/20",
  hover: "transition-[border-color,box-shadow] duration-200 ease-in-out hover:bg-none",
  disabled: "text-muted-foreground hover:text-muted-foreground",
};

type DateFieldControlProps = {
  dateValue: Date | undefined;
  isLocked: boolean;
  readOnly: boolean;
  isInvalid: boolean;
  placeholder?: string;
  clearable?: boolean;
  onSelect: (date: Date | undefined) => void;
  onClear: () => void;
  onBlur: () => void;
};

function DateFieldControl({
  dateValue,
  isLocked,
  readOnly,
  isInvalid,
  placeholder,
  clearable,
  onSelect,
  onClear,
  onBlur,
}: DateFieldControlProps) {
  const [open, setOpen] = useState(false);

  return (
    <Popover
      open={open}
      onOpenChange={(next) => {
        if (next && isLocked) return;
        setOpen(next);
        if (!next) {
          onBlur();
        }
      }}
    >
      <PopoverTrigger
        render={
          <Button
            variant="outline"
            disabled={isLocked}
            aria-readonly={readOnly || undefined}
            className={cn(
              styles.base,
              styles.focusVisible,
              styles.hover,
              !dateValue && styles.disabled,
              isLocked && "cursor-not-allowed opacity-50",
              isInvalid && styles.invalid,
            )}
          >
            <CalendarIcon className="mr-0.5" />
            {dateValue ? format(dateValue, "PPP") : <span>{placeholder || "Pick a date"}</span>}
            {clearable && dateValue && !isLocked && (
              <XIcon
                className="ml-auto h-4 w-4 cursor-pointer"
                onClick={(e) => {
                  e.stopPropagation();
                  e.preventDefault();
                  onClear();
                }}
              />
            )}
          </Button>
        }
      />

      <PopoverContent className="w-auto p-0">
        <Calendar
          mode="single"
          selected={dateValue}
          defaultMonth={dateValue}
          onSelect={(date) => {
            onSelect(date);
            setOpen(false);
          }}
        />
      </PopoverContent>
    </Popover>
  );
}

export function DateField<T extends FieldValues>({
  label,
  description,
  name,
  control,
  rules,
  className,
  placeholder,
  clearable,
  disabled = false,
  readOnly = false,
}: DateFieldProps<T>) {
  return (
    <Controller<T>
      name={name}
      control={control}
      rules={rules}
      render={({ field, fieldState }) => {
        const dateValue = toDate(field.value);
        const isLocked = disabled || !!field.disabled || readOnly;

        return (
          <FieldWrapper
            label={label}
            description={description}
            required={!!rules?.required}
            error={fieldState.error?.message}
            className={className}
          >
            <DateFieldControl
              dateValue={dateValue}
              isLocked={isLocked}
              readOnly={readOnly}
              isInvalid={fieldState.invalid}
              placeholder={placeholder}
              clearable={clearable}
              onSelect={(date) => {
                if (isLocked) return;
                field.onChange(toUnixTimeStamp(date) ?? null);
              }}
              onClear={() => field.onChange(null)}
              onBlur={field.onBlur}
            />
          </FieldWrapper>
        );
      }}
    />
  );
}

export interface DatePickerProps
  extends Omit<React.ComponentProps<"input">, "value" | "defaultValue" | "onChange"> {
  date: Date | undefined;
  setDate: (date: Date | undefined) => void;
  isInvalid?: boolean;
  clearable?: boolean;
}

export type AutoCompleteDateFieldProps<T extends FieldValues> = Omit<
  DatePickerProps,
  "date" | "setDate"
> &
  FormControlProps<T> & {
    label?: string;
    description?: string;
  };

export function AutoCompleteDateField<T extends FieldValues>({
  name,
  control,
  rules,
  className,
  label,
  description,
  placeholder,
  disabled,
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
            descriptionId={descriptionId}
            errorId={errorId}
            required={!!rules?.required}
            error={fieldState.error?.message}
            className={className}
          >
            <AutoCompleteDatePicker
              {...props}
              id={inputId}
              name={field.name}
              ref={field.ref}
              aria-label={label}
              date={field.value ? toDate(field.value) : undefined}
              setDate={(date) => field.onChange(date ? (toUnixTimeStamp(date) ?? null) : null)}
              onBlur={field.onBlur}
              placeholder={placeholder}
              disabled={disabled || field.disabled}
              isInvalid={fieldState.invalid}
              aria-describedby={
                cn(description && descriptionId, fieldState.error && errorId) || undefined
              }
            />
          </FieldWrapper>
        );
      }}
    />
  );
}
