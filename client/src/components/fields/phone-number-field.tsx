import type { FormControlProps } from "@/types/fields";
import { Controller, type FieldValues } from "react-hook-form";
import { PhoneInput, type PhoneInputProps } from "../ui/phone-input";
import { FieldWrapper } from "./field-components";

type BasePhoneNumberFieldProps = Omit<PhoneInputProps, "name"> & {
  label?: string;
  description?: string;
  inputClassProps?: string;
};

export type PhoneNumberFieldProps<T extends FieldValues> =
  BasePhoneNumberFieldProps & FormControlProps<T>;

export function PhoneNumberField<T extends FieldValues>({
  label,
  name,
  control,
  rules,
  description,
  className,
  inputClassProps,
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
          label={label}
          description={description}
          required={!!rules?.required}
          error={fieldState.error?.message}
          className={className}
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
