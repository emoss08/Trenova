import { isRequired, withRequired } from "@trenova/shared/lib/field-rules";
import type React from "react";
import type { FormControlProps } from "@trenova/shared/types/fields";
import { Controller, type FieldValues } from "react-hook-form";
import { Input, type InputProps } from "@trenova/shared/components/ui/input";
import { cn } from "@trenova/shared/lib/utils";
import { FieldClearButton } from "./field-clear-button";
import { fieldEndSlotClass, fieldUnitYieldClass } from "./field-clear-reveal";
import { FieldWrapper, type FieldLayout } from "./field-components";

type BaseInputFieldProps = Omit<InputProps, "name" | "required"> & {
  label?: React.ReactNode;
  /** A small control beside the label. */
  labelAction?: React.ReactNode;
  description?: React.ReactNode;
  inputClassProps?: string;
  hideLabel?: boolean;
  maxLength?: number;
  layout?: FieldLayout;
  /**
   * Whether the field can be emptied with the control at its end. A field that is not
   * required can be, unless this says otherwise; a required one cannot.
   */
  clearable?: boolean;
};

export type InputFieldProps<T extends FieldValues> = BaseInputFieldProps & FormControlProps<T>;

export function InputField<T extends FieldValues>({
  label,
  hideLabel,
  labelAction,
  name,
  control,
  rules,
  required,
  description,
  className,
  inputClassProps,
  layout,
  clearable,
  rightElement,
  sideText,
  ...props
}: InputFieldProps<T>) {
  const inputId = `input-${name}`;
  const descriptionId = `${inputId}-description`;
  const errorId = `${inputId}-error`;
  const canClear =
    (clearable ?? !isRequired(rules, required)) && !props.disabled && !props.readOnly;

  return (
    <Controller<T>
      name={name}
      control={control}
      rules={withRequired(rules, required)}
      render={({ field, fieldState }) => {
        const clears = canClear && field.value != null && field.value !== "";
        // With a unit beside it ("minutes", "%") the clear control takes the unit's place
        // while the field is hovered or focused, rather than crowding in beside it. It is
        // the last thing in the field, hard against its right edge: the unit keeps its own
        // inset, the control has none.
        const end = clears ? (
          <>
            {rightElement}
            {sideText ? (
              <span className={fieldEndSlotClass}>
                <span
                  aria-hidden
                  className={cn("pr-1.5 text-xs text-muted-foreground", fieldUnitYieldClass)}
                >
                  {sideText}
                </span>
                <FieldClearButton overUnit onClear={() => field.onChange("")} />
              </span>
            ) : (
              <FieldClearButton onClear={() => field.onChange("")} />
            )}
          </>
        ) : (
          rightElement
        );

        return (
          <FieldWrapper
            name={name}
            label={label}
            hideLabel={hideLabel}
            labelAction={labelAction}
            required={isRequired(rules, required)}
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
              rightElement={end}
              sideText={clears ? undefined : sideText}
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
