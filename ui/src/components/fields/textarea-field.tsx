import { cn } from "@/lib/utils";
import { TextareaFieldProps } from "@/types/fields";
import { Controller, FieldValues } from "react-hook-form";
import { Textarea } from "../ui/textarea";
import { FieldWrapper } from "./field-components";

export function TextareaField<T extends FieldValues>({
  label,
  description,
  name,
  control,
  rules,
  className,
  disabled,
  autoComplete,
  placeholder,
  "aria-label": ariaLabel,
  "aria-describedby": ariaDescribedBy,
  ...props
}: TextareaFieldProps<T>) {
  const inputId = `textarea-${name}`;
  const descriptionId = `${inputId}-description`;
  const errorId = `${inputId}-error`;

  return (
    <Controller<T>
      name={name}
      control={control}
      rules={rules}
      render={({ field, fieldState }) => (
        <FieldWrapper
          label={label}
          description={description}
          required={!!rules?.required}
          error={fieldState.error?.message}
          className={className}
        >
          <Textarea
            {...field}
            {...props}
            id={inputId}
            disabled={disabled}
            autoComplete={autoComplete}
            placeholder={placeholder}
            aria-label={ariaLabel || label}
            isInvalid={fieldState.invalid}
            aria-describedby={cn(
              description && descriptionId,
              fieldState.error && errorId,
              ariaDescribedBy,
            )}
          />
        </FieldWrapper>
      )}
    />
  );
}
