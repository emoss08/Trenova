import { useState } from "react";
import { Controller, type FieldValues } from "react-hook-form";
import { Input } from "../ui/input";
import { FieldWrapper } from "./field-components";
import type { InputFieldProps } from "./input-field";

export function SensitiveField<T extends FieldValues>({
  label,
  description,
  name,
  control,
  rules,
  className,
  ...props
}: InputFieldProps<T>) {
  const inputId = `input-${name}`;
  const descriptionId = `${inputId}-description`;
  const errorId = `${inputId}-error`;
  const [show, setShow] = useState(false);

  const togglePasswordVisibility = () => {
    setShow(!show);
  };

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
          <div className="relative">
            <Input
              {...field}
              {...props}
              id={inputId}
              type={show ? "text" : "password"}
              aria-describedby={
                (description && descriptionId) || (fieldState.error && errorId)
              }
              aria-invalid={fieldState.invalid}
              rightElement={
                field.value && !fieldState.invalid ? (
                  <button
                    type="button"
                    className="size-full cursor-pointer px-2 py-1 text-xs text-muted-foreground"
                    onClick={togglePasswordVisibility}
                  >
                    {show ? "hide" : "show"}
                  </button>
                ) : undefined
              }
            />
          </div>
        </FieldWrapper>
      )}
    />
  );
}
