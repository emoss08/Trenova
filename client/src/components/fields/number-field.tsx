import { cn } from "@/lib/utils";
import type { FormControlProps } from "@/types/fields";
import { ChevronDownIcon, ChevronUpIcon } from "lucide-react";
import { Controller, type FieldValues } from "react-hook-form";
import { NumericFormat } from "react-number-format";
import { FieldWrapper } from "./field-components";

type BaseNumberFieldProps = {
  label?: string;
  description?: string;
  className?: string;
  placeholder?: string;
  sideText?: string;
  tabIndex?: number;
  disabled?: boolean;
  readOnly?: boolean;
  "aria-label"?: string;
  "aria-describedby"?: string;
  decimalScale?: number;
  fixedDecimalScale?: boolean;
  allowNegative?: boolean;
  thousandSeparator?: boolean | string;
  prefix?: string;
  suffix?: string;
  min?: number;
  max?: number;
  step?: number;
};

export type NumberFieldProps<T extends FieldValues> = BaseNumberFieldProps &
  FormControlProps<T>;

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
  decimalScale = 0,
  fixedDecimalScale = false,
  allowNegative = false,
  thousandSeparator,
  prefix,
  suffix,
  min,
  max,
  step = 1,
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
      render={({ field, fieldState }) => {
        const currentValue =
          typeof field.value === "number" ? field.value : 0;

        return (
          <FieldWrapper
            label={label}
            description={description}
            required={!!rules?.required}
            error={fieldState.error?.message}
            className={className}
          >
            <div className="relative w-full">
              <NumericFormat
                value={field.value ?? ""}
                onValueChange={(values) => {
                  field.onChange(values.floatValue ?? null);
                }}
                onBlur={field.onBlur}
                getInputRef={field.ref}
                decimalScale={decimalScale}
                fixedDecimalScale={fixedDecimalScale}
                allowNegative={allowNegative}
                thousandSeparator={thousandSeparator}
                prefix={prefix}
                suffix={suffix}
                name={field.name}
                tabIndex={tabIndex}
                placeholder={placeholder}
                id={inputId}
                disabled={props.disabled}
                readOnly={props.readOnly}
                aria-label={props["aria-label"] || label}
                aria-describedby={cn(
                  description && descriptionId,
                  fieldState.error && errorId,
                  props["aria-describedby"],
                )}
                className={cn(
                  "flex h-8 w-full min-w-0 rounded-md border border-input bg-muted px-2.5 py-1 outline-none md:text-xs",
                  "file:border-0 file:bg-transparent file:text-sm file:font-medium",
                  "placeholder:text-muted-foreground",
                  "disabled:cursor-not-allowed disabled:opacity-50",
                  "read-only:cursor-default read-only:text-muted-foreground",
                  "focus-visible:border-brand focus-visible:ring-4 focus-visible:ring-brand/20 focus-visible:outline-hidden",
                  "transition-[border-color,box-shadow] duration-200 ease-in-out",
                  props.readOnly &&
                    "pointer-events-none cursor-not-allowed opacity-60",
                  fieldState.invalid &&
                    "border-red-500 bg-red-500/20 ring-0 ring-red-500 placeholder:text-red-500 focus:outline-hidden focus-visible:border-red-600 focus-visible:ring-4 focus-visible:ring-red-400/20",
                  sideText ? "pr-16" : "pr-12",
                  className,
                )}
              />

              <div className="absolute top-px right-px bottom-px flex items-center gap-1 pr-0">
                {sideText && (
                  <div className="pointer-events-none mr-2 text-xs text-muted-foreground select-none">
                    {sideText}
                  </div>
                )}
                <div className="flex h-full flex-col items-stretch rounded-r-md border-l border-muted-foreground/20 bg-transparent">
                  <button
                    type="button"
                    aria-label="Increment"
                    className="inline-flex w-7 flex-1 items-center justify-center border-b border-muted-foreground/20 text-muted-foreground hover:bg-muted-foreground/10 hover:text-foreground disabled:opacity-50"
                    disabled={props.disabled || props.readOnly}
                    onClick={() => {
                      let next = currentValue + step;
                      if (max !== undefined) next = Math.min(next, max);
                      field.onChange(next);
                    }}
                  >
                    <ChevronUpIcon className="h-3 w-3" />
                  </button>
                  <button
                    type="button"
                    aria-label="Decrement"
                    className="inline-flex w-7 flex-1 items-center justify-center text-muted-foreground hover:bg-muted-foreground/10 hover:text-foreground disabled:opacity-50"
                    disabled={props.disabled || props.readOnly}
                    onClick={() => {
                      let next = currentValue - step;
                      if (min !== undefined) next = Math.max(next, min);
                      field.onChange(next);
                    }}
                  >
                    <ChevronDownIcon className="h-3 w-3" />
                  </button>
                </div>
              </div>
            </div>
          </FieldWrapper>
        );
      }}
    />
  );
}
