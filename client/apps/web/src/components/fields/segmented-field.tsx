import { SegmentedControl } from "@trenova/shared/components/ui/segmented-control";
import type { FormControlProps } from "@trenova/shared/types/fields";
import type React from "react";
import { Controller, type FieldValues } from "react-hook-form";
import { FieldWrapper, type FieldLayout } from "./field-components";

export type SegmentedFieldOption<TValue extends string | number> = {
  value: TValue;
  label: React.ReactNode;
  disabled?: boolean;
};

type SegmentedFieldProps<
  T extends FieldValues,
  TValue extends string | number,
> = FormControlProps<T> & {
  label: string;
  description?: React.ReactNode;
  options: readonly SegmentedFieldOption<TValue>[];
  /** Stretches the segments to the field's width. */
  fullWidth?: boolean;
  className?: string;
  layout?: FieldLayout;
};

/**
 * A choice of a few, every one shown at once, bound to a form field. For two to
 * four options where a select would hide what the others are. A numeric choice
 * is written back as the option's number, not the text the control keys it by.
 */
export function SegmentedField<T extends FieldValues, TValue extends string | number>({
  name,
  control,
  rules,
  label,
  description,
  options,
  fullWidth = true,
  className,
  layout,
}: SegmentedFieldProps<T, TValue>) {
  const fieldId = `segmented-${name}`;

  return (
    <Controller
      name={name}
      control={control}
      rules={rules}
      render={({ field, fieldState }) => (
        <FieldWrapper
          name={name}
          label={label}
          required={!!rules?.required}
          description={description}
          error={fieldState.error?.message}
          descriptionId={`${fieldId}-description`}
          errorId={`${fieldId}-error`}
          className={className}
          layout={layout}
          formatValue={(selected) =>
            options.find((option) => String(option.value) === String(selected))?.label
          }
        >
          <SegmentedControl<string>
            aria-label={label}
            fullWidth={fullWidth}
            value={String(field.value)}
            onValueChange={(key) =>
              field.onChange(options.find((option) => String(option.value) === key)?.value)
            }
            items={options.map((option) => ({
              value: String(option.value),
              label: option.label,
              disabled: option.disabled || field.disabled,
            }))}
          />
        </FieldWrapper>
      )}
    />
  );
}
