import { useT } from "@trenova/shared/i18n/use-t";
import {
  AgingDistributionBar,
  type AgingBucketTotals,
} from "@/components/accounting/aging-buckets";
import { CustomerAutocompleteField } from "@/components/autocomplete-fields";
import { AutoCompleteDateField } from "@/components/fields/date-field/date-field";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { usePermission } from "@/hooks/use-permission";
import { queries } from "@/lib/queries";
import { useQuery } from "@tanstack/react-query";
import type { RowSelectionState } from "@tanstack/react-table";
import { Button } from "@trenova/shared/components/ui/button";
import { Card, CardContent } from "@trenova/shared/components/ui/card";
import { EmptyTable } from "@trenova/shared/components/ui/empty-table";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { getEndOfDay } from "@trenova/shared/lib/date";
import { cn, formatCurrency } from "@trenova/shared/lib/utils";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { HandCoinsIcon, XIcon } from "lucide-react";
import { m } from "motion/react";
import { useMemo, useState } from "react";
import { useForm, useWatch } from "react-hook-form";
import { useNavigate } from "react-router";
import { OpenItemsTable } from "./_components/open-items-table";

type FilterValues = {
  customerId: string;
  asOfDate: number | null;
};

const OPEN_ITEM_COLUMNS = [
  { label: "Invoice" },
  { label: "Customer" },
  { label: "Due" },
  { label: "Days", numeric: true },
  { label: "Original", numeric: true },
  { label: "Open", numeric: true },
] as const;

function bucketize(daysPastDue: number): keyof AgingBucketTotals {
  if (daysPastDue <= 0) return "currentMinor";
  if (daysPastDue <= 30) return "days1To30Minor";
  if (daysPastDue <= 60) return "days31To60Minor";
  if (daysPastDue <= 90) return "days61To90Minor";
  return "daysOver90Minor";
}

