import { AmountDisplay } from "@trenova/shared/components/accounting/amount-display";
import {
  Table,
  TableBody,
  TableCell,
  TableFooter,
  TableHead,
  TableHeader,
  TableRow,
} from "@trenova/shared/components/ui/table";
import type { ARAgingRow, ARAgingSummary } from "@/lib/graphql/accounts-receivable";
import { cn } from "@trenova/shared/lib/utils";
import {
  flexRender,
  getCoreRowModel,
  getSortedRowModel,
  useReactTable,
  type ColumnDef,
  type SortingState,
} from "@tanstack/react-table";
import { ArrowDownIcon, ArrowUpIcon, ChevronsUpDownIcon } from "lucide-react";
import { useMemo, useState } from "react";
import { Link } from "react-router";

const OVERDUE_CELL_CLASSES: Record<string, string> = {
  days1To30Minor: "text-amber-700 dark:text-amber-400",
  days31To60Minor: "text-orange-700 dark:text-orange-400",
  days61To90Minor: "text-red-600 dark:text-red-400",
  daysOver90Minor: "font-medium text-red-700 dark:text-red-400",
};

function bucketColumn(
  key: keyof ARAgingRow["buckets"],
  header: string,
): ColumnDef<ARAgingRow> {
  return {
    id: key,
    header,
    accessorFn: (row) => row.buckets[key],
    cell: ({ row }) => {
      const value = row.original.buckets[key];
      return (
        <AmountDisplay
          value={value}
          className={cn(
            "text-xs",
            value > 0 ? OVERDUE_CELL_CLASSES[key] : "text-muted-foreground/60",
          )}
        />
      );
    },
    meta: { align: "right" },
  };
}

export function AgingTable({
  totals,
  rows,
}: {
  totals: ARAgingSummary["totals"];
  rows: ARAgingRow[];
}) {
  const [sorting, setSorting] = useState<SortingState>([
    { id: "totalOpenMinor", desc: true },
  ]);

  const columns = useMemo<ColumnDef<ARAgingRow>[]>(
    () => [
      {
        id: "customerName",
        header: "Customer",
        accessorFn: (row) => row.customerName,
        cell: ({ row }) => (
          <Link
            to={`/accounting/ar/customer-ledger?customerId=${row.original.customerId}`}
            className="text-xs font-medium hover:underline"
          >
            {row.original.customerName}
          </Link>
        ),
      },
      {
        id: "currentMinor",
        header: "Current",
        accessorFn: (row) => row.buckets.currentMinor,
        cell: ({ row }) => (
          <AmountDisplay
            value={row.original.buckets.currentMinor}
            className={cn(
              "text-xs",
              row.original.buckets.currentMinor > 0
                ? "text-emerald-700 dark:text-emerald-400"
                : "text-muted-foreground/60",
            )}
          />
        ),
        meta: { align: "right" },
      },
      bucketColumn("days1To30Minor", "1–30"),
      bucketColumn("days31To60Minor", "31–60"),
      bucketColumn("days61To90Minor", "61–90"),
      bucketColumn("daysOver90Minor", "90+"),
      {
        id: "totalOpenMinor",
        header: "Total Open",
        accessorFn: (row) => row.buckets.totalOpenMinor,
        cell: ({ row }) => (
          <AmountDisplay
            value={row.original.buckets.totalOpenMinor}
            className="text-xs font-semibold"
          />
        ),
        meta: { align: "right" },
      },
    ],
    [],
  );

  const table = useReactTable({
    data: rows,
    columns,
    state: { sorting },
    onSortingChange: setSorting,
    getCoreRowModel: getCoreRowModel(),
    getSortedRowModel: getSortedRowModel(),
  });

  return (
    <div className="overflow-hidden rounded-md border">
      <Table>
        <TableHeader className="bg-muted/50">
          {table.getHeaderGroups().map((headerGroup) => (
            <TableRow key={headerGroup.id} className="hover:bg-transparent">
              {headerGroup.headers.map((header) => {
                const align = (header.column.columnDef.meta as { align?: string } | undefined)
                  ?.align;
                const sorted = header.column.getIsSorted();
                return (
                  <TableHead
                    key={header.id}
                    className={cn("h-9 text-xs", align === "right" && "text-right")}
                  >
                    <button
                      type="button"
                      onClick={header.column.getToggleSortingHandler()}
                      className={cn(
                        "inline-flex items-center gap-1 font-medium hover:text-foreground",
                        align === "right" && "flex-row-reverse",
                      )}
                    >
                      {flexRender(header.column.columnDef.header, header.getContext())}
                      {sorted === "asc" ? (
                        <ArrowUpIcon className="size-3" />
                      ) : sorted === "desc" ? (
                        <ArrowDownIcon className="size-3" />
                      ) : (
                        <ChevronsUpDownIcon className="size-3 opacity-40" />
                      )}
                    </button>
                  </TableHead>
                );
              })}
            </TableRow>
          ))}
        </TableHeader>
        <TableBody>
          {table.getRowModel().rows.map((row) => (
            <TableRow key={row.id} className="transition-colors">
              {row.getVisibleCells().map((cell) => {
                const align = (cell.column.columnDef.meta as { align?: string } | undefined)
                  ?.align;
                return (
                  <TableCell
                    key={cell.id}
                    className={cn("py-2", align === "right" && "text-right")}
                  >
                    {flexRender(cell.column.columnDef.cell, cell.getContext())}
                  </TableCell>
                );
              })}
            </TableRow>
          ))}
        </TableBody>
        <TableFooter className="sticky bottom-0 bg-muted/40">
          <TableRow className="hover:bg-transparent">
            <TableCell className="py-2 text-xs font-medium">
              Totals · {rows.length} {rows.length === 1 ? "customer" : "customers"}
            </TableCell>
            <TableCell className="py-2 text-right">
              <AmountDisplay value={totals.currentMinor} className="text-xs font-semibold" />
            </TableCell>
            <TableCell className="py-2 text-right">
              <AmountDisplay value={totals.days1To30Minor} className="text-xs font-semibold" />
            </TableCell>
            <TableCell className="py-2 text-right">
              <AmountDisplay value={totals.days31To60Minor} className="text-xs font-semibold" />
            </TableCell>
            <TableCell className="py-2 text-right">
              <AmountDisplay value={totals.days61To90Minor} className="text-xs font-semibold" />
            </TableCell>
            <TableCell className="py-2 text-right">
              <AmountDisplay value={totals.daysOver90Minor} className="text-xs font-semibold" />
            </TableCell>
            <TableCell className="py-2 text-right">
              <AmountDisplay value={totals.totalOpenMinor} className="text-xs font-bold" />
            </TableCell>
          </TableRow>
        </TableFooter>
      </Table>
    </div>
  );
}
