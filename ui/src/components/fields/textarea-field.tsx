/**
 * # Copyright 2023-2025 Eric Moss
 * # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md
 */

import { cn } from "@/lib/utils";
import { TextareaFieldProps } from "@/types/fields";
import { Controller, FieldValues } from "react-hook-form";
import { AITextarea, AutoResizeTextarea, Textarea } from "../ui/textarea";
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
        >
          <Textarea
            {...field}
            {...props}
            id={inputId}
            className={className}
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

export function AutoResizeTextareaField<T extends FieldValues>({
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
        >
          <AutoResizeTextarea
            {...field}
            {...props}
            id={inputId}
            disabled={disabled}
            autoComplete={autoComplete}
            className={className}
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

export function AITextareaField<T extends FieldValues>({
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
          <AITextarea
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
