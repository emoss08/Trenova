import { NumberField } from "@/components/fields/number-field";
import { DescriptionEmpty } from "@trenova/shared/components/ui/description-list";
import { Switch } from "@trenova/shared/components/ui/switch";
import { TableCell, TableRow } from "@trenova/shared/components/ui/table";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixDateMedium } from "@trenova/shared/lib/date";
import { useState } from "react";
import { useFormContext, useWatch } from "react-hook-form";
import { retentionCutoff, type DataRetentionFormValues } from "../data-retention-form";

type RetentionField = keyof DataRetentionFormValues;

/**
 * One window in the policy: the data, how long it is kept, the date the next
 * purge reaches back to, and what happens to what is older. A window the
 * server reads 0 on as "no limit" carries a Forever switch rather than asking
 * anyone to know that 0 means forever; turning the limit back on restores the
 * last one it had.
 */
export function RetentionPolicyRow({
  name,
  label,
  description,
  minDays,
  outcome,
  noLimit,
  now,
  disabled,
}: {
  name: RetentionField;
  label: string;
  description: string;
  minDays?: number;
  outcome: string;
  noLimit?: { fallbackDays: number };
  now: number;
  disabled: boolean;
}) {
  const t = useT();
  const { control, setValue } = useFormContext<DataRetentionFormValues>();
  const days = useWatch({ control, name });
  const unlimited = noLimit !== undefined && days === 0;
  const [lastLimit, setLastLimit] = useState(() =>
    days > 0 ? days : (noLimit?.fallbackDays ?? 0),
  );
  const cutoff = retentionCutoff(days, now);

  const setUnlimited = (on: boolean) => {
    if (on && days > 0) {
      setLastLimit(days);
    }
    setValue(name, on ? 0 : lastLimit, { shouldDirty: true, shouldValidate: true });
  };

  return (
    <TableRow className="align-top hover:bg-transparent">
      <TableCell className="py-3 pl-4 whitespace-normal">
        <div className="flex max-w-sm flex-col gap-0.5">
          <span className="text-sm font-medium">{label}</span>
          <span className="text-foreground-muted text-xs leading-relaxed">{description}</span>
          {minDays !== undefined && (
            <span className="text-foreground-subtle text-xs">
              {t("At least {0, plural, one {# day} other {# days}}", minDays)}
            </span>
          )}
        </div>
      </TableCell>
      <TableCell className="py-3">
        <div className="flex w-44 flex-col gap-2">
          {noLimit !== undefined && (
            <label className="flex h-8 items-center gap-2 text-xs font-medium">
              <Switch
                size="sm"
                aria-label={t("Keep forever")}
                checked={unlimited}
                disabled={disabled}
                onCheckedChange={setUnlimited}
              />
              {t("Forever")}
            </label>
          )}
          {!unlimited && (
            <NumberField
              control={control}
              name={name}
              aria-label={label}
              sideText={t("days")}
              placeholder={t("Days")}
              disabled={disabled}
              min={noLimit !== undefined ? 0 : minDays}
            />
          )}
        </div>
      </TableCell>
      <TableCell className="text-foreground-muted py-3 pt-5 tabular-nums">
        {cutoff === null ? <DescriptionEmpty /> : formatUnixDateMedium(cutoff)}
      </TableCell>
      <TableCell className="text-foreground-muted py-3 pt-5 pr-4">
        {unlimited ? t("Kept") : outcome}
      </TableCell>
    </TableRow>
  );
}
