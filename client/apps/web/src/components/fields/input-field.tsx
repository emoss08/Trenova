import type React from "react";
import type { FormControlProps } from "@trenova/shared/types/fields";
import { Controller, type FieldValues } from "react-hook-form";
import { Input, type InputProps } from "@trenova/shared/components/ui/input";
import { FieldWrapper, type FieldLayout } from "./field-components";

type BaseInputFieldProps = Omit<InputProps, "name"> & {
  label?: React.ReactNode;
  /** A small control beside the label. */
  labelAction?: React.ReactNode;
  description?: React.ReactNode;
  inputClassProps?: string;
  hideLabel?: boolean;
  maxLength?: number;
  layout?: FieldLayout;
};

export type InputFieldProps<T extends FieldValues> = BaseInputFieldProps & FormControlProps<T>;

export function InputField<T extends FieldValues>({
  label,
  labelAction,
  name,
  control,
  rules,
  description,
  className,
  inputClassProps,
  layout,
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
            name={name}
            label={label}
            labelAction={labelAction}
            required={!!rules?.required}
            description={description}
            error={fieldState.error?.message}
            className={className}
            layout={layout}
            htmlFor={inputId}
          >
            <Input
              {...field}
              {...props}
              id={inputId}
              value={field.value ?? ""}
              onChange={field.onChange}
              className={inputClassProps}
              aria-label={props["aria-label"] ?? (typeof label === "string" ? label : undefined)}
              aria-invalid={fieldState.invalid}
              aria-describedby={(description && descriptionId) || (fieldState.error && errorId)}
            />
          </FieldWrapper>
        );
      }}
    />
  );
}

export { InputFieldSkeleton } from "./field-skeletons";
