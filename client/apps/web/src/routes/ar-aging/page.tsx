import type { AgingBucketTotals } from "@/components/accounting/aging-buckets";
import { CustomerAutocompleteField } from "@/components/autocomplete-fields";
import { EmptyState } from "@/components/empty-state";
import { AutoCompleteDateField } from "@/components/fields/date-field/date-field";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { Button } from "@trenova/shared/components/ui/button";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { dateToUnixTimestamp, toDate } from "@trenova/shared/lib/date";
import { queries } from "@/lib/queries";
import { useQuery } from "@tanstack/react-query";
import { format } from "date-fns";
import { ClipboardListIcon, DownloadIcon, FileTextIcon, UsersIcon } from "lucide-react";
import { useMemo } from "react";
import { useForm, useWatch } from "react-hook-form";
import { AgingSummaryHeader } from "./_components/aging-summary-header";
import { AgingTable } from "./_components/aging-table";

type FilterValues = {
  customerId: string;
  asOfDate: number | null;
};

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
  const filterForm = useForm<FilterValues>({
    defaultValues: { customerId: "", asOfDate: null },
  });
  const customerId = useWatch({ control: filterForm.control, name: "customerId" });
  const asOfValue = useWatch({ control: filterForm.control, name: "asOfDate" });

  const asOfUnix = useMemo(() => {
    const date = toDate(asOfValue ?? undefined);
    if (!date) return undefined;
    date.setHours(23, 59, 59, 0);
    return dateToUnixTimestamp(date);
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
    const asOfDate = toDate(asOfValue ?? undefined) ?? new Date();
    anchor.download = `ar-aging-${format(asOfDate, "yyyy-MM-dd")}.csv`;
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
            Export CSV
          </Button>
        ),
      }}
    >
      <div className="mx-4 mt-3 mb-4 space-y-4">
        <div className="flex flex-wrap items-end gap-3">
          <div className="w-[260px]">
            <CustomerAutocompleteField
              control={filterForm.control}
              name="customerId"
              label="Customer"
              placeholder="All customers"
              clearable
            />
          </div>
          <div className="w-[180px]">
            <AutoCompleteDateField
              control={filterForm.control}
              name="asOfDate"
              label="As of Date"
              placeholder="Today"
              clearable
            />
          </div>
        </div>

        <AgingSummaryHeader totals={filteredTotals} isLoading={isLoading} />

        {isLoading ? (
          <Skeleton className="h-64 w-full rounded-md" />
        ) : filteredRows.length === 0 ? (
          <div className="flex justify-center pt-8">
            <EmptyState
              title="No open balances"
              description="No customers have outstanding receivables matching your filters."
              icons={[UsersIcon, ClipboardListIcon, FileTextIcon]}
            />
          </div>
        ) : filteredTotals ? (
          <AgingTable totals={filteredTotals} rows={filteredRows} />
        ) : null}
      </div>
    </PageLayout>
  );
}
