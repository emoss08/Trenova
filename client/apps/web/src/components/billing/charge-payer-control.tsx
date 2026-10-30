"use no memo";
import { CustomerAutocompleteField } from "@/components/autocomplete-fields";
import { allocationPayerLabel } from "@/components/billing/charge-payer-chip";
import { ChargeSplitDialog } from "@/components/billing/charge-split-dialog";
import type { SelectOption as GraphQLSelectOption } from "@/lib/graphql/select-options";
import { selectOptionMetaString } from "@/lib/select-option-meta";
import { SplitIcon, User01Icon } from "@trenova/shared/components/icons";
import { Button } from "@trenova/shared/components/ui/button";
import { Popover, PopoverContent, PopoverTrigger } from "@trenova/shared/components/ui/popover";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import type { DecimalInput } from "@trenova/shared/rules/wire";
import type { ChargeAllocation } from "@trenova/shared/types/shipment";
import { useState } from "react";
import { type FieldValues, useForm, useFormContext, useWatch } from "react-hook-form";

export type ChargePayerControlProps = {
  /** Path of the charge's allocation array, e.g. `additionalCharges.2.allocations`. */
  name: string;
  /** What the charge comes to; null when it is not known here. */
  chargeAmount: DecimalInput;
  /**
   * Whether the charge's rows add up to the whole charge, as `allocationRemainders` reports it.
   * The parent measures every charge it lists in one call and passes each answer down.
   */
  complete: boolean;
  percentOnly?: boolean;
  /** The shipment's own payer: who is billed when the charge has no rows. */
  defaultPayer: { id: string; label: string } | null;
  splitTitle: string;
  splitDescription: string;
  disabled?: boolean;
  className?: string;
};

export function ChargePayerControl({
  name,
  chargeAmount,
  complete,
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

  // One row that adds up to the whole charge bills it to that payer; anything else is a split.
  const single = rows.length === 1 && complete ? rows[0] : null;
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
                rows.length > 0 && "bg-accent-indigo/10 text-accent-indigo-on-subtle",
                className,
              )}
              data-testid="charge-payer-control"
            />
          }
        >
          {single || rows.length === 0 ? (
            <User01Icon className="size-3" />
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
