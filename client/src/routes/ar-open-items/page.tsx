import { AmountDisplay } from "@/components/accounting/amount-display";
import { CustomerAutocompleteField } from "@/components/autocomplete-fields";
import { EmptyState } from "@/components/empty-state";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { Badge } from "@/components/ui/badge";
import { Input } from "@/components/ui/input";
import { Skeleton } from "@/components/ui/skeleton";
import { queries } from "@/lib/queries";
import { cn, formatCurrency } from "@/lib/utils";
import type { AROpenItem } from "@/types/ar-open-items";
import { useQuery } from "@tanstack/react-query";
import { ClipboardListIcon, FileTextIcon, ReceiptTextIcon } from "lucide-react";
import { useMemo, useState } from "react";
import { useForm } from "react-hook-form";
import { Link } from "react-router";

type FilterValues = {
  customerId: string;
};

function formatDate(unix: number): string {
  return new Date(unix * 1000).toLocaleDateString("en-US", {
    month: "short",
    day: "numeric",
    year: "numeric",
  });
}

function AgingBadge({ daysPastDue }: { daysPastDue: number }) {
  if (daysPastDue <= 0) {
    return <Badge variant="active">Current</Badge>;
  }
  if (daysPastDue <= 30) {
    return <Badge variant="orange">{daysPastDue}d overdue</Badge>;
  }
  if (daysPastDue <= 60) {
    return <Badge variant="inactive">{daysPastDue}d overdue</Badge>;
  }
  return <Badge variant="inactive">{daysPastDue}d overdue</Badge>;
}

function SummaryCard({
  label,
  value,
  count,
  colorClass,
}: {
  label: string;
  value: number;
  count: number;
  colorClass?: string;
}) {
  return (
    <div className="rounded-lg border bg-card px-4 py-3">
      <p className="text-[11px] font-medium tracking-wide text-muted-foreground uppercase">
        {label}
      </p>
      <p className={cn("mt-1 text-2xl font-semibold tracking-tight tabular-nums", colorClass)}>
        {formatCurrency(value / 100)}
      </p>
      <p className="mt-0.5 text-xs text-muted-foreground tabular-nums">
        {count} {count === 1 ? "item" : "items"}
      </p>
    </div>
  );
}

