/**
 * # Copyright 2023-2025 Eric Moss
 * # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md
 */

import { cn } from "@/lib/utils";
import { CheckboxFieldProps } from "@/types/fields";
import { Controller, FieldValues } from "react-hook-form";
import { Checkbox } from "../ui/checkbox";
import { Label } from "../ui/label";

export function CheckboxField<T extends FieldValues>({
  label,
  description,
  name,
  control,
  rules,
  outlined,
  "aria-describedby": ariaDescribedBy,
  ...props
}: CheckboxFieldProps<T>) {
  const inputId = `checkbox-${name}`;
  const descriptionId = `${inputId}-description`;
  const errorId = `${inputId}-error`;

  return (
    <Controller
      name={name}
      control={control}
      rules={rules}
      render={({
        field: { value, onChange, disabled, onBlur, name, ref },
        fieldState,
      }) => (
        <div
          className={cn(
            "relative flex w-full items-start gap-2 rounded-md p-3",
            outlined &&
              "border border-muted-foreground/20 has-data-[state=checked]:border-blue-600 has-data-[state=checked]:ring-4 has-data-[state=checked]:ring-blue-600/20 bg-muted transition-[border-color,box-shadow] duration-200 ease-in-out",
          )}
        >
          <Checkbox
            id={inputId}
            aria-describedby={cn(
              description && descriptionId,
              fieldState.error && errorId,
              ariaDescribedBy,
            )}
            ref={ref}
            name={name}
            onBlur={onBlur}
            checked={value}
            onCheckedChange={onChange}
            disabled={disabled}
            className="order-1 after:absolute after:inset-0"
            onClick={(e) => e.stopPropagation()}
            {...props}
          />
          <div className="grid grow gap-2">
            <Label htmlFor={inputId}>{label}</Label>
            <p
              id={`${inputId}-description`}
              className="text-2xs text-muted-foreground"
            >
              {description}
            </p>
          </div>
        </div>
      )}
    />
  );
}
