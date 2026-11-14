import { isRequired, withRequired } from "@trenova/shared/lib/field-rules";
import {
  amountSeparators,
  amountTextToMinor,
  currencySymbol,
  minorToAmountText,
} from "@trenova/shared/lib/money";
import { cn } from "@trenova/shared/lib/utils";
import { fieldInvalidClass } from "@trenova/shared/lib/variants/field";
import type { FormControlProps } from "@trenova/shared/types/fields";
import type { ReactNode } from "react";
import { Controller, type FieldValues } from "react-hook-form";
import { NumericFormat } from "react-number-format";
import { FieldClearButton } from "./field-clear-button";
import { FieldWrapper, type FieldLayout } from "./field-components";
import { FieldStepper } from "./field-stepper";

/** One whole unit of currency in minor units: what a press of the stepper moves by. */
const ONE_UNIT_MINOR = 100;

type BaseMoneyFieldProps = {
  label?: string;
  hideLabel?: boolean;
  /** A small control beside the label. */
  labelAction?: ReactNode;
  description?: string;
  layout?: FieldLayout;
  /** Classes for the field's box: its height and type size. */
  className?: string;
  placeholder?: string;
  tabIndex?: number;
  disabled?: boolean;
  readOnly?: boolean;
  allowNegative?: boolean;
  /**
   * Shows a stored 0 as "0.00". Left out, a field that can be typed in is drawn empty at 0,
   * with its placeholder, which can say what 0 means there ("Automatic"). A field that is
   * read-only or disabled is showing a figure, and always shows it.
   */
  showZero?: boolean;
  /**
   * Draws the amount as one cell in a column of amounts (a journal's debits and credits):
   * its digits against the right edge, with no currency mark, stepper or clear control, and
   * nothing at all in a cell nobody filled. The column's heading and total say what the
   * figures are, so the cell is only its digits; it is emptied by deleting them.
   */
  ledger?: boolean;
  /** The ISO code of the currency the amount is in. US dollars when left out. */
  currency?: string | null;
  /** The buttons at the field's end that move the amount up and down. On unless turned off. */
  stepper?: boolean;
  /** How far one press of the stepper moves the amount, in minor units. One whole unit by default. */
  step?: number;
  /**
   * Whether the field can be emptied with the control at its end. A field that is not
   * required can be, unless this says otherwise; a required one cannot.
   */
  clearable?: boolean;
  "aria-label"?: string;
  /** Told the amount, in minor units, each time the person changes it. */
  onAmountChange?: (minor: number) => void;
};

export type MoneyFieldProps<T extends FieldValues> = BaseMoneyFieldProps & FormControlProps<T>;

/**
 * An amount of money, held by the form in minor units (cents) and typed in whole units. It
 * shows its currency's mark, writes its digits the way the reader's locale does, and has
 * what a number field has: a stepper, and a control that empties it when it is not required.
 * In a column of amounts it is a `ledger` cell instead, which is its digits and nothing else.
 *
 * The form's value is always a number, so an emptied field holds 0, and the form cannot
 * tell an amount nobody entered from a zero somebody did. A field that can be typed in is
 * therefore drawn empty at 0 rather than claiming "0.00" was entered; emptying it and
 * finding it empty are then the same thing. The amount goes to and from the text by its
 * digits, never through a float: 0.29 typed is 29, not 28.999….
 */
export function MoneyField<T extends FieldValues>({
  name,
  control,
  rules,
  required,
  label,
  hideLabel,
  labelAction,
  description,
  layout,
  className,
  ledger = false,
  placeholder = ledger ? "" : "0.00",
  tabIndex,
  disabled,
  readOnly,
  allowNegative = false,
  showZero = false,
  currency,
  stepper = !ledger,
  step = ONE_UNIT_MINOR,
  clearable,
  onAmountChange,
  ...props
}: MoneyFieldProps<T>) {
  const inputId = `input-${name}`;
  const descriptionId = `${inputId}-description`;
  const errorId = `${inputId}-error`;
  const locked = Boolean(disabled || readOnly);
  const canClear = (clearable ?? (!ledger && !isRequired(rules, required))) && !locked;
  const hidesZero = !showZero && !locked;
  const symbol = ledger ? "" : currencySymbol(currency);
  const { decimal, group } = amountSeparators();

  return (
    <Controller<T>
      name={name}
      control={control}
      rules={withRequired(rules, required)}
      render={({ field, fieldState }) => {
        const amount = typeof field.value === "number" ? field.value : 0;
        const write = (minor: number) => {
          field.onChange(minor);
          onAmountChange?.(minor);
        };
        const stepBy = (delta: number) => {
          const next = amount + delta;
          write(allowNegative ? next : Math.max(0, next));
        };

        return (
          <FieldWrapper
            name={name}
            label={label}
            hideLabel={hideLabel}
            labelAction={labelAction}
            description={description}
            required={isRequired(rules, required)}
            error={fieldState.error?.message}
            layout={layout}
            controlSize="compact"
            htmlFor={inputId}
            descriptionId={descriptionId}
            errorId={errorId}
          >
            <div
              className={cn(
                "flex h-7 w-full min-w-0 items-center ui-field ui-container-focus-ring",
                disabled && "cursor-not-allowed opacity-50",
                readOnly && "pointer-events-none opacity-60",
                fieldState.invalid && fieldInvalidClass,
                className,
              )}
            >
              {symbol ? (
                <span aria-hidden className="pl-2 text-xs text-muted-foreground select-none">
                  {symbol}
                </span>
              ) : null}
              <NumericFormat
                value={amount === 0 && hidesZero ? "" : minorToAmountText(amount)}
                valueIsNumericString
                onValueChange={(values, source) => {
                  // A value set from outside (a reset, the stepper) redraws the digits the
                  // form already holds; writing them back would only churn it.
                  if (source.source === "event") write(amountTextToMinor(values.value));
                }}
                onBlur={field.onBlur}
                getInputRef={field.ref}
                decimalScale={2}
                fixedDecimalScale
                decimalSeparator={decimal}
                thousandSeparator={group || false}
                allowNegative={allowNegative}
                name={field.name}
                tabIndex={tabIndex}
                placeholder={placeholder}
                id={inputId}
                inputMode="decimal"
                disabled={disabled}
                readOnly={readOnly}
                aria-label={props["aria-label"] || label}
                aria-invalid={fieldState.invalid}
                aria-describedby={
                  cn(description && descriptionId, fieldState.error && errorId) || undefined
                }
                className={cn(
                  "h-full min-w-0 flex-1 bg-transparent py-0.5 tabular-nums outline-none md:text-xs",
                  ledger ? "px-2 text-right" : "px-1.5 text-left",
                  "placeholder:text-muted-foreground",
                  "read-only:cursor-default read-only:text-muted-foreground disabled:cursor-not-allowed",
                )}
              />
              {canClear && amount !== 0 ? (
                <FieldClearButton className="mr-0.5" onClear={() => write(0)} />
              ) : null}
              {stepper ? (
                <FieldStepper
                  disabled={locked}
                  onIncrement={() => stepBy(step)}
                  onDecrement={() => stepBy(-step)}
                />
              ) : null}
            </div>
          </FieldWrapper>
        );
      }}
    />
  );
}
