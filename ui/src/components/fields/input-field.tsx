import { cn } from "@/lib/utils";
import { InputFieldProps } from "@/types/fields";
import { Controller, FieldValues } from "react-hook-form";
import { Input } from "../ui/input";
import { FieldWrapper } from "./field-components";

export function InputField<T extends FieldValues>({
  label,
  description,
  icon,
  name,
  control,
  rules,
  className,
  type = "text",
  disabled,
  autoComplete,
  placeholder,
  "aria-label": ariaLabel,
  "aria-describedby": ariaDescribedBy,
  ...props
}: InputFieldProps<T>) {
  const inputId = `input-${name}`;
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
          <Input
            {...field}
            {...props}
            id={inputId}
            type={type}
            disabled={disabled}
            autoComplete={autoComplete}
            placeholder={placeholder}
            aria-label={ariaLabel || label}
            isInvalid={fieldState.invalid}
            icon={icon}
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
