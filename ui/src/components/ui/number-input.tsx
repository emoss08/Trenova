/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import { cn } from "@/lib/utils";
import type { NumberFieldProps } from "@/types/fields";
import { Controller, type FieldValues } from "react-hook-form";
import { FieldWrapper } from "../fields/field-components";
import { Input } from "./input";

export function NumberField<T extends FieldValues>({
  name,
  control,
  description,
  label,
  className,
  placeholder = "Enter Valid Number",
  sideText,
  rules,
  tabIndex,
  ...props
}: NumberFieldProps<T>) {
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
            tabIndex={tabIndex}
            type="number"
            placeholder={placeholder}
            id={inputId}
            className={className}
            disabled={props.disabled}
            aria-label={props["aria-label"] || label}
            aria-describedby={cn(
              description && descriptionId,
              fieldState.error && errorId,
              props["aria-describedby"],
            )}
            isInvalid={fieldState.invalid}
            rightElement={
              sideText && (
                <div className="pointer-events-none absolute inset-y-0 right-0 flex items-center pr-3 text-xs text-muted-foreground">
                  {sideText}
                </div>
              )
            }
          />
        </FieldWrapper>
      )}
    />
  );
}
