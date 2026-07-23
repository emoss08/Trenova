import { cn } from "@trenova/shared/lib/utils";
import type { FormControlProps } from "@trenova/shared/types/fields";
import { Controller, type FieldValues } from "react-hook-form";
import { NumericFormat } from "react-number-format";
import { FieldWrapper } from "./field-components";

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
              "flex h-7 w-full min-w-0 rounded-md border border-input bg-muted px-2 py-0.5 text-right tabular-nums outline-none md:text-xs",
              "placeholder:text-muted-foreground",
              "disabled:cursor-not-allowed disabled:opacity-50",
              "read-only:cursor-default read-only:text-muted-foreground",
              "focus-visible:border-brand focus-visible:ring-4 focus-visible:ring-brand/20 focus-visible:outline-hidden",
              "transition-[border-color,box-shadow] duration-200 ease-in-out",
              props.readOnly && "pointer-events-none cursor-not-allowed opacity-60",
              fieldState.invalid &&
                "border-red-500 bg-red-500/20 ring-0 ring-red-500 placeholder:text-red-500 focus:outline-hidden focus-visible:border-red-600 focus-visible:ring-4 focus-visible:ring-red-400/20",
              className,
            )}
          />
        </FieldWrapper>
      )}
    />
  );
}
