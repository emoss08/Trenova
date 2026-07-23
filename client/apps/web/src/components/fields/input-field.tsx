import type React from "react";
import type { FormControlProps } from "@trenova/shared/types/fields";
import { Controller, type FieldValues } from "react-hook-form";
import { Input, type InputProps } from "@trenova/shared/components/ui/input";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { FieldWrapper } from "./field-components";

type BaseInputFieldProps = Omit<InputProps, "name"> & {
  label?: React.ReactNode;
  description?: React.ReactNode;
  inputClassProps?: string;
  hideLabel?: boolean;
  maxLength?: number;
};

export type InputFieldProps<T extends FieldValues> = BaseInputFieldProps & FormControlProps<T>;

export function InputField<T extends FieldValues>({
  label,
  name,
  control,
  rules,
  description,
  className,
  inputClassProps,
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
      render={({ field, fieldState }) => {
        return (
          <FieldWrapper
            label={label}
            required={!!rules?.required}
            description={description}
            error={fieldState.error?.message}
            className={className}
          >
            <Input
              {...field}
              {...props}
              id={inputId}
              value={field.value ?? ""}
              onChange={field.onChange}
              className={inputClassProps}
              aria-invalid={fieldState.invalid}
              aria-describedby={(description && descriptionId) || (fieldState.error && errorId)}
            />
          </FieldWrapper>
        );
      }}
    />
  );
}

export function InputFieldSkeleton() {
  return (
    <div className="flex flex-col gap-0.5">
      <Skeleton className="flex h-4 w-37.5 items-center" />
      <Skeleton className="h-7 max-w-prose" />
      <Skeleton className="h-4 max-w-md" />
    </div>
  );
}
