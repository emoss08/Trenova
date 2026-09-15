import { useApiMutation } from "@/hooks/use-api-mutation";
import { applyCreditMemo } from "@/lib/graphql/invoice";
import { queries } from "@/lib/queries";
import { invalidateInvoiceQueries } from "@/lib/queries/invoice";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { AmountDisplay } from "@trenova/shared/components/accounting/amount-display";
import { Button } from "@trenova/shared/components/ui/button";
import { Checkbox } from "@trenova/shared/components/ui/checkbox";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@trenova/shared/components/ui/dialog";
import { Input } from "@trenova/shared/components/ui/input";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@trenova/shared/components/ui/table";
import { useT } from "@trenova/shared/i18n/use-t";
import { getEndOfDay } from "@trenova/shared/lib/date";
import { toMinorUnits } from "@trenova/shared/lib/charge-split";
import { formatCurrency } from "@trenova/shared/lib/utils";
import type { Invoice } from "@trenova/shared/types/invoice";
import { useEffect, useMemo, useState } from "react";
import { toast } from "sonner";

type Allocation = { checked: boolean; amount: string };

/**
 * Settles the customer's open invoices with a posted credit memo. Each row is
 * capped at what the invoice still owes and the total at what the memo has
 * left, so the form cannot ask the server for something it will refuse.
 */
export function ApplyCreditMemoDialog({
  creditMemo,
  creditRemaining,
  open,
  onOpenChange,
}: {
  creditMemo: Invoice;
  creditRemaining: number;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const t = useT();
  const queryClient = useQueryClient();
  const [allocations, setAllocations] = useState<Record<string, Allocation>>({});

  const openItems = useQuery({
    ...queries.ar.openItems({ customerId: creditMemo.customerId }),
    enabled: open,
  });
  const items = useMemo(
    () => (openItems.data ?? []).filter((item) => item.invoiceId !== creditMemo.id),
    [openItems.data, creditMemo.id],
  );

  useEffect(() => {
    if (open) setAllocations({});
  }, [open]);

  const remainingMinor = toMinorUnits(creditRemaining);
  const selected = items
    .map((item) => ({ item, allocation: allocations[item.invoiceId] }))
    .filter((row) => row.allocation?.checked && Number(row.allocation.amount) > 0);
  const totalMinor = selected.reduce(
    (sum, row) => sum + toMinorUnits(Number(row.allocation.amount)),
    0,
  );
  const overRemaining = totalMinor > remainingMinor;
  const overOpen = selected.some(
    (row) => toMinorUnits(Number(row.allocation.amount)) > row.item.openAmountMinor,
  );

  const mutation = useApiMutation({
    resourceName: "credit memo application",
    mutationFn: () =>
      applyCreditMemo({
        creditMemoId: creditMemo.id,
        accountingDate: getEndOfDay(new Date()),
        applications: selected.map((row) => ({
          invoiceId: row.item.invoiceId,
          appliedAmountMinor: toMinorUnits(Number(row.allocation.amount)),
        })),
      }),
    onSuccess: (applied) => {
      invalidateInvoiceQueries(queryClient);
      toast.success(
        t(
          "{0} applied to {1, plural, one {# invoice} other {# invoices}}",
          creditMemo.number,
          applied.length,
        ),
      );
      onOpenChange(false);
    },
  });

  const toggle = (invoiceId: string, checked: boolean, openAmountMinor: number) => {
    setAllocations((prev) => ({
      ...prev,
      [invoiceId]: {
        checked,
        amount:
          prev[invoiceId]?.amount ||
          (Math.min(openAmountMinor, remainingMinor - totalMinor) / 100).toFixed(2),
      },
    }));
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[640px]">
        <DialogHeader>
          <DialogTitle>{t("Apply {0} to invoices", creditMemo.number)}</DialogTitle>
          <DialogDescription>
            {t(
              "{0} remains on this credit memo. Pick the open invoices it settles and how much of each.",
              formatCurrency(creditRemaining, creditMemo.currencyCode),
            )}
          </DialogDescription>
        </DialogHeader>

        {openItems.isLoading ? (
          <Skeleton className="h-40 w-full" />
        ) : items.length === 0 ? (
          <p className="text-muted-foreground rounded-md border border-dashed p-6 text-center text-sm">
            {t("This customer has no open invoices to apply the credit to.")}
          </p>
        ) : (
          <div className="max-h-[50dvh] overflow-auto rounded-md border">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead className="w-8" />
                  <TableHead>{t("Invoice")}</TableHead>
                  <TableHead className="text-right">{t("Open")}</TableHead>
                  <TableHead className="w-36 text-right">{t("Apply")}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {items.map((item) => {
                  const allocation = allocations[item.invoiceId];
                  return (
                    <TableRow key={item.invoiceId}>
                      <TableCell>
                        <Checkbox
                          checked={allocation?.checked ?? false}
                          onCheckedChange={(checked) =>
                            toggle(item.invoiceId, checked === true, item.openAmountMinor)
                          }
                          aria-label={t("Apply to {0}", item.invoiceNumber)}
                        />
                      </TableCell>
                      <TableCell className="font-mono text-xs">{item.invoiceNumber}</TableCell>
                      <TableCell className="text-right text-xs">
                        <AmountDisplay
                          value={item.openAmountMinor}
                          currency={creditMemo.currencyCode}
                        />
                      </TableCell>
                      <TableCell className="text-right">
                        <Input
                          type="number"
                          inputMode="decimal"
                          min={0}
                          step="0.01"
                          className="h-7 text-right text-xs"
                          disabled={!allocation?.checked}
                          value={allocation?.amount ?? ""}
                          aria-label={t("Amount to apply to {0}", item.invoiceNumber)}
                          onChange={(event) =>
                            setAllocations((prev) => ({
                              ...prev,
                              [item.invoiceId]: {
                                checked: true,
                                amount: event.target.value,
                              },
                            }))
                          }
                        />
                      </TableCell>
                    </TableRow>
                  );
                })}
              </TableBody>
            </Table>
          </div>
        )}

        <div className="flex items-center justify-between text-sm">
          <span className="text-muted-foreground">{t("Applying")}</span>
          <span
            className={
              overRemaining || overOpen
                ? "font-semibold text-red-600 tabular-nums dark:text-red-400"
                : "font-semibold tabular-nums"
            }
          >
            {formatCurrency(totalMinor / 100, creditMemo.currencyCode)}
            {overRemaining ? ` · ${t("exceeds the credit remaining")}` : ""}
            {overOpen ? ` · ${t("exceeds an invoice's open balance")}` : ""}
          </span>
        </div>

        <DialogFooter>
          <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
            {t("Cancel")}
          </Button>
          <Button
            type="button"
            disabled={selected.length === 0 || overRemaining || overOpen || mutation.isPending}
            onClick={() => mutation.mutate()}
          >
            {t("Apply credit")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
