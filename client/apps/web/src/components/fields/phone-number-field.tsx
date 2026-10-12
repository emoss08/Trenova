import type { FormControlProps } from "@trenova/shared/types/fields";
import { Controller, type FieldValues } from "react-hook-form";
import { PhoneInput, type PhoneInputProps } from "@trenova/shared/components/ui/phone-input";
import type React from "react";
import { FieldWrapper, type FieldLayout } from "./field-components";

type BasePhoneNumberFieldProps = Omit<PhoneInputProps, "name"> & {
  label?: string;
  description?: React.ReactNode;
  inputClassProps?: string;
  layout?: FieldLayout;
};

export type PhoneNumberFieldProps<T extends FieldValues> = BasePhoneNumberFieldProps &
  FormControlProps<T>;

export function PhoneNumberField<T extends FieldValues>({
  label,
  name,
  control,
  rules,
  description,
  className,
  inputClassProps,
  layout,
  ...props
}: PhoneNumberFieldProps<T>) {
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
          name={name}
          label={label}
          description={description}
          required={!!rules?.required}
          error={fieldState.error?.message}
          className={className}
          layout={layout}
        >
          <PhoneInput
            {...field}
            {...props}
            id={inputId}
            value={field.value ?? undefined}
            className={inputClassProps}
            onChange={field.onChange}
            aria-invalid={fieldState.invalid}
            aria-describedby={descriptionId || errorId}
            defaultCountry="US"
          />
        </FieldWrapper>
      )}
    />
  );
}