export function AROpenItemsPage() {
  const t = useT();

  const navigate = useNavigate();
  const { allowed: canRecordPayment } = usePermission(Resource.CustomerPayment, Operation.Create);
  const [rowSelection, setRowSelection] = useState<RowSelectionState>({});

  const filterForm = useForm<FilterValues>({
    defaultValues: { customerId: "", asOfDate: null },
  });
  const customerId = useWatch({ control: filterForm.control, name: "customerId" });
  const asOfValue = useWatch({ control: filterForm.control, name: "asOfDate" });
  const hasActiveFilters = Boolean(customerId || asOfValue);

  const asOfUnix = useMemo(() => {
    if (!asOfValue) return undefined;
    return getEndOfDay(new Date(asOfValue * 1000));
  }, [asOfValue]);

  const {
    data: items,
    isLoading,
    isError,
  } = useQuery(queries.ar.openItems({ customerId: customerId || undefined, asOfDate: asOfUnix }));

  const openItems = useMemo(() => items ?? [], [items]);

  const stats = useMemo(() => {
    const totals: AgingBucketTotals = {
      currentMinor: 0,
      days1To30Minor: 0,
      days31To60Minor: 0,
      days61To90Minor: 0,
      daysOver90Minor: 0,
      totalOpenMinor: 0,
    };
    let overdueAmount = 0;
    let overdueCount = 0;
    let weightedAgeSum = 0;
    for (const item of openItems) {
      totals[bucketize(item.daysPastDue)] += item.openAmountMinor;
      totals.totalOpenMinor += item.openAmountMinor;
      if (item.daysPastDue > 0) {
        overdueAmount += item.openAmountMinor;
        overdueCount += 1;
      }
      weightedAgeSum += item.daysPastDue * item.openAmountMinor;
    }
    return {
      totals,
      overdueAmount,
      overdueCount,
      currentAmount: totals.currentMinor,
      currentCount: openItems.length - overdueCount,
      avgAgeDays: totals.totalOpenMinor > 0 ? weightedAgeSum / totals.totalOpenMinor : 0,
    };
  }, [openItems]);

  const selection = useMemo(() => {
    const selectedIds = Object.keys(rowSelection).filter((id) => rowSelection[id]);
    const selectedItems = openItems.filter((item) => selectedIds.includes(item.invoiceId));
    const customerIds = new Set(selectedItems.map((item) => item.customerId));
    return {
      items: selectedItems,
      totalOpen: selectedItems.reduce((sum, item) => sum + item.openAmountMinor, 0),
      singleCustomerId: customerIds.size === 1 ? [...customerIds][0] : undefined,
    };
  }, [rowSelection, openItems]);

  const handleApplyPayment = () => {
    if (!selection.singleCustomerId) return;
    const invoiceIds = selection.items.map((item) => item.invoiceId).join(",");
    void navigate(
      `/accounting/ar/payments?panelType=create&customerId=${selection.singleCustomerId}&invoiceIds=${invoiceIds}`,
    );
  };

  return (
    <PageLayout
      pageHeaderProps={{
        title: "Open Items",
        description: "Outstanding invoices and their payment status across all customers.",
        actions: canRecordPayment ? (
          <Button
            size="sm"
            onClick={() => void navigate("/accounting/ar/payments?panelType=create")}
          >
            <HandCoinsIcon className="size-4" />
            {t("Record Payment")}
          </Button>
        ) : undefined,
      }}
      className="p-0 px-4 pt-2"
    >
      <div className="space-y-4">
        <div className="flex flex-wrap items-end gap-3">
          <div className="w-65">
            <CustomerAutocompleteField
              control={filterForm.control}
              name="customerId"
              label={t("Customer")}
              placeholder={t("All customers")}
              clearable
            />
          </div>
          <div className="w-45">
            <AutoCompleteDateField
              control={filterForm.control}
              name="asOfDate"
              label={t("As of Date")}
              placeholder={t("Today")}
              clearable
            />
          </div>
        </div>

        {isLoading ? (
          <>
            <div className="grid gap-2.5 md:grid-cols-5">
              {Array.from({ length: 5 }).map((_, i) => (
                <Skeleton key={i} className="h-24 rounded-lg" />
              ))}
            </div>
            <Skeleton className="h-64 w-full rounded-md" />
          </>
        ) : isError ? (
          <div className="rounded-lg border border-red-200 bg-red-50 p-4 text-sm text-red-700 dark:border-red-900 dark:bg-red-950 dark:text-red-300">
            {t("Failed to load open items. Try refreshing the page.")}
          </div>
        ) : (
          <>
            <div className="grid grid-cols-2 gap-2.5 md:grid-cols-5">
              <SummaryCard
                index={0}
                label={t("Total Open")}
                value={formatCurrency(stats.totals.totalOpenMinor / 100)}
                detail={`${openItems.length} ${openItems.length === 1 ? "item" : "items"}`}
              />
              <SummaryCard
                index={1}
                label={t("Current")}
                value={formatCurrency(stats.currentAmount / 100)}
                detail={`${stats.currentCount} items`}
                valueClassName="text-emerald-600 dark:text-emerald-400"
              />
              <SummaryCard
                index={2}
                label={t("Overdue")}
                value={formatCurrency(stats.overdueAmount / 100)}
                detail={`${stats.overdueCount} items`}
                valueClassName={
                  stats.overdueAmount > 0 ? "text-red-600 dark:text-red-400" : undefined
                }
              />
              <SummaryCard
                index={3}
                label={t("Avg Age")}
                value={`${stats.avgAgeDays.toFixed(0)}d`}
                detail={t("weighted by open $")}
              />
              <SummaryCard
                index={4}
                label={t("Count")}
                value={String(openItems.length)}
                detail={t("open invoices")}
              />
            </div>

            {stats.totals.totalOpenMinor > 0 ? (
              <Card className="gap-0 rounded-md py-3">
                <CardContent className="px-4">
                  <AgingDistributionBar totals={stats.totals} />
                </CardContent>
              </Card>
            ) : null}

            {selection.items.length > 0 ? (
              <m.div
                initial={{ opacity: 0, y: 6 }}
                animate={{ opacity: 1, y: 0 }}
                transition={{ duration: 0.2, ease: "easeOut" }}
                className="bg-card flex items-center justify-between rounded-md border px-3 py-2"
              >
                <div className="flex items-center gap-3">
                  <span className="text-xs font-medium tabular-nums">
                    {t("{0} selected · {1}", selection.items.length, formatCurrency(selection.totalOpen / 100))}
                  </span>
                  {!selection.singleCustomerId ? (
                    <span className="text-muted-foreground text-xs">
                      {t("Select invoices from a single customer to apply a payment")}
                    </span>
                  ) : null}
                </div>
                <div className="flex items-center gap-2">
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => setRowSelection({})}
                    className="h-7 text-xs"
                  >
                    <XIcon className="size-3.5" />
                    {t("Clear")}
                  </Button>
                  {canRecordPayment ? (
                    <Button
                      size="sm"
                      onClick={handleApplyPayment}
                      disabled={!selection.singleCustomerId}
                      className="h-7 text-xs"
                    >
                      <HandCoinsIcon className="size-3.5" />
                      {t("Apply Payment")}
                    </Button>
                  ) : null}
                </div>
              </m.div>
            ) : null}

            {openItems.length === 0 ? (
              <EmptyTable
                title={hasActiveFilters ? "Nothing matches" : "Nothing open"}
                description={
                  hasActiveFilters
                    ? "No invoice is open under that filter. Widen it, or clear it to see everything outstanding."
                    : "Every invoice is settled. An invoice sits here from the day it posts until it is paid in full."
                }
                columns={OPEN_ITEM_COLUMNS}
                onClearFilters={hasActiveFilters ? () => filterForm.reset() : undefined}
              />
            ) : (
              <OpenItemsTable
                items={openItems}
                rowSelection={rowSelection}
                onRowSelectionChange={setRowSelection}
              />
            )}
          </>
        )}
      </div>
    </PageLayout>
  );
}

function SummaryCard({
  index,
  label,
  value,
  detail,
  valueClassName,
}: {
  index: number;
  label: string;
  value: string;
  detail?: string;
  valueClassName?: string;
}) {
  return (
    <m.div
      initial={{ opacity: 0, y: 8 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.3, delay: index * 0.05, ease: "easeOut" }}
    >
      <Card className="h-full gap-0 rounded-lg py-3">
        <CardContent className="px-4">
          <p className="text-muted-foreground text-[11px] font-medium tracking-wide uppercase">
            {label}
          </p>
          <p
            className={cn(
              "mt-1 text-2xl font-semibold tracking-tight tabular-nums",
              valueClassName,
            )}
          >
            {value}
          </p>
          {detail ? (
            <p className="text-muted-foreground mt-0.5 text-xs tabular-nums">{detail}</p>
          ) : null}
        </CardContent>
      </Card>
    </m.div>
  );
}
