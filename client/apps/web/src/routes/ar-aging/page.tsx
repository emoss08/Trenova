import { useT } from "@trenova/shared/i18n/use-t";
import { EmptyTable } from "@trenova/shared/components/ui/empty-table";
import type { AgingBucketTotals } from "@/components/accounting/aging-buckets";
import { CustomerAutocompleteField } from "@/components/autocomplete-fields";
import { AutoCompleteDateField } from "@/components/fields/date-field/date-field";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { queries } from "@/lib/queries";
import { useQuery } from "@tanstack/react-query";
import { Button } from "@trenova/shared/components/ui/button";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import {
  getEndOfDay,
  toISODateString,
  toUserWallClock,
  userWallClockNow,
} from "@trenova/shared/lib/date";
import { DownloadIcon } from "lucide-react";
import { useMemo } from "react";
import { useForm, useWatch } from "react-hook-form";
import { AgingSummaryHeader } from "./_components/aging-summary-header";
import { AgingTable } from "./_components/aging-table";

type FilterValues = {
  customerId: string;
  asOfDate: number | null;
};

const AGING_COLUMNS = [
  { label: "Customer" },
  { label: "Current", numeric: true },
  { label: "1-30", numeric: true },
  { label: "31-60", numeric: true },
  { label: "61-90", numeric: true },
  { label: "90+", numeric: true },
  { label: "Total open", numeric: true },
] as const;

function toCsv(rows: ReturnType<typeof buildCsvRows>): string {
  return rows.map((row) => row.map((cell) => `"${cell}"`).join(",")).join("\n");
}

function buildCsvRows(
  rows: {
    customerName: string;
    buckets: AgingBucketTotals;
  }[],
): string[][] {
  const header = ["Customer", "Current", "1-30", "31-60", "61-90", "90+", "Total Open"];
  const body = rows.map((row) => [
    row.customerName,
    (row.buckets.currentMinor / 100).toFixed(2),
    (row.buckets.days1To30Minor / 100).toFixed(2),
    (row.buckets.days31To60Minor / 100).toFixed(2),
    (row.buckets.days61To90Minor / 100).toFixed(2),
    (row.buckets.daysOver90Minor / 100).toFixed(2),
    (row.buckets.totalOpenMinor / 100).toFixed(2),
  ]);
  return [header, ...body];
}

export function ARAgingPage() {
  const t = useT();

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

  const { data: summary, isLoading } = useQuery(queries.ar.agingSummary(asOfUnix));

  const filteredRows = useMemo(() => {
    const rows = summary?.rows ?? [];
    if (!customerId) return rows;
    return rows.filter((row) => row.customerId === customerId);
  }, [summary, customerId]);

  const filteredTotals = useMemo(() => {
    if (!summary) return undefined;
    if (!customerId) return summary.totals;
    return filteredRows.reduce(
      (acc, row) => ({
        currentMinor: acc.currentMinor + row.buckets.currentMinor,
        days1To30Minor: acc.days1To30Minor + row.buckets.days1To30Minor,
        days31To60Minor: acc.days31To60Minor + row.buckets.days31To60Minor,
        days61To90Minor: acc.days61To90Minor + row.buckets.days61To90Minor,
        daysOver90Minor: acc.daysOver90Minor + row.buckets.daysOver90Minor,
        totalOpenMinor: acc.totalOpenMinor + row.buckets.totalOpenMinor,
      }),
      {
        currentMinor: 0,
        days1To30Minor: 0,
        days31To60Minor: 0,
        days61To90Minor: 0,
        daysOver90Minor: 0,
        totalOpenMinor: 0,
      },
    );
  }, [summary, customerId, filteredRows]);

  const handleExport = () => {
    const csv = toCsv(buildCsvRows(filteredRows));
    const blob = new Blob([csv], { type: "text/csv;charset=utf-8" });
    const url = URL.createObjectURL(blob);
    const anchor = document.createElement("a");
    anchor.href = url;
    const asOfDate = toUserWallClock(asOfValue) ?? userWallClockNow();
    anchor.download = `ar-aging-${toISODateString(asOfDate)}.csv`;
    anchor.click();
    URL.revokeObjectURL(url);
  };

  return (
    <PageLayout
      pageHeaderProps={{
        title: "AR Aging",
        description: "Receivables aging by customer with drill-down to the ledger.",
        actions: (
          <Button
            variant="outline"
            size="sm"
            onClick={handleExport}
            disabled={filteredRows.length === 0}
          >
            <DownloadIcon className="size-4" />
            {t("Export CSV")}
          </Button>
        ),
      }}
      className="p-0"
    >
      <div className="mx-4 mt-3 mb-4 space-y-4">
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

        <AgingSummaryHeader totals={filteredTotals} isLoading={isLoading} />

        {isLoading ? (
          <Skeleton className="h-64 w-full rounded-md" />
        ) : filteredRows.length === 0 ? (
          <EmptyTable
            title={hasActiveFilters ? "Nothing matches" : "Nothing outstanding"}
            description={
              hasActiveFilters
                ? "No customer carried a balance under that filter. Widen it, or clear it to see everyone who owes."
                : "Every invoice is paid, or none has been raised yet. A customer appears here with their balance by age the moment an invoice posts."
            }
            columns={AGING_COLUMNS}
            onClearFilters={hasActiveFilters ? () => filterForm.reset() : undefined}
          />
        ) : filteredTotals ? (
          <AgingTable totals={filteredTotals} rows={filteredRows} />
        ) : null}
      </div>
    </PageLayout>
  );
}
