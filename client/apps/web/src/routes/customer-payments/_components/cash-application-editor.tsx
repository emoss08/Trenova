import { AgingBadge } from "@/components/accounting/aging-buckets";
import { AmountDisplay } from "@trenova/shared/components/accounting/amount-display";
import { NumberField } from "@/components/fields/number-field";
import { Button } from "@trenova/shared/components/ui/button";
import { Checkbox } from "@trenova/shared/components/ui/checkbox";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@trenova/shared/components/ui/table";
import { computeApplicationTotals } from "@/lib/cash-application";
import { cn, formatCurrency } from "@trenova/shared/lib/utils";
import type { CashApplicationRow } from "@trenova/shared/types/customer-payment";
import { WandSparklesIcon } from "lucide-react";
import { useMemo } from "react";
import { useFormContext, useWatch } from "react-hook-form";

type CashApplicationFormShape = {
  applications: CashApplicationRow[];
};

function formatDate(unix: number): string {
  if (!unix) return "—";
  return new Date(unix * 1000).toLocaleDateString("en-US", {
    month: "short",
    day: "numeric",
    year: "numeric",
  });
}

export function CashApplicationEditor({
  budgetMinor,
  budgetLabel,
  onAutoApply,
  isLoadingItems,
  emptyMessage,
}: {
  budgetMinor: number;
  budgetLabel: string;
  onAutoApply: () => void;
  isLoadingItems: boolean;
  emptyMessage: string;
}) {
  const { control, setValue } = useFormContext<CashApplicationFormShape>();
  const watchedRows = useWatch({ control, name: "applications" });
  const rows = useMemo(() => watchedRows ?? [], [watchedRows]);

  const totals = useMemo(
    () => computeApplicationTotals(rows, budgetMinor),
    [rows, budgetMinor],
  );

  return (
    <div className="space-y-3">
      <div className="flex items-center justify-between">
        <p className="text-sm font-medium">Apply to open invoices</p>
        <Button
          type="button"
          variant="outline"
          size="sm"
          onClick={onAutoApply}
          disabled={budgetMinor <= 0 || rows.length === 0}
          className="h-7 text-xs"
        >
          <WandSparklesIcon className="size-3.5" />
          Auto-apply oldest first
        </Button>
      </div>

      {isLoadingItems ? (
        <div className="flex h-32 items-center justify-center rounded-md border text-sm text-muted-foreground">
          Loading open invoices…
        </div>
      ) : rows.length === 0 ? (
        <div className="flex h-32 items-center justify-center rounded-md border border-dashed text-sm text-muted-foreground">
          {emptyMessage}
        </div>
      ) : (
        <div className="max-h-72 overflow-y-auto rounded-md border">
          <Table>
            <TableHeader className="sticky top-0 z-10 bg-muted/80 backdrop-blur">
              <TableRow className="hover:bg-transparent">
                <TableHead className="h-8 w-8" />
                <TableHead className="h-8 text-xs">Invoice</TableHead>
                <TableHead className="h-8 text-xs">Due</TableHead>
                <TableHead className="h-8 text-right text-xs">Open</TableHead>
                <TableHead className="h-8 w-32 text-right text-xs">Applied</TableHead>
                <TableHead className="h-8 w-32 text-right text-xs">Short-pay</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {rows.map((row, index) => {
                const isOverApplied = totals.overAppliedRows.includes(index);
                return (
                  <TableRow
                    key={row.invoiceId}
                    className={cn(
                      "transition-colors",
                      isOverApplied && "bg-red-500/5",
                      !row.checked && "opacity-60",
                    )}
                  >
                    <TableCell className="py-1.5">
                      <Checkbox
                        checked={row.checked}
                        onCheckedChange={(checked) => {
                          setValue(`applications.${index}.checked`, checked === true, {
                            shouldDirty: true,
                          });
                          if (checked !== true) {
                            setValue(`applications.${index}.appliedAmount`, 0);
                            setValue(`applications.${index}.shortPayAmount`, 0);
                          }
                        }}
                        aria-label={`Apply to invoice ${row.invoiceNumber}`}
                      />
                    </TableCell>
                    <TableCell className="py-1.5">
                      <div className="flex flex-col">
                        <span className="font-mono text-xs font-medium">
                          {row.invoiceNumber}
                        </span>
                        <span className="text-[11px] text-muted-foreground">
                          {formatDate(row.invoiceDate)}
                        </span>
                      </div>
                    </TableCell>
                    <TableCell className="py-1.5">
                      <div className="flex flex-col gap-0.5">
                        <span className="text-xs">{formatDate(row.dueDate)}</span>
                        <AgingBadge daysPastDue={row.daysPastDue} />
                      </div>
                    </TableCell>
                    <TableCell className="py-1.5 text-right">
                      <AmountDisplay value={row.openAmountMinor} className="text-xs" />
                      {isOverApplied ? (
                        <p className="text-[10px] text-red-600 dark:text-red-400">
                          exceeds open amount
                        </p>
                      ) : null}
                    </TableCell>
                    <TableCell className="py-1.5">
                      <NumberField
                        control={control}
                        name={`applications.${index}.appliedAmount`}
                        aria-label={`Applied amount for ${row.invoiceNumber}`}
                        placeholder="0.00"
                        decimalScale={2}
                        fixedDecimalScale
                        disabled={!row.checked}
                      />
                    </TableCell>
                    <TableCell className="py-1.5">
                      <NumberField
                        control={control}
                        name={`applications.${index}.shortPayAmount`}
                        aria-label={`Short-pay amount for ${row.invoiceNumber}`}
                        placeholder="0.00"
                        decimalScale={2}
                        fixedDecimalScale
                        disabled={!row.checked}
                      />
                    </TableCell>
                  </TableRow>
                );
              })}
            </TableBody>
          </Table>
        </div>
      )}

      <div
        className={cn(
          "flex flex-wrap items-center justify-between gap-2 rounded-md border bg-muted/30 px-3 py-2",
          totals.isOverBudget && "border-red-500/50 bg-red-500/5",
        )}
      >
        <div className="flex flex-wrap gap-x-5 gap-y-1 text-xs">
          <SummaryStat label={budgetLabel} value={budgetMinor} />
          <SummaryStat label="Applied" value={totals.appliedMinor} />
          <SummaryStat label="Short-pay" value={totals.shortPayMinor} />
          <SummaryStat
            label="Unapplied"
            value={totals.unappliedMinor}
            className={
              totals.unappliedMinor > 0 ? "text-amber-600 dark:text-amber-400" : undefined
            }
          />
        </div>
        {totals.isOverBudget ? (
          <p className="text-xs font-medium text-red-600 dark:text-red-400">
            Applied exceeds {budgetLabel.toLowerCase()} by{" "}
            {formatCurrency((totals.appliedMinor - budgetMinor) / 100)}
          </p>
        ) : null}
      </div>
    </div>
  );
}

function SummaryStat({
  label,
  value,
  className,
}: {
  label: string;
  value: number;
  className?: string;
}) {
  return (
    <span className="inline-flex items-baseline gap-1.5">
      <span className="text-muted-foreground">{label}</span>
      <span className={cn("font-semibold tabular-nums", className)}>
        {formatCurrency(value / 100)}
      </span>
    </span>
  );
}