export function AROpenItemsPage() {
  const [asOfDate, setAsOfDate] = useState("");

  const filterForm = useForm<FilterValues>({
    defaultValues: { customerId: "" },
  });

  const customerId = filterForm.watch("customerId");

  const queryParams = useMemo(() => {
    const params: Record<string, string> = {};
    if (customerId) params.customerId = customerId;
    if (asOfDate) {
      const [year, month, day] = asOfDate.split("-").map(Number);
      params.asOfDate = String(Math.floor(new Date(year, month - 1, day).getTime() / 1000));
    }
    return Object.keys(params).length > 0 ? params : undefined;
  }, [customerId, asOfDate]);

  const { data: items, isLoading, isError } = useQuery({
    ...queries.ar.openItems(queryParams),
  });

  const openItems = useMemo(() => items ?? [], [items]);

  const stats = useMemo(() => {
    const current = openItems.filter((i) => i.daysPastDue <= 0);
    const overdue = openItems.filter((i) => i.daysPastDue > 0);
    return {
      totalOpen: openItems.reduce((s, i) => s + i.openAmountMinor, 0),
      totalCount: openItems.length,
      currentAmount: current.reduce((s, i) => s + i.openAmountMinor, 0),
      currentCount: current.length,
      overdueAmount: overdue.reduce((s, i) => s + i.openAmountMinor, 0),
      overdueCount: overdue.length,
    };
  }, [openItems]);

  return (
    <PageLayout
      pageHeaderProps={{
        title: "Open Items",
        description: "Outstanding invoices and their payment status across all customers.",
      }}
    >
      <div className="mx-4 mt-3 mb-4 space-y-4">
        <div className="flex flex-wrap items-end gap-3">
          <div className="w-[260px]">
            <label className="mb-1 block text-2xs font-medium text-muted-foreground">
              Customer
            </label>
            <CustomerAutocompleteField
              control={filterForm.control}
              name="customerId"
              placeholder="All customers"
              clearable
            />
          </div>
          <div className="w-[180px]">
            <label className="mb-1 block text-2xs font-medium text-muted-foreground">
              As of Date
            </label>
            <Input
              type="date"
              value={asOfDate}
              onChange={(e) => setAsOfDate(e.target.value)}
              className="h-9 text-xs"
            />
          </div>
        </div>

        {isLoading ? (
          <>
            <div className="grid gap-2.5 md:grid-cols-3">
              {Array.from({ length: 3 }).map((_, i) => (
                <Skeleton key={i} className="h-24 rounded-lg" />
              ))}
            </div>
            <div className="space-y-2">
              {Array.from({ length: 6 }).map((_, i) => (
                <Skeleton key={i} className="h-12 w-full" />
              ))}
            </div>
          </>
        ) : isError ? (
          <div className="rounded-lg border border-red-200 bg-red-50 p-4 text-sm text-red-700 dark:border-red-900 dark:bg-red-950 dark:text-red-300">
            Failed to load open items. Try refreshing the page.
          </div>
        ) : (
          <>
            <div className="grid gap-2.5 md:grid-cols-3">
              <SummaryCard
                label="Total Open"
                value={stats.totalOpen}
                count={stats.totalCount}
              />
              <SummaryCard
                label="Current"
                value={stats.currentAmount}
                count={stats.currentCount}
                colorClass="text-green-600 dark:text-green-400"
              />
              <SummaryCard
                label="Overdue"
                value={stats.overdueAmount}
                count={stats.overdueCount}
                colorClass="text-red-600 dark:text-red-400"
              />
            </div>

            {openItems.length === 0 ? (
              <div className="flex justify-center pt-8">
                <EmptyState
                  title="No open items"
                  description="There are no outstanding invoices matching your filters."
                  icons={[ClipboardListIcon, FileTextIcon, ReceiptTextIcon]}
                />
              </div>
            ) : (
              <div className="overflow-hidden rounded-lg border">
                <div className="overflow-x-auto">
                  <table className="w-full text-sm">
                    <thead className="bg-muted/50 text-left text-muted-foreground">
                      <tr>
                        <th className="px-3 py-2.5 text-xs font-medium">Invoice</th>
                        <th className="px-3 py-2.5 text-xs font-medium">Customer</th>
                        <th className="px-3 py-2.5 text-xs font-medium">Type</th>
                        <th className="px-3 py-2.5 text-xs font-medium">PRO / BOL</th>
                        <th className="px-3 py-2.5 text-xs font-medium">Invoice Date</th>
                        <th className="px-3 py-2.5 text-xs font-medium">Due Date</th>
                        <th className="px-3 py-2.5 text-xs font-medium">Status</th>
                        <th className="px-3 py-2.5 text-right text-xs font-medium">Total</th>
                        <th className="px-3 py-2.5 text-right text-xs font-medium">Applied</th>
                        <th className="px-3 py-2.5 text-right text-xs font-medium">Open</th>
                      </tr>
                    </thead>
                    <tbody>
                      {openItems.map((item: AROpenItem) => (
                        <tr
                          key={item.invoiceId}
                          className="border-t transition-colors hover:bg-muted/40"
                        >
                          <td className="px-3 py-2.5 font-mono text-xs font-medium">
                            {item.invoiceNumber}
                          </td>
                          <td className="px-3 py-2.5">
                            <Link
                              to={`/accounting/ar/customer-statement/${item.customerId}`}
                              className="text-xs font-medium hover:underline"
                            >
                              {item.customerName}
                            </Link>
                          </td>
                          <td className="px-3 py-2.5 text-xs text-muted-foreground capitalize">
                            {item.billType}
                          </td>
                          <td className="px-3 py-2.5 font-mono text-xs text-muted-foreground">
                            {item.shipmentProNumber || item.shipmentBolNumber || "\u2014"}
                          </td>
                          <td className="px-3 py-2.5 text-xs">{formatDate(item.invoiceDate)}</td>
                          <td className="px-3 py-2.5 text-xs">{formatDate(item.dueDate)}</td>
                          <td className="px-3 py-2.5">
                            <AgingBadge daysPastDue={item.daysPastDue} />
                          </td>
                          <td className="px-3 py-2.5 text-right">
                            <AmountDisplay value={item.totalAmountMinor} className="text-xs" />
                          </td>
                          <td className="px-3 py-2.5 text-right">
                            <AmountDisplay
                              value={item.appliedAmountMinor}
                              className="text-xs text-muted-foreground"
                            />
                          </td>
                          <td className="px-3 py-2.5 text-right">
                            <AmountDisplay
                              value={item.openAmountMinor}
                              className="text-xs font-semibold"
                            />
                          </td>
                        </tr>
                      ))}
                    </tbody>
                    <tfoot className="border-t bg-muted/30">
                      <tr>
                        <td colSpan={7} className="px-3 py-2.5 text-right text-xs font-medium">
                          Totals
                        </td>
                        <td className="px-3 py-2.5 text-right">
                          <AmountDisplay
                            value={openItems.reduce((s, i) => s + i.totalAmountMinor, 0)}
                            className="text-xs font-semibold"
                          />
                        </td>
                        <td className="px-3 py-2.5 text-right">
                          <AmountDisplay
                            value={openItems.reduce((s, i) => s + i.appliedAmountMinor, 0)}
                            className="text-xs font-semibold"
                          />
                        </td>
                        <td className="px-3 py-2.5 text-right">
                          <AmountDisplay
                            value={stats.totalOpen}
                            className="text-xs font-bold"
                          />
                        </td>
                      </tr>
                    </tfoot>
                  </table>
                </div>
              </div>
            )}
          </>
        )}
      </div>
    </PageLayout>
  );
}
