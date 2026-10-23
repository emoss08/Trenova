import { FieldWrapper } from "@/components/fields/field-components";
import { InputField } from "@/components/fields/input-field";
import { NumberField, type NumberFieldCodec } from "@/components/fields/number-field";
import { SelectField } from "@/components/fields/select-field";
import { SwitchField } from "@/components/fields/switch-field";
import { TextChipsField } from "@/components/fields/text-chips-field";
import { Input } from "@trenova/shared/components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@trenova/shared/components/ui/select";
import { useT } from "@trenova/shared/i18n/use-t";
import {
  isTariffDate,
  isTariffNumber,
  isTariffQuantity,
  tariffDate,
  tariffDecimal,
  tariffNumber,
  tariffQuantity,
  tariffUnitsOf,
} from "@trenova/shared/rules/tariff-wire";
import { cn } from "@trenova/shared/lib/utils";
import type { TariffType } from "@trenova/shared/types/tariff";
import { Controller, useFormContext, useWatch, type FieldValues, type Path } from "react-hook-form";

/**
 * What an emptied field writes: `undefined` leaves the input unset so it inherits (a top-level
 * input, or a required record field the check then reports), `null` writes `none` (an optional
 * record field).
 */
export type TariffEmptyValue = undefined | null;

export type TariffScalarFieldProps = {
  /** The react-hook-form path of the value. */
  path: string;
  type: TariffType;
  /** The type as the tariff wrote it (`Money/pallet`), for the unit a folded count shows. */
  typeText?: string;
  label?: string;
  description?: string | null;
  placeholder?: string;
  /** A table cell: no label or description, named for assistive technology instead. */
  ariaLabel?: string;
  currency?: string | null;
  disabled?: boolean;
  empty?: TariffEmptyValue;
  /** For a bool: what it shows while unset (the inherited value). */
  inheritedBool?: boolean;
  /**
   * The only values a text or enum input accepts (its `where` list), offered as a select in
   * place of free text or every variant.
   */
  oneOf?: readonly string[] | null;
};

/** The decimal places a field keeps for each kind of number; amounts stay exact past cents. */
const SCALE = { money: 4, rate: 6, quantity: 6, percent: 4, number: 10, int: 0 } as const;

/** The `/pallet` a money type folds into its value, which the wire unit no longer carries. */
function moneyPer(typeText: string | undefined): string {
  const match = /^Money\/(pallet|piece|stop)\??$/.exec(typeText ?? "");
  return match ? `/${match[1]}` : "";
}

/** The number typed so far: `2.` is 2 while the decimals are still coming, `-` is nothing yet. */
function typedDecimal(text: string): string | null {
  return tariffDecimal(text.endsWith(".") ? text.slice(0, -1) : text);
}

function quantityCodec(unit: () => string, empty: TariffEmptyValue): NumberFieldCodec {
  return {
    decode: (value) => (isTariffQuantity(value) ? value.$q : ""),
    encode: (text, previous) => {
      const amount = typedDecimal(text);
      if (amount === null) return empty;
      return tariffQuantity(amount, isTariffQuantity(previous) ? previous.u : unit());
    },
  };
}

function numberCodec(empty: TariffEmptyValue): NumberFieldCodec {
  return {
    decode: (value) => (isTariffNumber(value) ? value.$n : ""),
    encode: (text) => {
      const amount = typedDecimal(text);
      return amount === null ? empty : tariffNumber(amount);
    },
  };
}

/**
 * One scalar input (or a scalar field of a record) as a form field, writing its wire value:
 * money, rates and quantities as decimal text with their unit, percents, whole and decimal
 * numbers, yes or no, text, dates and enums. A quantity whose dimension has more than one unit
 * gets a unit select limited to that dimension.
 */
