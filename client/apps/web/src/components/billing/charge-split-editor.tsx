"use no memo";
import { useT } from "@trenova/shared/i18n/use-t";
import { CustomerAutocompleteField } from "@/components/autocomplete-fields";
import { NumberField } from "@/components/fields/number-field";
import type { SelectOption as GraphQLSelectOption } from "@/lib/graphql/select-options";
import { Button } from "@trenova/shared/components/ui/button";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@trenova/shared/components/ui/select";
import { allocationRemainder } from "@trenova/shared/lib/charge-split";
import { cn, formatCurrency } from "@trenova/shared/lib/utils";
import type { ChargeAllocation, ChargeAllocationMethod } from "@trenova/shared/types/shipment";
import { PlusIcon, SplitIcon, Trash2Icon } from "lucide-react";
import { useState } from "react";
import {
  useFieldArray,
  useFormContext,
  useWatch,
  type Control,
  type FieldValues,
} from "react-hook-form";
import { allocationMethodChoices } from "@/lib/choices";
import { selectOptionMetaString } from "@/lib/select-option-meta";

export type ChargeSplitEditorProps = {
  control: Control<FieldValues>;
  /** Path of the allocation array in the form, e.g. `additionalCharges.2.allocations`. */
  name: string;
  /** What the charge comes to, for amount splits and the remainder line. */
  chargeAmount: number | null;
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
 * row editor, where every row must name a payer and the rows must add up.
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
  const state = allocationRemainder(rows, chargeAmount);
  const rootError = getFieldState(name, formState).error;
  const rootMessage =
    (rootError as { message?: string } | undefined)?.message ??
    (rootError as { root?: { message?: string } } | undefined)?.root?.message;

  const setRows = (next: ChargeAllocation[]) => {
    replace(next);
    setValue(name, next, { shouldDirty: true, shouldValidate: true });
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

  const handleMethodChange = (next: string | null) => {
    if (next !== "Percent" && next !== "Amount") return;
    setRows(
      rows.map((row) => ({
        ...row,
        method: next,
        percent: next === "Percent" ? row.percent : null,
        amount: next === "Amount" ? row.amount : null,
      })),
    );
  };

  const enterSplitMode = () => {
    setSplitMode(true);
    if (rows.length === 0) {
      const first = { ...emptyRow("Percent") } as ChargeAllocation;
      if (defaultPayer) {
        first.billToCustomerId = defaultPayer.id;
        first.percent = 100;
        first.billToCustomer = { id: defaultPayer.id, name: defaultPayer.label, code: null };
      }
      setRows([first, emptyRow("Percent")]);
    } else if (rows.length === 1) {
      append(emptyRow(method));
    }
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
      <div className="flex items-center justify-between gap-2">
        <span className="text-xs font-medium">{t("Split between payers")}</span>
        {percentOnly ? null : (
          <Select value={method} items={allocationMethodChoices} onValueChange={handleMethodChange}>
            <SelectTrigger className="h-7 w-28 text-xs" aria-label={t("Split method")}>
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {allocationMethodChoices.map((choice) => (
                <SelectItem key={choice.value} value={choice.value}>
                  {t(choice.label)}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        )}
      </div>
      <div className="flex flex-col gap-2">
        {fields.map((field, index) => (
          <div key={field.fieldId} className="grid grid-cols-12 items-start gap-2">
            <div className="col-span-7">
              <CustomerAutocompleteField
                control={control}
                name={`${name}.${index}.billToCustomerId`}
                label={index === 0 ? t("Payer") : undefined}
                placeholder={t("Select payer")}
                disabled={disabled}
                rules={{ required: true }}
                onOptionChange={(option) =>
                  setValue(`${name}.${index}.billToCustomer`, snapshotFrom(option), {
                    shouldDirty: true,
                  })
                }
              />
            </div>
            <div className="col-span-4">
              <NumberField
                control={control}
                name={`${name}.${index}.${method === "Amount" ? "amount" : "percent"}`}
                label={index === 0 ? (method === "Amount" ? t("Amount") : t("Percent")) : undefined}
                decimalScale={2}
                placeholder={method === "Amount" ? "0.00" : "0"}
                sideText={method === "Amount" ? currencyCode : "%"}
                disabled={disabled}
                rules={{ required: true, min: 0.01 }}
              />
            </div>
            <div className={cn("col-span-1 flex justify-end", index === 0 && "pt-6")}>
              <Button
                type="button"
                variant="ghost"
                size="icon"
                className="size-7"
                aria-label={t("Remove payer")}
                disabled={disabled}
                onClick={() => remove(index)}
              >
                <Trash2Icon className="text-muted-foreground size-3.5" />
              </Button>
            </div>
          </div>
        ))}
      </div>
      <div className="flex items-center justify-between gap-2">
        <Button
          type="button"
          variant="outline"
          size="xxs"
          disabled={disabled}
          onClick={() => append(emptyRow(method))}
        >
          <PlusIcon className="size-3" />
          {t("Add payer")}
        </Button>
        <Button
          type="button"
          variant="ghost"
          size="xxs"
          disabled={disabled}
          onClick={leaveSplitMode}
        >
          {t("Bill to one payer")}
        </Button>
      </div>
      <p
        className={cn(
          "text-2xs tabular-nums",
          state.isOver ? "text-destructive font-medium" : "text-muted-foreground",
        )}
        data-testid="charge-split-remainder"
      >
        {method === "Amount"
          ? t(
              "Allocated {0} · {1} remaining",
              formatCurrency(state.allocated, currencyCode),
              formatCurrency(state.remaining, currencyCode),
            )
          : t("Allocated {0}% · {1}% remaining", state.allocated, state.remaining)}
      </p>
      {rootMessage ? <p className="text-destructive text-2xs">{rootMessage}</p> : null}
    </div>
  );
}
