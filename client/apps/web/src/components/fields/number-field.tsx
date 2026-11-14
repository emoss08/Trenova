import { isRequired, withRequired } from "@trenova/shared/lib/field-rules";
import { cn } from "@trenova/shared/lib/utils";
import {
  addDecimalStrings,
  compareDecimalStrings,
  formatDecimalString,
  isDecimalString,
} from "@trenova/shared/types/decimal";
import type { FormControlProps } from "@trenova/shared/types/fields";
import { useState } from "react";
import { Controller, type FieldPathValue, type FieldValues, type Path } from "react-hook-form";
import { NumericFormat } from "react-number-format";
import { FieldClearButton } from "./field-clear-button";
import { fieldEndSlotClass, fieldUnitYieldClass } from "./field-clear-reveal";
import { FieldStepper } from "./field-stepper";
import { FieldWrapper, type FieldLayout } from "./field-components";
import { fieldInvalidClass } from "@trenova/shared/lib/variants/field";

type BaseNumberFieldProps = {
  /**
   * Whether the field can be emptied with the control at its end. A field that is not
   * required can be, unless this says otherwise; a required one cannot.
   */
  clearable?: boolean;
  /** Maps a value that is not a plain number or decimal text in and out of the field. */
  codec?: NumberFieldCodec;
  label?: React.ReactNode;
  hideLabel?: boolean;
  /** A small control beside the label. */
  labelAction?: React.ReactNode;
  description?: React.ReactNode;
  layout?: FieldLayout;
  /** Drawn beside the input at its height, such as a unit select. */
  endAddon?: React.ReactNode;
  className?: string;
  /** Classes for the input alone; `className` also reaches the label's wrapper. */
  inputClassName?: string;
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
  /** The largest value: typing past it is refused, and stepping stops at it. */
  max?: number;
  step?: number;
  /** Draws the increment and decrement buttons; off for a narrow cell in a table row. */
  stepper?: boolean;
};

/**
 * How the field writes back into form state.
 *
 * Money and other exact quantities are decimal strings end to end — the schema
 * validates a string, the GraphQL `Decimal` scalar carries a string — so a
 * field bound to one has to hand back the digits it was given rather than a
 * float that has already lost them.
 */
export type NumberFieldValueType = "number" | "string";

/**
 * How a field bound to a value that is neither a number nor decimal text (a
 * Tariff wire quantity, `{"$q": "2.35", "u": "$/mi"}`) reads its digits out of
 * the value and writes typed digits back into one. `encode` is given the value
 * it replaces, so whatever else the value carries (its unit) survives an edit.
 */
export type NumberFieldCodec = {
  decode: (value: unknown) => string;
  encode: (text: string, previous: unknown) => unknown;
};

/**
 * Fields whose value can only be a string must say so; the choice is optional
 * only where the bound value could be either (a dynamic or untyped control).
 */
type ValueTypeProps<TValue> = [Extract<TValue, string>] extends [never]
  ? { valueType?: "number" }
  : [Extract<TValue, number>] extends [never]
    ? { valueType: "string" }
    : { valueType?: NumberFieldValueType };

export type NumberFieldProps<
  T extends FieldValues,
  TName extends Path<T> = Path<T>,
> = BaseNumberFieldProps &
  Omit<FormControlProps<T>, "name"> & { name: TName } & ValueTypeProps<FieldPathValue<T, TName>>;

type NumberFieldImplProps<T extends FieldValues> = BaseNumberFieldProps &
  FormControlProps<T> & { valueType?: NumberFieldValueType };

function textOf(value: unknown): string {
  if (typeof value === "string") return value;
  if (typeof value === "number" && Number.isFinite(value)) return String(value);
  return "";
}

function stepDecimalString(current: string, delta: number, scale: number): string {
  const base = isDecimalString(current) ? current : "0";
  return addDecimalStrings([base, delta.toString()], scale);
}

function clampDecimalString(
  value: string,
  bound: number,
  direction: "min" | "max",
  scale: number,
): string {
  const limit = bound.toString();
  const comparison = compareDecimalStrings(value, limit);
  const outside = direction === "max" ? comparison > 0 : comparison < 0;
  return outside ? formatDecimalString(limit, scale) : value;
}

