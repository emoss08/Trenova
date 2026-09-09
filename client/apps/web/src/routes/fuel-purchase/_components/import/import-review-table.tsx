import {
  IMPORT_ROW_FILTER_LABELS,
  IMPORT_ROW_FILTER_STATUSES,
  IMPORT_ROW_STATUS_LABELS,
  type ImportRowFilter,
} from "@/lib/fuel-purchase-import";
import {
  fetchFuelPurchaseImportRows,
  FUEL_PURCHASE_IMPORT_ROWS_KEY,
  type FuelPurchaseImportBatch,
  type FuelPurchaseImportRow,
} from "@/lib/graphql/fuel-purchase-import";
import type { FuelPurchaseImportRowStatus } from "@trenova/graphql/generated/graphql";
import { useInfiniteQuery } from "@tanstack/react-query";
import { Badge, type BadgeVariant } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@trenova/shared/components/ui/table";
import { formatUnixDateTimeOrDash } from "@trenova/shared/lib/date";
import { cn, formatCurrency } from "@trenova/shared/lib/utils";
import { useMemo } from "react";

const PAGE_SIZE = 50;

const FILTERS: readonly ImportRowFilter[] = ["all", "new", "duplicates", "errors"];

const STATUS_VARIANT: Record<FuelPurchaseImportRowStatus, BadgeVariant> = {
  New: "active",
  DuplicateInFile: "warning",
  AlreadyImported: "warning",
  Error: "inactive",
  Committed: "info",
  Skipped: "outline",
};

type ImportReviewTableProps = {
  batch: FuelPurchaseImportBatch;
  filter: ImportRowFilter;
  onFilterChange?: (filter: ImportRowFilter) => void;
  showFilters?: boolean;
};

function filterCount(batch: FuelPurchaseImportBatch, filter: ImportRowFilter): number {
  const summary = batch.summary;
  switch (filter) {
    case "all":
      return batch.rowCount;
    case "new":
      return summary?.newCount ?? 0;
    case "duplicates":
      return (summary?.duplicateInFileCount ?? 0) + (summary?.alreadyImportedCount ?? 0);
    case "errors":
      return summary?.errorCount ?? batch.errorCount;
    default:
      return 0;
  }
}

function rowNote(row: FuelPurchaseImportRow): string {
  if (row.error) return row.error;
  return row.resolutionNotes.join(" ");
}

