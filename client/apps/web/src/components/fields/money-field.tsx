import { cn } from "@trenova/shared/lib/utils";
import type { FormControlProps } from "@trenova/shared/types/fields";
import { Controller, type FieldValues } from "react-hook-form";
import { NumericFormat } from "react-number-format";
import { FieldWrapper } from "./field-components";
import { fieldInvalidClass } from "@trenova/shared/lib/variants/field";

type BaseMoneyFieldProps = {
  label?: string;
  description?: string;
  className?: string;
  placeholder?: string;
  tabIndex?: number;
  disabled?: boolean;
  readOnly?: boolean;
  allowNegative?: boolean;
  "aria-label"?: string;
  onValueCommit?: (cents: number) => void;
};

export type MoneyFieldProps<T extends FieldValues> = BaseMoneyFieldProps & FormControlProps<T>;

export function MoneyField<T extends FieldValues>({
  name,
  control,
  rules,
  label,
  description,
  className,
  placeholder = "0.00",
  tabIndex,
  allowNegative = false,
  onValueCommit,
  ...props
}: MoneyFieldProps<T>) {
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
        >
          <NumericFormat
            value={typeof field.value === "number" ? field.value / 100 : ""}
            onValueChange={(values, sourceInfo) => {
              if (sourceInfo.source !== "event") return;
              const cents = values.floatValue == null ? 0 : Math.round(values.floatValue * 100);
              field.onChange(cents);
              onValueCommit?.(cents);
            }}
            onBlur={field.onBlur}
            getInputRef={field.ref}
            decimalScale={2}
            fixedDecimalScale
            thousandSeparator
            allowNegative={allowNegative}
            name={field.name}
            tabIndex={tabIndex}
            placeholder={placeholder}
            id={inputId}
            inputMode="decimal"
            disabled={props.disabled}
            readOnly={props.readOnly}
            aria-label={props["aria-label"] || label}
            aria-describedby={cn(description && descriptionId, fieldState.error && errorId)}
            className={cn(
              "ui-field flex h-7 w-full min-w-0 px-2 py-0.5 text-left tabular-nums outline-none md:text-xs",
              "placeholder:text-muted-foreground",
              "disabled:cursor-not-allowed disabled:opacity-50",
              "read-only:text-muted-foreground read-only:cursor-default",
"ui-focus-ring",
              props.readOnly && "pointer-events-none cursor-not-allowed opacity-60",
              fieldState.invalid &&
fieldInvalidClass,
              className,
            )}
          />
        </FieldWrapper>
      )}
    />
  );
}
