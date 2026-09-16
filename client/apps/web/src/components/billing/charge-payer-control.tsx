"use no memo";
import { useT } from "@trenova/shared/i18n/use-t";
import { CustomerAutocompleteField } from "@/components/autocomplete-fields";
import { allocationPayerLabel } from "@/components/billing/charge-payer-chip";
import { ChargeSplitDialog } from "@/components/billing/charge-split-dialog";
import type { SelectOption as GraphQLSelectOption } from "@/lib/graphql/select-options";
import { selectOptionMetaString } from "@/lib/select-option-meta";
import { Button } from "@trenova/shared/components/ui/button";
import { Popover, PopoverContent, PopoverTrigger } from "@trenova/shared/components/ui/popover";
import { cn } from "@trenova/shared/lib/utils";
import type { ChargeAllocation } from "@trenova/shared/types/shipment";
import { SplitIcon, UserRoundIcon } from "lucide-react";
import { useState } from "react";
import { type FieldValues, useForm, useFormContext, useWatch } from "react-hook-form";

export type ChargePayerControlProps = {
  /** Path of the charge's allocation array, e.g. `additionalCharges.2.allocations`. */
  name: string;
  chargeAmount: number | null;
  percentOnly?: boolean;
  /** The shipment's own payer: who is billed when the charge has no rows. */
  defaultPayer: { id: string; label: string } | null;
  splitTitle: string;
  splitDescription: string;
  disabled?: boolean;
  className?: string;
};

function isWholeCharge(row: ChargeAllocation, chargeAmount: number | null): boolean {
  if (row.method === "Percent") return Number(row.percent) === 100;
  return chargeAmount != null && Math.abs(Number(row.amount) - chargeAmount) < 0.005;
}

/**
 * Who pays one charge, and the one place to change it. The common case is a
 * single pick: the whole charge goes to that customer. Sharing a charge between
 * payers is a step further in, through the split editor.
 */
export function ChargePayerControl({
  name,
  chargeAmount,
  percentOnly = false,
  defaultPayer,
  splitTitle,
  splitDescription,
  disabled = false,
  className,
}: ChargePayerControlProps) {
  const t = useT();
  const { control, getValues, setValue } = useFormContext<FieldValues>();
  const rows = ((useWatch({ control, name }) ?? []) as ChargeAllocation[]).filter(
    (row) => row?.billToCustomerId,
  );
  const [open, setOpen] = useState(false);
  const [splitOpen, setSplitOpen] = useState(false);

  const single = rows.length === 1 && isWholeCharge(rows[0], chargeAmount) ? rows[0] : null;
  const picker = useForm<{ payerId: string }>({
    values: { payerId: single?.billToCustomerId ?? "" },
  });

  const writeRows = (next: ChargeAllocation[]) => {
    setValue(name, next, { shouldDirty: true, shouldValidate: true });
  };

  const handlePick = (option: GraphQLSelectOption | null) => {
    if (!option?.id) return;
    if (defaultPayer && option.id === defaultPayer.id) {
      writeRows([]);
      setOpen(false);
      return;
    }
    const existing = ((getValues(name) ?? []) as ChargeAllocation[]).filter(Boolean);
    const identity =
      existing.length === 1 && existing[0].id
        ? { id: existing[0].id, version: existing[0].version }
        : {};
    writeRows([
      {
        ...identity,
        billToCustomerId: option.id,
        method: "Percent",
        percent: 100,
        amount: null,
        sequence: 0,
        billToCustomer: {
          id: option.id,
          name: option.label,
          code: selectOptionMetaString(option, "code"),
        },
      } as ChargeAllocation,
    ]);
    setOpen(false);
  };

  const label =
    rows.length === 0
      ? t("Bill to: {0}", t("Same as shipment"))
      : single
        ? t("Bill to: {0}", allocationPayerLabel(single))
        : t("Split {0} ways", rows.length);

  return (
    <>
      <Popover open={open} onOpenChange={setOpen}>
        <PopoverTrigger
          render={
            <Button
              type="button"
              variant="ghost"
              size="xxs"
              disabled={disabled}
              className={cn(
                "text-2xs h-5 max-w-56 shrink-0 gap-1 truncate",
                rows.length > 0 && "bg-indigo-500/10 text-indigo-600 dark:text-indigo-400",
                className,
              )}
              data-testid="charge-payer-control"
            />
          }
        >
          {single || rows.length === 0 ? (
            <UserRoundIcon className="size-3" />
          ) : (
            <SplitIcon className="size-3" />
          )}
          <span className="truncate">{label}</span>
        </PopoverTrigger>
        <PopoverContent align="start" className="flex w-72 flex-col gap-2 p-3">
          <CustomerAutocompleteField
            control={picker.control}
            name="payerId"
            label={t("Bill this charge to")}
            placeholder={
              defaultPayer ? t("Same as shipment ({0})", defaultPayer.label) : t("Same as shipment")
            }
            onOptionChange={handlePick}
          />
          <div className="flex flex-col items-start gap-1">
            <Button
              type="button"
              variant="ghost"
              size="xxs"
              disabled={rows.length === 0}
              onClick={() => {
                writeRows([]);
                setOpen(false);
              }}
            >
              {t("Same as shipment payer")}
            </Button>
            <Button
              type="button"
              variant="ghost"
              size="xxs"
              onClick={() => {
                setOpen(false);
                setSplitOpen(true);
              }}
            >
              <SplitIcon className="size-3" />
              {t("Split this charge…")}
            </Button>
          </div>
        </PopoverContent>
      </Popover>
      {splitOpen ? (
        <ChargeSplitDialog
          open={splitOpen}
          onOpenChange={setSplitOpen}
          name={name}
          chargeAmount={chargeAmount}
          percentOnly={percentOnly}
          defaultPayer={defaultPayer}
          title={splitTitle}
          description={splitDescription}
        />
      ) : null}
    </>
  );
}
