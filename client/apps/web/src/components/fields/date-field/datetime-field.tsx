import { fromUserWallClock, toUserWallClock } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import { Controller, type FieldValues } from "react-hook-form";
import { FieldWrapper } from "../field-components";
import type { AutoCompleteDateFieldProps } from "./date-field";
import { DateTimePicker } from "./datetime-picker";

export function AutoCompleteDateTimeField<T extends FieldValues>({
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
            <DateTimePicker
              {...props}
              id={inputId}
              name={field.name}
              ref={field.ref}
              aria-label={label}
              dateTime={toUserWallClock(field.value)}
              setDateTime={(date) => field.onChange(fromUserWallClock(date) ?? null)}
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
