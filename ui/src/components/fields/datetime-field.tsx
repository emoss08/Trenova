import { toDate, toUnixTimeStamp } from "@/lib/date";
import { cn } from "@/lib/utils";
import { AutoCompleteDateFieldProps } from "@/types/fields";
import { useCallback } from "react";
import { Controller, FieldValues } from "react-hook-form";
import { DateTimePicker } from "./date-field/datetime-picker";
import { FieldWrapper } from "./field-components";

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
        const dateValue = field.value ? toDate(field.value) : undefined;

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