export function ImportReviewTable({
  batch,
  filter,
  onFilterChange,
  showFilters = true,
}: ImportReviewTableProps) {
  const query = useInfiniteQuery({
    queryKey: [FUEL_PURCHASE_IMPORT_ROWS_KEY, batch.id, batch.version, filter],
    queryFn: ({ pageParam, signal }) =>
      fetchFuelPurchaseImportRows(
        batch.id,
        { first: PAGE_SIZE, after: pageParam, statuses: IMPORT_ROW_FILTER_STATUSES[filter] },
        { signal },
      ),
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (lastPage) =>
      lastPage.pageInfo.hasNextPage ? (lastPage.pageInfo.endCursor ?? undefined) : undefined,
  });

  const rows = useMemo(
    () => query.data?.pages.flatMap((page) => page.edges.map((edge) => edge.node)) ?? [],
    [query.data],
  );
  const total = query.data?.pages[0]?.totalCount ?? null;

  return (
    <div className="flex flex-col gap-2">
      {showFilters ? (
        <div className="flex flex-wrap items-center gap-1" role="group" aria-label="Row filter">
          {FILTERS.map((option) => (
            <Button
              key={option}
              type="button"
              size="sm"
              variant={filter === option ? "default" : "ghost"}
              aria-pressed={filter === option}
              onClick={() => onFilterChange?.(option)}
            >
              {IMPORT_ROW_FILTER_LABELS[option]}
              <span
                className={cn("ml-1 tabular-nums", filter !== option && "text-muted-foreground")}
              >
                {filterCount(batch, option)}
              </span>
            </Button>
          ))}
        </div>
      ) : null}

      <div className="overflow-hidden rounded-lg border">
        <div className="max-h-80 overflow-auto">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className="w-12 text-xs">Line</TableHead>
                <TableHead className="text-xs">Status</TableHead>
                <TableHead className="text-xs">Purchased</TableHead>
                <TableHead className="text-xs">Tractor</TableHead>
                <TableHead className="text-xs">Where</TableHead>
                <TableHead className="text-xs">Fuel</TableHead>
                <TableHead className="text-right text-xs">Gallons</TableHead>
                <TableHead className="text-right text-xs">Amount</TableHead>
                <TableHead className="text-xs">Reference</TableHead>
                <TableHead className="text-xs">Note</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {query.isLoading ? (
                Array.from({ length: 3 }, (_, index) => (
                  <TableRow key={index}>
                    <TableCell colSpan={10}>
                      <Skeleton className="h-4 w-full" />
                    </TableCell>
                  </TableRow>
                ))
              ) : query.isError ? (
                <TableRow>
                  <TableCell colSpan={10} className="text-destructive text-xs">
                    The rows could not be loaded.{" "}
                    <Button
                      type="button"
                      variant="link"
                      size="sm"
                      className="h-auto px-1 py-0 text-xs"
                      onClick={() => void query.refetch()}
                    >
                      Retry
                    </Button>
                  </TableCell>
                </TableRow>
              ) : rows.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={10} className="text-muted-foreground text-xs">
                    No rows in this view.
                  </TableCell>
                </TableRow>
              ) : (
                rows.map((row) => (
                  <TableRow key={row.id} data-status={row.status}>
                    <TableCell className="font-mono text-xs tabular-nums">
                      {row.rowNumber}
                    </TableCell>
                    <TableCell>
                      <Badge
                        variant={STATUS_VARIANT[row.status]}
                        className="px-1.5 py-0 text-[10px] whitespace-nowrap"
                      >
                        {IMPORT_ROW_STATUS_LABELS[row.status]}
                      </Badge>
                    </TableCell>
                    <TableCell className="text-xs whitespace-nowrap tabular-nums">
                      {formatUnixDateTimeOrDash(row.parsed?.purchasedAt)}
                    </TableCell>
                    <TableCell className="text-xs">
                      {row.resolvedTractor?.code ?? row.parsed?.tractorCode ?? "—"}
                    </TableCell>
                    <TableCell className="text-xs">
                      {[row.parsed?.jurisdictionCode, row.parsed?.vendorCity]
                        .filter(Boolean)
                        .join(" · ") || "—"}
                    </TableCell>
                    <TableCell className="text-xs">{row.parsed?.fuelType ?? "—"}</TableCell>
                    <TableCell className="text-right text-xs tabular-nums">
                      {row.parsed?.gallons ?? "—"}
                    </TableCell>
                    <TableCell className="text-right text-xs tabular-nums">
                      {row.parsed?.totalAmount
                        ? formatCurrency(
                            Number(row.parsed.totalAmount),
                            row.parsed.currencyCode ?? batch.defaultCurrency,
                          )
                        : "—"}
                    </TableCell>
                    <TableCell className="font-mono text-xs">
                      {row.transactionReference ?? row.parsed?.transactionReference ?? "—"}
                    </TableCell>
                    <TableCell
                      className={cn(
                        "max-w-64 text-xs",
                        row.error ? "text-destructive" : "text-muted-foreground",
                      )}
                    >
                      {rowNote(row) || "—"}
                    </TableCell>
                  </TableRow>
                ))
              )}
            </TableBody>
          </Table>
        </div>
      </div>

      <div className="flex items-center justify-between">
        <p className="text-muted-foreground text-xs">
          {total !== null
            ? `Showing ${rows.length} of ${total} ${total === 1 ? "row" : "rows"}`
            : `Showing ${rows.length} rows`}
        </p>
        {query.hasNextPage ? (
          <Button
            type="button"
            variant="outline"
            size="sm"
            isLoading={query.isFetchingNextPage}
            onClick={() => void query.fetchNextPage()}
          >
            Load more
          </Button>
        ) : null}
      </div>
    </div>
  );
}