export function TariffScalarField(props: TariffScalarFieldProps) {
  const { type } = props;
  switch (type.kind) {
    case "optional":
      return <TariffScalarField {...props} type={type.inner} empty={props.empty ?? null} />;
    case "money":
    case "percent":
    case "int":
    case "number":
      return <TariffNumberField {...props} />;
    case "quantity":
      return <TariffQuantityField {...props} type={type} />;
    case "bool":
      return <TariffBoolField {...props} />;
    case "text":
      return <TariffTextField {...props} />;
    case "enum":
      return <TariffEnumField {...props} type={type} />;
    case "date":
    case "dateTime":
      return <TariffDateField {...props} dateTime={type.kind === "dateTime"} />;
    case "list":
      return type.item.kind === "text" ? <TariffTextListField {...props} /> : null;
    case "record":
    case "unknown":
      return null;
  }
}

function TariffNumberField({
  path,
  type,
  typeText,
  label,
  description,
  placeholder,
  ariaLabel,
  currency,
  disabled,
  empty,
}: TariffScalarFieldProps) {
  const { control } = useFormContext<FieldValues>();
  const kind = type.kind;
  const codec =
    kind === "money"
      ? quantityCodec(() => `$${moneyPer(typeText)}`, empty)
      : kind === "percent"
        ? quantityCodec(() => "%", empty)
        : numberCodec(empty);
  const sideText =
    kind === "money" ? `${currency || "$"}${moneyPer(typeText)}` : kind === "percent" ? "%" : "";
  const scale =
    kind === "money"
      ? SCALE.money
      : kind === "percent"
        ? SCALE.percent
        : kind === "int"
          ? SCALE.int
          : SCALE.number;

  return (
    <NumberField
      control={control}
      name={path as Path<FieldValues>}
      codec={codec}
      label={label}
      description={description ?? undefined}
      placeholder={placeholder ?? ""}
      aria-label={ariaLabel}
      sideText={sideText || undefined}
      decimalScale={scale}
      allowNegative
      thousandSeparator={kind !== "int" && kind !== "number"}
      disabled={disabled}
    />
  );
}