function NumberFieldImpl<T extends FieldValues>({
  name,
  control,
  description,
  label,
  hideLabel,
  labelAction,
  className,
  inputClassName,
  placeholder = "Enter Valid Number",
  sideText,
  layout,
  endAddon,
  rules,
  required,
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
  stepper = true,
  valueType,
  codec,
  clearable,
  ...props
}: NumberFieldImplProps<T>) {
  // What was typed into a codec field and the value it wrote. The value holds the
  // number so far (`2.` is still 2), so the text is shown back for as long as the
  // value is the one it wrote; a value set from outside replaces it.
  const [draft, setDraft] = useState<{ text: string; written: string } | null>(null);

  const inputId = `input-${name}`;
  const descriptionId = `${inputId}-description`;
  const errorId = `${inputId}-error`;

  return (
    <Controller<T>
      name={name}
      control={control}
      rules={withRequired(rules, required)}
      render={({ field, fieldState }) => {
        // An untyped control (custom fields, dynamic rows) declares nothing, so
        // fall back to the kind the value already has and keep it that kind.
        const asString =
          codec !== undefined ||
          (valueType ? valueType === "string" : typeof field.value === "string");
        const decoded = codec ? codec.decode(field.value) : textOf(field.value);
        const text =
          codec && draft && draft.written === JSON.stringify(field.value ?? null)
            ? draft.text
            : decoded;
        const write = (next: string) => {
          if (!codec) {
            field.onChange(next);
            return;
          }
          const encoded = codec.encode(next, field.value);
          setDraft({ text: next, written: JSON.stringify(encoded ?? null) });
          field.onChange(encoded);
        };
        const currentValue = typeof field.value === "number" ? field.value : 0;

        const stepBy = (delta: number, bound: number | undefined, direction: "min" | "max") => {
          if (asString) {
            const next = stepDecimalString(text, delta, decimalScale);
            write(
              bound === undefined ? next : clampDecimalString(next, bound, direction, decimalScale),
            );
            return;
          }

          const next = currentValue + delta;
          if (bound === undefined) {
            field.onChange(next);
            return;
          }
          field.onChange(direction === "max" ? Math.min(next, bound) : Math.max(next, bound));
        };

        const empty = () => {
          if (codec) write("");
          else field.onChange(asString ? "" : null);
        };
        const clear =
          (clearable ?? !isRequired(rules, required)) &&
          !props.disabled &&
          !props.readOnly &&
          field.value != null &&
          field.value !== "";

        return (
          <FieldWrapper
            name={name}
            label={label}
            hideLabel={hideLabel}
            labelAction={labelAction}
            description={description}
            required={isRequired(rules, required)}
            error={fieldState.error?.message}
            className={className}
            layout={layout}
            controlSize="compact"
            htmlFor={inputId}
          >
            <div className={cn(endAddon !== undefined && "flex items-start gap-1.5")}>
              <div className="relative min-w-0 flex-1">
                <NumericFormat
                  value={codec ? text : (field.value ?? "")}
                  onValueChange={(values, source) => {
                    if (codec) {
                      // A value set from outside (a reset) re-renders the digits it
                      // already holds; writing them back would only churn the form.
                      if (source.source === "event") write(values.value);
                      return;
                    }
                    field.onChange(asString ? values.value : (values.floatValue ?? null));
                  }}
                  isAllowed={
                    max === undefined
                      ? undefined
                      : (values) => values.floatValue === undefined || values.floatValue <= max
                  }
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
                  aria-label={props["aria-label"]}
                  aria-describedby={cn(
                    description && descriptionId,
                    fieldState.error && errorId,
                    props["aria-describedby"],
                  )}
                  className={cn(
                    "flex h-7 w-full min-w-0 ui-field px-2 py-0.5 outline-none md:text-xs",
                    "file:border-0 file:bg-transparent file:text-sm file:font-medium",
                    "placeholder:text-muted-foreground",
                    "disabled:cursor-not-allowed disabled:opacity-50",
                    "read-only:cursor-default read-only:text-muted-foreground",
                    "ui-focus-ring",
                    props.readOnly && "pointer-events-none cursor-not-allowed opacity-60",
                    fieldState.invalid && fieldInvalidClass,
                    !stepper ? (sideText ? "pr-10" : "pr-2") : sideText ? "pr-16" : "pr-12",
                    clear && !sideText && (stepper ? "pr-18" : "pr-8"),
                    className,
                    inputClassName,
                  )}
                />

                <div className="absolute top-px right-px bottom-px flex items-center pr-0">
                  {sideText ? (
                    // The unit and the clear control share this place: the unit shows
                    // until the field is hovered or focused, then gives way to it. The
                    // control is the last thing before the field's edge (or its stepper):
                    // the unit keeps its own inset, the control has almost none.
                    <div className={cn(fieldEndSlotClass, "mr-0.5")}>
                      <span
                        className={cn(
                          "pointer-events-none mr-1.5 text-xs text-muted-foreground select-none",
                          clear && fieldUnitYieldClass,
                        )}
                      >
                        {sideText}
                      </span>
                      {clear ? <FieldClearButton overUnit onClear={empty} /> : null}
                    </div>
                  ) : clear ? (
                    <FieldClearButton className="mr-0.5" onClear={empty} />
                  ) : null}
                  {stepper ? (
                    <FieldStepper
                      disabled={props.disabled || props.readOnly}
                      onIncrement={() => stepBy(step, max, "max")}
                      onDecrement={() => stepBy(-step, min, "min")}
                    />
                  ) : null}
                </div>
              </div>
              {endAddon}
            </div>
          </FieldWrapper>
        );
      }}
    />
  );
}

export const NumberField = NumberFieldImpl as <
  T extends FieldValues,
  TName extends Path<T> = Path<T>,
>(
  props: NumberFieldProps<T, TName>,
) => React.ReactElement;
