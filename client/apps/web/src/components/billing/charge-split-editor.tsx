"use no memo";
import { useT } from "@trenova/shared/i18n/use-t";
import { CustomerAutocompleteField } from "@/components/autocomplete-fields";
import { NumberField } from "@/components/fields/number-field";
import type { SelectOption as GraphQLSelectOption } from "@/lib/graphql/select-options";
import { Button } from "@trenova/shared/components/ui/button";
import { SegmentedControl } from "@trenova/shared/components/ui/segmented-control";
import { LazyComponent } from "@trenova/shared/components/error-boundary";
import { useRules } from "@trenova/shared/hooks/use-rules";
import { formatNumber } from "@trenova/shared/i18n/format";
import { allocationRemainders, type AllocationRemainder } from "@trenova/shared/rules/allocation";
import type { DecimalInput } from "@trenova/shared/rules/wire";
import { cn, formatCurrency } from "@trenova/shared/lib/utils";
import type { ChargeAllocation, ChargeAllocationMethod } from "@trenova/shared/types/shipment";
import {
  CheckCircleIcon,
  PlusIcon,
  SplitIcon,
  Trash01Icon,
} from "@trenova/shared/components/icons";
import { useState } from "react";
import {
  useFieldArray,
  useFormContext,
  useWatch,
  type Control,
  type FieldValues,
} from "react-hook-form";
import { selectOptionMetaString } from "@/lib/select-option-meta";
import { ChargeSplitRowsSkeleton } from "./charge-split-rows-skeleton";

export type ChargeSplitEditorProps = {
  control: Control<FieldValues>;
  /** Path of the allocation array in the form, e.g. `additionalCharges.2.allocations`. */
  name: string;
  /** What the charge comes to, for amount splits and the remainder line; null when unknown. */
  chargeAmount: DecimalInput;
  /** Restricts the split to percentages, for charges whose total is only known server-side. */
  percentOnly?: boolean;
  /** Who is billed when no split is entered. */
  defaultPayer: { id: string; label: string } | null;
  currencyCode?: string;
  disabled?: boolean;
};

function emptyRow(method: ChargeAllocationMethod): ChargeAllocation {
  return {
    billToCustomerId: "",
    method,
    percent: null,
    amount: null,
    sequence: 0,
  } as ChargeAllocation;
}

function snapshotFrom(option: GraphQLSelectOption | null) {
  if (!option?.id) return null;
  return { id: option.id, name: option.label, code: selectOptionMetaString(option, "code") };
}

/**
 * Splits one charge among payers. In its simple form it is a single "Bill to"
 * picker: choosing a customer bills the whole charge to them, clearing it
 * returns the charge to the shipment's payer. "Split between payers" opens the
 * row editor, split in half between that payer and a new one, where every row
 * must name a payer and the rows must add up.
 */
