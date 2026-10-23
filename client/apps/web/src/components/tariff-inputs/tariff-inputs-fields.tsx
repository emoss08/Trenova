import { FieldRow, type FieldLayout } from "@/components/fields/field-components";
import { useElementWidth } from "@trenova/shared/hooks/use-element-width";
import { Button } from "@trenova/shared/components/ui/button";
import { FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import { PlusIcon, RefreshCcw01Icon, Trash01Icon } from "@trenova/shared/components/icons";
import type { TranslateFn } from "@trenova/shared/i18n/runtime";
import { useT } from "@trenova/shared/i18n/use-t";
import { inheritedTariffValue, tariffInputText } from "@trenova/shared/lib/tariff-inputs";
import { formatTariffValue } from "@trenova/shared/lib/tariff-value";
import { emptyTariffValue } from "@trenova/shared/rules/tariff-wire";
import type {
  TariffInputDeclaration,
  TariffInputValues,
  TariffType,
  TariffTypeField,
  TariffWireValue,
} from "@trenova/shared/types/tariff";
import {
  useFieldArray,
  useFormContext,
  useFormState,
  useWatch,
  type FieldValues,
} from "react-hook-form";
import { tariffMessageText } from "@/components/tariff-bill/refusal-text";
import { TariffScalarField, type TariffEmptyValue } from "./tariff-input-field";

/** Where a rule's unset input takes its value from, for its placeholder. */
export type TariffInheritedFrom = "agreement";

export type TariffInputsFieldsProps = {
  /** The tariff's declared inputs, in declaration order, as `tariff.check` derives them. */
  declarations: readonly TariffInputDeclaration[];
  /**
   * The react-hook-form path the values live under, as a map from input name to wire value:
   * `inputs`, `tariffInputs`, `rules.${index}.inputs`.
   */
  name: string;
  /**
   * Values an input inherits when this form leaves it unset (the agreement's, under a rule),
   * shown as the field's placeholder; the declared default is the fallback after these.
   */
  inherited?: TariffInputValues;
  /** Whose values `inherited` holds, so a placeholder can say where a value comes from. */
  inheritedFrom?: TariffInheritedFrom;
  /** The rating currency's ISO code, for money placeholders. */
  currency?: string | null;
  /**
   * Whether an input with no default must be set here. False for an agreement, whose lanes
   * may set what it leaves unset.
   */
  requireUnset?: boolean;
  disabled?: boolean;
  /**
   * Shows each input's name and type under its label as a link to its declaration, for a form
   * beside the source (the Studio's).
   */
  onReveal?: (declaration: TariffInputDeclaration) => void;
};

/** Narrower than this, a row stacks its label over its control. */
const INLINE_MIN_WIDTH = 480;

function isComplex(type: TariffType): boolean {
  const inner = type.kind === "optional" ? type.inner : type;
  return inner.kind === "record" || (inner.kind === "list" && inner.item.kind !== "text");
}

/**
 * A record field's label and description, from its doc comment: the standard library's in the
 * reader's language, an organisation library's as written.
 */
function fieldText(
  field: TariffTypeField,
  t: TranslateFn,
): { label: string; description: string | null } {
  const doc = field.doc === null ? null : tariffMessageText(field.doc, t);
  return tariffInputText({ name: field.name, doc });
}

function joinPath(base: string, part: string | number): string {
  return base ? `${base}.${part}` : String(part);
}

type InheritanceCopy = {
  /** The placeholder an unset field shows. */
  placeholder: string;
  /** Where an unset value comes from, for a record or list that describes it in its own words. */
  source: "agreement" | "inherited" | "default" | null;
  /** What the reset button says, or null when there is nothing to go back to. */
  reset: string | null;
};

function inheritanceCopy(
  declaration: TariffInputDeclaration,
  inherited: TariffInputValues | undefined,
  inheritedFrom: TariffInheritedFrom | undefined,
  currency: string | null | undefined,
  requireUnset: boolean,
  t: TranslateFn,
): InheritanceCopy {
  const fromParent = inherited?.[declaration.name];
  if (fromParent !== undefined) {
    const shown = formatTariffValue(fromParent, t, { currency });
    return inheritedFrom === "agreement"
      ? {
          placeholder: t("Agreement: {0}", shown),
          source: "agreement",
          reset: t("Use the agreement's value"),
        }
      : {
          placeholder: t("Inherits {0}", shown),
          source: "inherited",
          reset: t("Use the inherited value"),
        };
  }
  if (declaration.default !== undefined) {
    return {
      placeholder: t("Default: {0}", formatTariffValue(declaration.default, t, { currency })),
      source: "default",
      reset: t("Use the default"),
    };
  }
  return requireUnset
    ? { placeholder: t("Required"), source: null, reset: null }
    : { placeholder: t("Not set"), source: null, reset: t("Leave unset") };
}

/**
 * A tariff's inputs as form fields under `name`: labels from the doc comments, unit-aware
 * money, rate and quantity fields, enums, optionals, records as a group of fields and lists of
 * records (weight breaks) as an editable table, each value written in its wire form so the form
 * saves exactly what the tariff reads. An input left unset inherits (the agreement's value under
 * a rule, then the declared default), which each field's placeholder says.
 *
 * Each input is a row: its label and doc beside its control where there is room, above it where
 * there is not, so no input's label can push another's control out of line.
 */
export function TariffInputsFields({
  declarations,
  name,
  inherited,
  inheritedFrom,
  currency,
  requireUnset = true,
  disabled,
  onReveal,
}: TariffInputsFieldsProps) {
  const t = useT();
  const [measure, width] = useElementWidth<HTMLDivElement>();
  const layout: FieldLayout = width === 0 || width >= INLINE_MIN_WIDTH ? "inline" : "stacked";

  if (declarations.length === 0) {
    return <p className="text-muted-foreground text-sm">{t("This tariff declares no inputs.")}</p>;
  }

  return (
    <div ref={measure} className="divide-border flex flex-col divide-y">
      {declarations.map((declaration) => (
        <div
          key={declaration.name}
          className="py-3 first:pt-0 last:pb-0"
          data-tariff-input={declaration.name}
        >
          <TariffInputField
            declaration={declaration}
            path={joinPath(name, declaration.name)}
            inherited={inherited}
            inheritedFrom={inheritedFrom}
            currency={currency}
            requireUnset={requireUnset}
            disabled={disabled}
            layout={layout}
            onReveal={onReveal}
          />
        </div>
      ))}
    </div>
  );
}

/** The input's name and type as written, as a link to where it is declared. */
function DeclarationLink({
  declaration,
  onReveal,
}: {
  declaration: TariffInputDeclaration;
  onReveal: (declaration: TariffInputDeclaration) => void;
}) {
  const t = useT();
  return (
    <button
      type="button"
      disabled={!declaration.range}
      onClick={() => onReveal(declaration)}
      aria-label={t("Show {0} in the source", declaration.name)}
      className="ui-focus-ring text-2xs text-foreground-subtle hover:text-foreground flex max-w-full items-center gap-1.5 self-start truncate rounded-sm font-mono disabled:pointer-events-none"
    >
      {declaration.name}
      <span className="text-syntax-type">{declaration.type}</span>
    </button>
  );
}

/** A row's description: the doc after its first sentence, then the declaration link. */
function rowDescription(
  declaration: TariffInputDeclaration,
  description: string | null,
  onReveal: TariffInputsFieldsProps["onReveal"],
): React.ReactNode {
  if (!onReveal) return description;
  return (
    <div className="flex min-w-0 flex-col gap-1">
      {description && <p className="text-2xs text-foreground/70 text-left">{description}</p>}
      <DeclarationLink declaration={declaration} onReveal={onReveal} />
    </div>
  );
}

type TariffInputFieldProps = {
  declaration: TariffInputDeclaration;
  path: string;
  inherited?: TariffInputValues;
  inheritedFrom?: TariffInheritedFrom;
  currency?: string | null;
  requireUnset: boolean;
  disabled?: boolean;
  layout: FieldLayout;
  onReveal?: TariffInputsFieldsProps["onReveal"];
};

function TariffInputField({
  declaration,
  path,
  inherited,
  inheritedFrom,
  currency,
  requireUnset,
  disabled,
  layout,
  onReveal,
}: TariffInputFieldProps) {
  const t = useT();
  const { control, setValue } = useFormContext<FieldValues>();
  const value = useWatch({ control, name: path }) as TariffWireValue | undefined;
  const { label, description: doc } = tariffInputText(declaration);
  const description = rowDescription(declaration, doc, onReveal);
  const copy = inheritanceCopy(declaration, inherited, inheritedFrom, currency, requireUnset, t);
  const fallback = inheritedTariffValue(declaration, inherited);
  const isSet = value !== undefined;

  const reset =
    isSet && copy.reset && !disabled ? (
      <Button
        type="button"
        variant="ghost"
        size="xs"
        className="text-muted-foreground mt-1 h-6 gap-1 self-start px-1.5"
        onClick={() => setValue(path, undefined, { shouldDirty: true, shouldValidate: true })}
      >
        <RefreshCcw01Icon className="size-3" />
        {copy.reset}
      </Button>
    ) : null;

  if (isComplex(declaration.descriptor)) {
    return (
      <TariffComplexInput
        declaration={declaration}
        path={path}
        label={label}
        description={description}
        value={value}
        fallback={fallback}
        placeholder={copy.placeholder}
        source={copy.source}
        currency={currency}
        disabled={disabled}
        reset={reset}
        layout={layout}
      />
    );
  }

  return (
    <>
      <TariffScalarField
        path={path}
        type={declaration.descriptor}
        typeText={declaration.type}
        label={label}
        description={description}
        layout={layout}
        placeholder={copy.placeholder}
        currency={currency}
        disabled={disabled}
        empty={undefined}
        inheritedBool={typeof fallback === "boolean" ? fallback : undefined}
      />
      {reset && (layout === "inline" ? <FieldRow>{reset}</FieldRow> : reset)}
    </>
  );
}

type TariffComplexInputProps = {
  declaration: TariffInputDeclaration;
  path: string;
  label: string;
  description: React.ReactNode;
  value: TariffWireValue | undefined;
  fallback: TariffWireValue | undefined;
  placeholder: string;
  source: InheritanceCopy["source"];
  currency?: string | null;
  disabled?: boolean;
  reset: React.ReactNode;
  layout: FieldLayout;
};

/**
 * A record or a list input. Unset, it shows the value it inherits and offers to set its own,
 * starting from a copy of that value; set, it is edited field by field or row by row, across the
 * whole row under its label.
 */
function TariffComplexInput({
  declaration,
  path,
  label,
  description,
  value,
  fallback,
  placeholder,
  source,
  currency,
  disabled,
  reset,
  layout,
}: TariffComplexInputProps) {
  const t = useT();
  const { setValue } = useFormContext<FieldValues>();
  const type =
    declaration.descriptor.kind === "optional"
      ? declaration.descriptor.inner
      : declaration.descriptor;
  const isSet = value !== undefined && value !== null;

  const status = (
    <div className="flex min-h-7 flex-wrap items-center gap-x-3 gap-y-1">
      {isSet ? null : (
        <p className="text-muted-foreground text-xs">
          {fallback === undefined || source === null
            ? placeholder
            : unsetComplexText(source, describeComplex(fallback, type, t, currency), t)}
        </p>
      )}
      {!isSet && !disabled && (
        <Button
          type="button"
          variant="outline"
          size="xs"
          onClick={() =>
            setValue(path, structuredClone(fallback ?? emptyTariffValue(type)), {
              shouldDirty: true,
              shouldValidate: true,
            })
          }
        >
          {t("Set here")}
        </Button>
      )}
      {reset}
    </div>
  );

  return (
    <section className="flex flex-col gap-2" aria-label={label}>
      {layout === "inline" ? (
        <FieldRow label={label} description={description}>
          {status}
        </FieldRow>
      ) : (
        <div className="flex flex-col gap-1">
          <span className="text-xs font-medium">{label}</span>
          {typeof description === "string" ? (
            <span className="text-muted-foreground text-2xs">{description}</span>
          ) : (
            description
          )}
          {status}
        </div>
      )}

      {isSet &&
        (type.kind === "list" ? (
          <TariffListEditor
            path={path}
            type={type}
            label={label}
            currency={currency}
            disabled={disabled}
          />
        ) : type.kind === "record" ? (
          <TariffRecordFields path={path} type={type} currency={currency} disabled={disabled} />
        ) : null)}
    </section>
  );
}

/** What an unset record or list says it takes instead. */
function unsetComplexText(
  source: NonNullable<InheritanceCopy["source"]>,
  shown: string,
  t: TranslateFn,
): string {
  switch (source) {
    case "agreement":
      return t("Uses the agreement's: {0}", shown);
    case "inherited":
      return t("Inherits {0}", shown);
    case "default":
      return t("Uses the default: {0}", shown);
  }
}

/** A list or record as a short phrase for a placeholder: its size, or its fields. */
function describeComplex(
  value: TariffWireValue,
  type: TariffType,
  t: TranslateFn,
  currency: string | null | undefined,
): string {
  if (type.kind === "list" && Array.isArray(value)) {
    return t("{0, plural, one {# row} other {# rows}}", value.length);
  }
  return formatTariffValue(value, t, { currency });
}

function TariffRecordFields({
  path,
  type,
  currency,
  disabled,
}: {
  path: string;
  type: Extract<TariffType, { kind: "record" }>;
  currency?: string | null;
  disabled?: boolean;
}) {
  const t = useT();
  return (
    <FormGroup cols={2}>
      {type.fields.map((field) => {
        const text = fieldText(field, t);
        const fieldPath = joinPath(path, field.name);
        if (isComplex(field.type)) {
          const inner = field.type.kind === "optional" ? field.type.inner : field.type;
          return (
            <FormControl key={field.name} cols="full">
              <div className="flex flex-col gap-1">
                <span className="text-xs font-medium">{text.label}</span>
                {inner.kind === "list" ? (
                  <TariffListEditor
                    path={fieldPath}
                    type={inner}
                    label={text.label}
                    currency={currency}
                    disabled={disabled}
                  />
                ) : inner.kind === "record" ? (
                  <TariffRecordFields
                    path={fieldPath}
                    type={inner}
                    currency={currency}
                    disabled={disabled}
                  />
                ) : null}
              </div>
            </FormControl>
          );
        }
        return (
          <FormControl key={field.name}>
            <TariffScalarField
              path={fieldPath}
              type={field.type}
              label={text.label}
              description={text.description}
              currency={currency}
              disabled={disabled}
              empty={emptyFor(field.type)}
            />
          </FormControl>
        );
      })}
    </FormGroup>
  );
}

/** An emptied record field writes `none` when its type allows it; otherwise it is missing. */
function emptyFor(type: TariffType): TariffEmptyValue {
  return type.kind === "optional" ? null : undefined;
}

/**
 * A list input. A list of records is a table, one column per field and one row per item, the
 * way a rate sheet lays out weight breaks; a list of anything else is a column of values.
 */
function TariffListEditor({
  path,
  type,
  label,
  currency,
  disabled,
}: {
  path: string;
  type: Extract<TariffType, { kind: "list" }>;
  label: string;
  currency?: string | null;
  disabled?: boolean;
}) {
  if (type.item.kind === "record") {
    return (
      <TariffRecordTable
        path={path}
        record={type.item}
        label={label}
        currency={currency}
        disabled={disabled}
      />
    );
  }
  return (
    <TariffValueList
      path={path}
      item={type.item}
      label={label}
      currency={currency}
      disabled={disabled}
    />
  );
}

function ListError({ path }: { path: string }) {
  const { errors } = useFormState({ name: path });
  let node: unknown = errors;
  for (const part of path.split(".")) {
    node = (node as Record<string, unknown> | undefined)?.[part];
  }
  const error = node as { message?: string; root?: { message?: string } } | undefined;
  const message = error?.message ?? error?.root?.message;
  return message ? <p className="text-destructive text-xs">{message}</p> : null;
}

function TariffRecordTable({
  path,
  record,
  label,
  currency,
  disabled,
}: {
  path: string;
  record: Extract<TariffType, { kind: "record" }>;
  label: string;
  currency?: string | null;
  disabled?: boolean;
}) {
  const t = useT();
  const { control } = useFormContext<FieldValues>();
  const { fields, append, remove } = useFieldArray({ control, name: path });
  const columns = record.fields.filter((field) => !isComplex(field.type));

  return (
    <div className="flex flex-col gap-2">
      <div className="overflow-x-auto rounded-md border">
        <table className="w-full border-collapse text-sm" aria-label={label}>
          <thead>
            <tr className="bg-muted/50">
              {columns.map((column) => (
                <th
                  key={column.name}
                  scope="col"
                  className="text-muted-foreground border-b px-2 py-1.5 text-left text-xs font-medium"
                >
                  {fieldText(column, t).label}
                </th>
              ))}
              <th scope="col" className="w-8 border-b">
                <span className="sr-only">{t("Actions")}</span>
              </th>
            </tr>
          </thead>
          <tbody>
            {fields.length === 0 && (
              <tr>
                <td
                  colSpan={columns.length + 1}
                  className="text-muted-foreground px-2 py-3 text-center text-xs"
                >
                  {t("No rows yet.")}
                </td>
              </tr>
            )}
            {fields.map((row, index) => (
              <tr key={row.id}>
                {columns.map((column) => {
                  const columnLabel = fieldText(column, t).label;
                  return (
                    <td key={column.name} className="border-b px-1.5 py-1 align-top">
                      <TariffScalarField
                        path={joinPath(joinPath(path, index), column.name)}
                        type={column.type}
                        ariaLabel={t("{0}, row {1}", columnLabel, index + 1)}
                        currency={currency}
                        disabled={disabled}
                        empty={emptyFor(column.type)}
                      />
                    </td>
                  );
                })}
                <td className="border-b px-1 py-1 align-top">
                  <Button
                    type="button"
                    variant="ghost"
                    size="icon-xs"
                    aria-label={t("Remove row {0}", index + 1)}
                    disabled={disabled}
                    onClick={() => remove(index)}
                  >
                    <Trash01Icon className="size-3.5" />
                  </Button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      <ListError path={path} />
      {!disabled && (
        <Button
          type="button"
          variant="outline"
          size="xs"
          className="self-start"
          onClick={() => append(emptyTariffValue(record) as Record<string, unknown>)}
        >
          <PlusIcon className="size-3" />
          {t("Add row")}
        </Button>
      )}
    </div>
  );
}

function TariffValueList({
  path,
  item,
  label,
  currency,
  disabled,
}: {
  path: string;
  item: TariffType;
  label: string;
  currency?: string | null;
  disabled?: boolean;
}) {
  const t = useT();
  const { control, getValues, setValue } = useFormContext<FieldValues>();
  const items = (useWatch({ control, name: path }) as unknown[] | undefined) ?? [];

  const write = (next: unknown[]) =>
    setValue(path, next, { shouldDirty: true, shouldValidate: true });

  return (
    <div className="flex flex-col gap-1.5">
      {items.length === 0 && <p className="text-muted-foreground text-xs">{t("No values yet.")}</p>}
      {items.map((_, index) => (
        // oxlint-disable-next-line react/no-array-index-key -- a list's values have no identity but their place
        <div key={index} className="flex items-start gap-1.5">
          <div className="min-w-0 flex-1">
            <TariffScalarField
              path={joinPath(path, index)}
              type={item}
              ariaLabel={t("{0}, value {1}", label, index + 1)}
              currency={currency}
              disabled={disabled}
              empty={emptyFor(item)}
            />
          </div>
          <Button
            type="button"
            variant="ghost"
            size="icon-xs"
            aria-label={t("Remove value {0}", index + 1)}
            disabled={disabled}
            onClick={() => {
              const current = (getValues(path) as unknown[] | undefined) ?? [];
              write(current.filter((__, at) => at !== index));
            }}
          >
            <Trash01Icon className="size-3.5" />
          </Button>
        </div>
      ))}
      <ListError path={path} />
      {!disabled && (
        <Button
          type="button"
          variant="outline"
          size="xs"
          className="self-start"
          onClick={() => {
            const current = (getValues(path) as unknown[] | undefined) ?? [];
            write([...current, emptyTariffValue(item)]);
          }}
        >
          <PlusIcon className="size-3" />
          {t("Add value")}
        </Button>
      )}
    </div>
  );
}
