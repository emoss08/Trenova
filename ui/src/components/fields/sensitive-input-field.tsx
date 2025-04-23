import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { InputFieldProps } from "@/types/fields";
import { faEye, faEyeSlash } from "@fortawesome/pro-regular-svg-icons";
import { useState } from "react";
import { Controller, FieldValues } from "react-hook-form";
import { Icon } from "../ui/icons";
import { Input } from "../ui/input";
import { FieldWrapper, PasswordFieldWrapper } from "./field-components";

export function SensitiveInputField<T extends FieldValues>({
  label,
  description,
  icon,
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
}: InputFieldProps<T>) {
  const inputId = `input-${name}`;
  const descriptionId = `${inputId}-description`;
  const errorId = `${inputId}-error`;
  const [showPassword, setShowPassword] = useState(false);

  const togglePasswordVisibility = () => {
    setShowPassword(!showPassword);
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
              type={showPassword ? "text" : "password"}
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
              rightElement={
                field.value && !fieldState.invalid ? (
                  <Button
                    type="button"
                    size="icon"
                    variant="ghost"
                    className="size-5 mr-0.5 rounded-sm"
                    onClick={togglePasswordVisibility}
                  >
                    {showPassword ? (
                      <Icon icon={faEyeSlash} className="size-3.5" />
                    ) : (
                      <Icon icon={faEye} className="size-3.5" />
                    )}
                  </Button>
                ) : undefined
              }
            />
          </div>
        </FieldWrapper>
      )}
    />
  );
}

type PasswordFieldProps<T extends FieldValues> = InputFieldProps<T> & {
  onPasswordReset: () => void;
};

export function PasswordField<T extends FieldValues>({
  label,
  description,
  icon,
  name,
  control,
  rules,
  className,
  disabled,
  autoComplete,
  placeholder,
  "aria-label": ariaLabel,
  "aria-describedby": ariaDescribedBy,
  onPasswordReset,
  ...props
}: PasswordFieldProps<T>) {
  const inputId = `input-${name}`;
  const descriptionId = `${inputId}-description`;
  const errorId = `${inputId}-error`;
  const [showPassword, setShowPassword] = useState(false);

  const togglePasswordVisibility = () => {
    setShowPassword(!showPassword);
  };

  const handlePasswordReset = () => {
    onPasswordReset();
  };

  return (
    <Controller<T>
      name={name}
      control={control}
      rules={rules}
      render={({ field, fieldState }) => (
        <PasswordFieldWrapper
          label={label}
          description={description}
          required={!!rules?.required}
          error={fieldState.error?.message}
          className={className}
          onPasswordReset={handlePasswordReset}
        >
          <div className="relative">
            <Input
              {...field}
              {...props}
              id={inputId}
              type={showPassword ? "text" : "password"}
              disabled={disabled}
              autoComplete={autoComplete}
              placeholder={placeholder}
              aria-label={ariaLabel || label}
              isInvalid={fieldState.invalid}
              icon={icon}
              className="items-center justify-center"
              aria-describedby={cn(
                description && descriptionId,
                fieldState.error && errorId,
                ariaDescribedBy,
              )}
              rightElement={
                field.value && !fieldState.invalid ? (
                  <Button
                    type="button"
                    size="icon"
                    variant="ghost"
                    className="size-5 mr-0.5 rounded-sm"
                    onClick={togglePasswordVisibility}
                  >
                    {showPassword ? (
                      <Icon icon={faEyeSlash} className="size-3" />
                    ) : (
                      <Icon icon={faEye} className="size-3" />
                    )}
                  </Button>
                ) : undefined
              }
            />
          </div>
        </PasswordFieldWrapper>
      )}
    />
  );
}
