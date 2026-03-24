import { toDate, toUnixTimeStamp } from "@/lib/date";
import { cn } from "@/lib/utils";
import { useCallback, useMemo } from "react";
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
        // eslint-disable-next-line react-hooks/rules-of-hooks
        const dateValue = useMemo(
          () => (field.value ? toDate(field.value) : undefined),
          [field.value],
        );

        // eslint-disable-next-line react-hooks/rules-of-hooks
        const handleChange = useCallback(
          (date: Date | undefined) => {
            const formattedDate = toUnixTimeStamp(date);
            field.onChange(formattedDate);
          },
          [field],
        );

        return (
          <FieldWrapper
            label={label}
            description={description}
            required={!!rules?.required}
            error={fieldState.error?.message}
            className={className}
          >
            <DateTimePicker
              {...props}
              {...field}
              name={name}
              id={inputId}
              aria-label={label}
              dateTime={dateValue || undefined}
              placeholder={placeholder}
              setDateTime={handleChange}
              onBlur={field.onBlur}
              className={className}
              isInvalid={fieldState.invalid}
              autoComplete="off"
              aria-describedby={cn(description && descriptionId, fieldState.error && errorId)}
            />
          </FieldWrapper>
        );
      }}
    />
  );
}