export function ChargeSplitEditor({
  control,
  name,
  chargeAmount,
  percentOnly = false,
  defaultPayer,
  currencyCode = "USD",
  disabled = false,
}: ChargeSplitEditorProps) {
  const t = useT();
  const { setValue, getFieldState, formState } = useFormContext<FieldValues>();
  const { fields, append, remove, replace } = useFieldArray({ control, name, keyName: "fieldId" });
  const rows = (useWatch({ control, name }) ?? []) as ChargeAllocation[];
  const [splitMode, setSplitMode] = useState(rows.length > 1);

  const method: ChargeAllocationMethod = rows[0]?.method === "Amount" ? "Amount" : "Percent";
  const rootError = getFieldState(name, formState).error;
  const rootMessage =
    (rootError as { message?: string } | undefined)?.message ??
    (rootError as { root?: { message?: string } } | undefined)?.root?.message;

  // Rows are judged as they change only once a save has been tried; before then a row
  // still waiting for its payer is work in progress, not an error.
  const setRows = (next: ChargeAllocation[]) => {
    replace(next);
    setValue(name, next, { shouldDirty: true, shouldValidate: formState.isSubmitted });
  };

  const handleSinglePayer = (option: GraphQLSelectOption | null) => {
    if (!option?.id) {
      setRows([]);
      return;
    }
    setRows([
      {
        ...emptyRow("Percent"),
        billToCustomerId: option.id,
        percent: 100,
        billToCustomer: snapshotFrom(option),
      } as ChargeAllocation,
    ]);
  };

  // A split starts in half between whoever pays now and a payer still to choose, so a two-way
  // split is one choice away.
  const enterSplitMode = () => {
    setSplitMode(true);
    const half = { ...emptyRow("Percent"), percent: 50 } as ChargeAllocation;
    const current = rows[0];
    const first: ChargeAllocation = current?.billToCustomerId
      ? { ...current, method: "Percent", percent: 50, amount: null }
      : defaultPayer
        ? ({
            ...half,
            billToCustomerId: defaultPayer.id,
            billToCustomer: { id: defaultPayer.id, name: defaultPayer.label, code: null },
          } as ChargeAllocation)
        : half;
    setRows([first, { ...half }]);
  };

  const leaveSplitMode = () => {
    setSplitMode(false);
    setRows([]);
  };

  if (!splitMode) {
    return (
      <div className="flex flex-col gap-2" data-testid="charge-split-simple">
        <CustomerAutocompleteField
          control={control}
          name={`${name}.0.billToCustomerId`}
          label={t("Bill to")}
          clearable
          disabled={disabled}
          placeholder={
            defaultPayer ? t("Same as shipment ({0})", defaultPayer.label) : t("Same as shipment")
          }
          description={t(
            "Who is invoiced for this charge. Leave blank to bill the shipment's payer.",
          )}
          onOptionChange={handleSinglePayer}
        />
        <Button
          type="button"
          variant="ghost"
          size="xxs"
          className="self-start"
          disabled={disabled}
          onClick={enterSplitMode}
        >
          <SplitIcon className="size-3" />
          {t("Split between payers")}
        </Button>
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-2" data-testid="charge-split-rows">
      <LazyComponent fallback={<ChargeSplitRowsSkeleton />}>
        <SplitRows
          control={control}
          name={name}
          rows={rows}
          fieldIds={fields.map((field) => field.fieldId)}
          method={method}
          chargeAmount={chargeAmount}
          percentOnly={percentOnly}
          currencyCode={currencyCode}
          disabled={disabled}
          setRows={setRows}
          appendRow={append}
          removeRow={remove}
          onBillOnePayer={leaveSplitMode}
        />
      </LazyComponent>
      {rootMessage ? <p className="text-destructive text-2xs">{rootMessage}</p> : null}
    </div>
  );
}

/** One accent per payer, in row order, shared by the row's dot and its slice of the bar. */
const PAYER_ACCENTS = [
  "bg-accent-indigo",
  "bg-accent-teal",
  "bg-accent-amber",
  "bg-accent-rose",
  "bg-accent-emerald",
  "bg-accent-sky",
  "bg-accent-violet",
  "bg-accent-slate",
] as const;

function payerAccent(index: number) {
  return PAYER_ACCENTS[index % PAYER_ACCENTS.length];
}

const ROW_GRID = "grid grid-cols-[0.5rem_minmax(0,1fr)_7.5rem_5.5rem_1.75rem] items-start gap-2";
const PERCENT_DIGITS = { maximumFractionDigits: 2 } as const;

/**
 * The rows of a split, measured by the rules module the way the server will judge them: each
 * row shows the other half of what it bills (a percent row its dollars, an amount row its
 * share), a bar shows the split's shape, and the status line says what is left, with a way to
 * fill it. Splitting evenly, filling, adding a payer and switching between percent and amount
 * take their values from the rules module too, so nothing here rounds money on its own.
 */
function SplitRows({
  control,
  name,
  rows,
  fieldIds,
  method,
  chargeAmount,
  percentOnly,
  currencyCode,
  disabled,
  setRows,
  appendRow,
  removeRow,
  onBillOnePayer,
}: {
  control: Control<FieldValues>;
  name: string;
  rows: ChargeAllocation[];
  fieldIds: string[];
  method: ChargeAllocationMethod;
  chargeAmount: DecimalInput;
  percentOnly: boolean;
  currencyCode: string;
  disabled: boolean;
  setRows: (next: ChargeAllocation[]) => void;
  appendRow: (row: ChargeAllocation) => void;
  removeRow: (index: number) => void;
  onBillOnePayer: () => void;
}) {
  const t = useT();
  const { setValue } = useFormContext<FieldValues>();
  const rules = useRules();
  const totalKnown = chargeAmount !== null && chargeAmount !== undefined;
  const measure = (next: readonly ChargeAllocation[]): AllocationRemainder => {
    const [state] = allocationRemainders(rules, {
      charges: [{ ...(totalKnown ? { total: chargeAmount } : {}), allocations: next }],
    });
    return state!;
  };
  const state = measure(rows);
  const valueKey = method === "Amount" ? "amount" : "percent";
  const withValue = (
    row: ChargeAllocation,
    key: "amount" | "percent",
    value: string | null | undefined,
  ): ChargeAllocation => ({ ...row, [key]: value == null ? null : Number(value) });
  const hasValue = (row: ChargeAllocation) => Number(row[valueKey] ?? 0) > 0;

  // A new payer takes whatever is left, so adding the last payer finishes the split.
  const addPayer = () => {
    const next = [...rows, emptyRow(method)];
    appendRow(withValue(emptyRow(method), valueKey, measure(next).fills[next.length - 1]));
  };

  // What is left goes to the first row with nothing in it yet, or else to the last row.
  const fillTarget = rows.findIndex((row) => !hasValue(row));
  const fillIndex = fillTarget === -1 ? rows.length - 1 : fillTarget;
  const fillValue = state.fills[fillIndex] ?? null;
  const fillRest = () =>
    setRows(
      rows.map((row, index) => (index === fillIndex ? withValue(row, valueKey, fillValue) : row)),
    );

  const canSplitEvenly = rows.length > 1 && state.even.every((value) => value !== null);
  const splitEvenly = () =>
    setRows(rows.map((row, index) => withValue(row, valueKey, state.even[index])));

  // Switching keeps the split: each row takes the other half of its share. A complete split
  // stays complete, its last row taking whatever the rounding of the others left.
  const changeMethod = (next: ChargeAllocationMethod) => {
    if (next === method) return;
    const key = next === "Amount" ? "amount" : "percent";
    const converted = rows.map((row, index) => {
      const share = state.shares[index];
      return withValue(
        { ...row, method: next, percent: null, amount: null },
        key,
        next === "Amount" ? share?.amount : share?.percent,
      );
    });
    const last = converted.length - 1;
    if (state.complete && last >= 0) {
      converted[last] = withValue(converted[last]!, key, measure(converted).fills[last]);
    }
    setRows(converted);
  };

  return (
    <>
      <div className="flex items-center justify-between gap-2">
        {percentOnly ? (
          <span className="text-xs font-medium">{t("Split between payers")}</span>
        ) : (
          <SegmentedControl<ChargeAllocationMethod>
            aria-label={t("Split method")}
            value={method}
            onValueChange={changeMethod}
            items={[
              { value: "Percent", label: t("By percent"), disabled },
              { value: "Amount", label: t("By amount"), disabled },
            ]}
          />
        )}
        <Button
          type="button"
          variant="ghost"
          size="xxs"
          disabled={disabled || !canSplitEvenly}
          onClick={splitEvenly}
        >
          {t("Split evenly")}
        </Button>
      </div>

      <SplitBar state={state} rowIds={fieldIds} />

      <div className="flex flex-col gap-2">
        {fieldIds.map((fieldId, index) => {
          const share = state.shares[index];
          const fill = state.fills[index];
          const other =
            method === "Amount"
              ? share?.percent != null
                ? t("{0}%", formatNumber(Number(share.percent), PERCENT_DIGITS))
                : "—"
              : share?.amount != null
                ? formatCurrency(Number(share.amount), currencyCode)
                : "—";
          // The first row's fields carry labels; its other cells drop below them to line up
          // with the inputs. The drop is on a wrapper so each cell keeps the input's height.
          const below = index === 0 ? "pt-6" : undefined;
          return (
            <div key={fieldId} className={ROW_GRID}>
              <div aria-hidden="true" className={below}>
                <span className="flex h-7 items-center">
                  <span className={cn("size-2 rounded-full", payerAccent(index))} />
                </span>
              </div>
              <CustomerAutocompleteField
                control={control}
                name={`${name}.${index}.billToCustomerId`}
                label={index === 0 ? t("Payer") : undefined}
                placeholder={t("Select payer")}
                disabled={disabled}
                required
                onOptionChange={(option) =>
                  setValue(`${name}.${index}.billToCustomer`, snapshotFrom(option), {
                    shouldDirty: true,
                  })
                }
              />
              <NumberField
                control={control}
                name={`${name}.${index}.${valueKey}`}
                label={index === 0 ? (method === "Amount" ? t("Amount") : t("Percent")) : undefined}
                aria-label={method === "Amount" ? t("Amount") : t("Percent")}
                decimalScale={method === "Amount" ? 2 : 6}
                placeholder={fill != null ? formatNumber(Number(fill), PERCENT_DIGITS) : "0"}
                sideText={method === "Amount" ? currencyCode : "%"}
                stepper={false}
                disabled={disabled}
                required
                rules={{ min: 0.01 }}
              />
              <div className={cn("min-w-0", below)}>
                <span className="flex h-7 min-w-0 items-center justify-end">
                  <span
                    className="text-foreground-muted truncate text-xs tabular-nums"
                    title={other}
                    data-testid={`charge-split-share-${index}`}
                  >
                    {other}
                  </span>
                </span>
              </div>
              <div className={cn("flex justify-end", below)}>
                <Button
                  type="button"
                  variant="ghost"
                  size="icon"
                  className="size-7"
                  aria-label={t("Remove payer")}
                  disabled={disabled || rows.length <= 1}
                  onClick={() => removeRow(index)}
                >
                  <Trash01Icon className="text-muted-foreground size-3.5" />
                </Button>
              </div>
            </div>
          );
        })}
      </div>

      <div className="flex items-center justify-between gap-2">
        <Button type="button" variant="outline" size="xxs" disabled={disabled} onClick={addPayer}>
          <PlusIcon className="size-3" />
          {t("Add payer")}
        </Button>
        <Button
          type="button"
          variant="ghost"
          size="xxs"
          disabled={disabled}
          onClick={onBillOnePayer}
        >
          {t("Bill to one payer")}
        </Button>
      </div>

      <SplitStatus
        state={state}
        method={method}
        chargeAmount={chargeAmount}
        currencyCode={currencyCode}
        canFill={!disabled && fillValue !== null && !state.complete && !state.over}
        onFill={fillRest}
      />
    </>
  );
}

/**
 * The split's shape: one slice per row in its payer's accent, widths in proportion to each
 * row's share. Over 100% the slices shrink to fit and the excess shows at the end in red.
 */
function SplitBar({ state, rowIds }: { state: AllocationRemainder; rowIds: string[] }) {
  const percents = state.shares.map((share) => Number(share?.percent ?? 0));
  const assigned = percents.reduce((sum, percent) => sum + percent, 0);
  const scale = Math.max(100, assigned);

  return (
    <div
      aria-hidden="true"
      className="bg-muted flex h-1.5 w-full overflow-hidden rounded-full"
      data-testid="charge-split-bar"
    >
      {percents.map((percent, index) => (
        <span
          key={rowIds[index] ?? `over-${index}`}
          className={cn("h-full transition-[width]", payerAccent(index))}
          style={{ width: `${(percent / scale) * 100}%` }}
        />
      ))}
      {state.over ? (
        <span
          className="bg-destructive h-full transition-[width]"
          style={{ width: `${((assigned - 100) / scale) * 100}%` }}
        />
      ) : null}
    </div>
  );
}

/**
 * What the rows leave: covered, a remainder with a way to fill it, or how far over they are.
 * Amount rows of a charge whose total is unknown can only say what they assign.
 */
function SplitStatus({
  state,
  method,
  chargeAmount,
  currencyCode,
  canFill,
  onFill,
}: {
  state: AllocationRemainder;
  method: ChargeAllocationMethod;
  chargeAmount: DecimalInput;
  currencyCode: string;
  canFill: boolean;
  onFill: () => void;
}) {
  const t = useT();
  const money = (value: DecimalInput) => formatCurrency(Number(value ?? 0), currencyCode);
  const percent = (value: DecimalInput) => formatNumber(Number(value ?? 0), PERCENT_DIGITS);
  const totalKnown = chargeAmount !== null && chargeAmount !== undefined;

  let text: string;
  let tone: "muted" | "success" | "danger" = "muted";
  if (method === "Amount" && state.remaining === null) {
    text = t("Assigned {0}", money(state.allocated));
  } else if (state.complete) {
    tone = "success";
    text = totalKnown
      ? t("The full {0} is covered", money(chargeAmount))
      : t("All 100% is assigned");
  } else if (state.over) {
    tone = "danger";
    const over = Number(state.remaining ?? 0) * -1;
    text = method === "Amount" ? t("{0} over", money(over)) : t("{0}% over", percent(over));
  } else {
    text =
      method === "Amount"
        ? t("{0} left to assign", money(state.remaining))
        : t("{0}% left to assign", percent(state.remaining));
  }

  return (
    <div className="flex min-h-5 items-center gap-1.5">
      <p
        className={cn(
          "flex items-center gap-1 text-xs tabular-nums",
          tone === "success" && "text-success-foreground",
          tone === "danger" && "text-destructive font-medium",
          tone === "muted" && "text-muted-foreground",
        )}
        data-testid="charge-split-remainder"
      >
        {tone === "success" ? <CheckCircleIcon className="size-3.5" /> : null}
        {text}
      </p>
      {canFill ? (
        <Button type="button" variant="ghost" size="xxs" onClick={onFill}>
          {t("Fill")}
        </Button>
      ) : null}
    </div>
  );
}
