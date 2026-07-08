import { cn } from "@/lib/utils";
import { EyeIcon, EyeOffIcon } from "lucide-react";
import { useState } from "react";
import { Controller, type FieldValues } from "react-hook-form";
import { Button } from "../ui/button";
import { Textarea } from "../ui/textarea";
import { FieldWrapper } from "./field-components";
import type { TextareaFieldProps } from "./textarea-field";

export function SensitiveTextareaField<T extends FieldValues>({
  label,
  description,
  name,
  control,
  rules,
  className,
  disabled,
  placeholder,
  "aria-label": ariaLabel,
  "aria-describedby": ariaDescribedBy,
  ...props
}: Omit<TextareaFieldProps<T>, "presets">) {
  const [show, setShow] = useState(false);
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
          <div className="relative">
            <Textarea
              {...field}
              {...props}
              id={inputId}
              className={cn(!show && "[-webkit-text-security:disc]", className)}
              disabled={disabled}
              minRows={3}
              autoComplete="off"
              spellCheck={false}
              placeholder={placeholder}
              aria-label={ariaLabel || label}
              isInvalid={fieldState.invalid}
              aria-describedby={cn(
                description && descriptionId,
                fieldState.error && errorId,
                ariaDescribedBy,
              )}
            />
            {!!field.value && (
              <Button
                type="button"
                variant="ghost"
                size="icon"
                className="absolute top-1 right-1 size-6 text-muted-foreground"
                title={show ? "Hide value" : "Show value"}
                onClick={() => setShow((current) => !current)}
              >
                {show ? <EyeOffIcon className="size-3.5" /> : <EyeIcon className="size-3.5" />}
              </Button>
            )}
          </div>
        </FieldWrapper>
      )}
    />
  );
}