function TariffQuantityField({
  path,
  type,
  label,
  description,
  placeholder,
  ariaLabel,
  disabled,
  empty,
}: TariffScalarFieldProps & { type: Extract<TariffType, { kind: "quantity" }> }) {
  const t = useT();
  const { control, getValues, setValue } = useFormContext<FieldValues>();
  const units = tariffUnitsOf(type.dimension);
  const money = type.unit.startsWith("$");
  const codec = quantityCodec(() => type.unit, empty);
  const value: unknown = useWatch({ control, name: path });
  const unit = isTariffQuantity(value) ? value.u : type.unit;

  return (
    <div className="flex min-w-0 items-start gap-1.5">
      <div className="min-w-0 flex-1">
        <NumberField
          control={control}
          name={path as Path<FieldValues>}
          codec={codec}
          label={label}
          description={description ?? undefined}
          placeholder={placeholder ?? ""}
          aria-label={ariaLabel}
          sideText={units.length > 1 ? undefined : unit}
          decimalScale={money ? SCALE.rate : SCALE.quantity}
          allowNegative
          thousandSeparator
          disabled={disabled}
        />
      </div>
      {units.length > 1 && (
        <div className={cn("shrink-0", label ? "pt-5" : undefined)}>
          <Select
            value={unit}
            disabled={disabled}
            onValueChange={(next) => {
              if (!next) return;
              const current = getValues(path);
              // A unit chosen before any digits waits for them; the field writes the
              // value in the unit it shows.
              if (isTariffQuantity(current)) {
                setValue(path, tariffQuantity(current.$q, next), {
                  shouldDirty: true,
                  shouldValidate: true,
                });
              } else {
                setValue(path, tariffQuantity("0", next), {
                  shouldDirty: true,
                  shouldValidate: true,
                });
              }
            }}
          >
            <SelectTrigger
              className="h-7 w-24 font-mono text-xs"
              aria-label={t("Unit of {0}", label ?? ariaLabel ?? "")}
            >
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {units.map((option) => (
                <SelectItem key={option} value={option} className="font-mono text-xs">
                  {option}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
      )}
    </div>
  );
}

function TariffBoolField({
  path,
  label,
  description,
  ariaLabel,
  disabled,
  inheritedBool,
}: TariffScalarFieldProps) {
  const { control } = useFormContext<FieldValues>();
  const value: unknown = useWatch({ control, name: path });
  return (
    <SwitchField
      control={control}
      name={path as Path<FieldValues>}
      label={label ?? ariaLabel ?? ""}
      aria-label={ariaLabel}
      description={description ?? undefined}
      disabled={disabled}
      outlined={Boolean(label)}
      checked={typeof value === "boolean" ? value : (inheritedBool ?? false)}
    />
  );
}

function TariffTextField({
  path,
  label,
  description,
  placeholder,
  ariaLabel,
  disabled,
  oneOf,
}: TariffScalarFieldProps) {
  const { control } = useFormContext<FieldValues>();
  if (oneOf && oneOf.length > 0) {
    return (
      <SelectField
        control={control}
        name={path as Path<FieldValues>}
        label={label}
        description={description ?? undefined}
        placeholder={placeholder ?? ""}
        options={oneOf.map((value) => ({ label: value, value }))}
        isReadOnly={disabled}
      />
    );
  }
  return (
    <InputField
      control={control}
      name={path as Path<FieldValues>}
      label={label}
      description={description ?? undefined}
      placeholder={placeholder}
      aria-label={ariaLabel}
      disabled={disabled}
    />
  );
}

function TariffTextListField({ path, label, description, placeholder }: TariffScalarFieldProps) {
  const { control } = useFormContext<FieldValues>();
  return (
    <TextChipsField
      control={control}
      name={path as Path<FieldValues>}
      label={label ?? ""}
      description={description ?? undefined}
      placeholder={placeholder}
    />
  );
}

function TariffEnumField({
  path,
  type,
  label,
  description,
  placeholder,
  disabled,
  oneOf,
}: TariffScalarFieldProps & { type: Extract<TariffType, { kind: "enum" }> }) {
  const { control } = useFormContext<FieldValues>();
  const variants =
    oneOf && oneOf.length > 0
      ? type.variants.filter((variant) => oneOf.includes(variant))
      : type.variants;
  return (
    <SelectField
      control={control}
      name={path as Path<FieldValues>}
      label={label}
      description={description ?? undefined}
      placeholder={placeholder ?? ""}
      options={variants.map((variant) => ({ label: variant, value: variant }))}
      isReadOnly={disabled}
    />
  );
}

/** The wall-clock offset of the reader's zone at a moment, as `-05:00`. */
function localOffset(at: Date): string {
  const minutes = -at.getTimezoneOffset();
  const sign = minutes < 0 ? "-" : "+";
  const absolute = Math.abs(minutes);
  const hours = String(Math.floor(absolute / 60)).padStart(2, "0");
  return `${sign}${hours}:${String(absolute % 60).padStart(2, "0")}`;
}

/** `2026-10-22T08:00` from a date-time's wire text, dropping its seconds and offset. */
function wallClock(text: string): string {
  return text.slice(0, 16);
}

function TariffDateField({
  path,
  label,
  description,
  ariaLabel,
  disabled,
  empty,
  dateTime,
}: TariffScalarFieldProps & { dateTime: boolean }) {
  const { control } = useFormContext<FieldValues>();
  return (
    <Controller
      control={control}
      name={path}
      render={({ field, fieldState }) => {
        const text = isTariffDate(field.value) ? field.value.$date : "";
        return (
          <FieldWrapper
            name={path}
            label={label}
            description={description ?? undefined}
            error={fieldState.error?.message}
          >
            <Input
              type={dateTime ? "datetime-local" : "date"}
              value={dateTime ? wallClock(text) : text}
              disabled={disabled}
              aria-label={ariaLabel}
              aria-invalid={fieldState.invalid}
              onBlur={field.onBlur}
              onChange={(event) => {
                const next = event.target.value;
                if (!next) {
                  field.onChange(empty);
                  return;
                }
                field.onChange(
                  tariffDate(dateTime ? `${next}:00${localOffset(new Date(next))}` : next),
                );
              }}
            />
          </FieldWrapper>
        );
      }}
    />
  );
}
