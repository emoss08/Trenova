import { AgingBadge } from "@/components/accounting/aging-buckets";
import { AmountDisplay } from "@/components/accounting/amount-display";
import { PlainSettlementStatusBadge } from "@/components/status-badge";
import { Badge } from "@/components/ui/badge";
import { Checkbox } from "@/components/ui/checkbox";
import {
  Table,
  TableBody,
  TableCell,
  TableFooter,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import type { AROpenItem } from "@/lib/graphql/accounts-receivable";
import { cn } from "@/lib/utils";
import type { SettlementStatus } from "@/types/invoice";
import {
  flexRender,
  getCoreRowModel,
  getSortedRowModel,
  useReactTable,
  type ColumnDef,
  type RowSelectionState,
  type SortingState,
} from "@tanstack/react-table";
import { ArrowDownIcon, ArrowUpIcon, ChevronsUpDownIcon } from "lucide-react";
import { useMemo, useState } from "react";
import { Link } from "react-router";

function formatDate(unix: number): string {
  if (!unix) return "—";
  return new Date(unix * 1000).toLocaleDateString("en-US", {
    month: "short",
    day: "numeric",
    year: "numeric",
  });
}

export function OpenItemsTable({
  items,
  rowSelection,
  onRowSelectionChange,
}: {
  items: AROpenItem[];
  rowSelection: RowSelectionState;
  onRowSelectionChange: (
    updater: RowSelectionState | ((old: RowSelectionState) => RowSelectionState),
  ) => void;
}) {
  const [sorting, setSorting] = useState<SortingState>([{ id: "dueDate", desc: false }]);

  const columns = useMemo<ColumnDef<AROpenItem>[]>(
    () => [
      {
        id: "select",
        enableSorting: false,
        header: ({ table }) => (
          <Checkbox
            checked={table.getIsAllRowsSelected()}
            indeterminate={table.getIsSomeRowsSelected()}
            onCheckedChange={(checked) => table.toggleAllRowsSelected(checked === true)}
            aria-label="Select all"
          />
        ),
        cell: ({ row }) => (
          <Checkbox
            checked={row.getIsSelected()}
            onCheckedChange={(checked) => row.toggleSelected(checked === true)}
            aria-label={`Select invoice ${row.original.invoiceNumber}`}
            onClick={(e) => e.stopPropagation()}
          />
        ),
      },
      {
        id: "invoiceNumber",
        header: "Invoice",
        accessorFn: (row) => row.invoiceNumber,
        cell: ({ row }) => (
          <span className="font-mono text-xs font-medium">{row.original.invoiceNumber}</span>
        ),
      },
      {
        id: "customerName",
        header: "Customer",
        accessorFn: (row) => row.customerName,
        cell: ({ row }) => (
          <Link
            to={`/accounting/ar/customer-statement/${row.original.customerId}`}
            className="text-xs font-medium hover:underline"
            onClick={(e) => e.stopPropagation()}
          >
            {row.original.customerName}
          </Link>
        ),
      },
      {
        id: "billType",
        header: "Type",
        accessorFn: (row) => row.billType,
        cell: ({ row }) => (
          <span className="text-xs text-muted-foreground capitalize">
            {row.original.billType}
          </span>
        ),
      },
      {
        id: "reference",
        header: "PRO / BOL",
        enableSorting: false,
        cell: ({ row }) => (
          <span className="font-mono text-xs text-muted-foreground">
            {row.original.shipmentProNumber || row.original.shipmentBol || "—"}
          </span>
        ),
      },
      {
        id: "invoiceDate",
        header: "Invoice Date",
        accessorFn: (row) => row.invoiceDate,
        cell: ({ row }) => (
          <span className="text-xs">{formatDate(row.original.invoiceDate)}</span>
        ),
      },
      {
        id: "dueDate",
        header: "Due Date",
        accessorFn: (row) => row.dueDate,
        cell: ({ row }) => <span className="text-xs">{formatDate(row.original.dueDate)}</span>,
      },
      {
        id: "status",
        header: "Status",
        accessorFn: (row) => row.daysPastDue,
        cell: ({ row }) => (
          <div className="flex flex-wrap items-center gap-1">
            <AgingBadge daysPastDue={row.original.daysPastDue} />
            <PlainSettlementStatusBadge
              status={row.original.settlementStatus as SettlementStatus}
            />
            {row.original.disputeStatus === "Disputed" ? (
              <Badge variant="orange">Disputed</Badge>
            ) : null}
            {row.original.hasShortPay ? <Badge variant="inactive">Short-paid</Badge> : null}
          </div>
        ),
      },
      {
        id: "totalAmountMinor",
        header: "Total",
        accessorFn: (row) => row.totalAmountMinor,
        cell: ({ row }) => (
          <AmountDisplay value={row.original.totalAmountMinor} className="text-xs" />
        ),
        meta: { align: "right" },
      },
      {
        id: "appliedAmountMinor",
        header: "Applied",
        accessorFn: (row) => row.appliedAmountMinor,
        cell: ({ row }) => (
          <AmountDisplay
            value={row.original.appliedAmountMinor}
            className="text-xs text-muted-foreground"
          />
        ),
        meta: { align: "right" },
      },
      {
        id: "openAmountMinor",
        header: "Open",
        accessorFn: (row) => row.openAmountMinor,
        cell: ({ row }) => (
          <AmountDisplay
            value={row.original.openAmountMinor}
            className="text-xs font-semibold"
          />
        ),
        meta: { align: "right" },
      },
    ],
    [],
  );

  const table = useReactTable({
    data: items,
    columns,
    state: { sorting, rowSelection },
    onSortingChange: setSorting,
    onRowSelectionChange,
    enableRowSelection: true,
    getRowId: (row) => row.invoiceId,
    getCoreRowModel: getCoreRowModel(),
    getSortedRowModel: getSortedRowModel(),
  });

  const totals = useMemo(
    () =>
      items.reduce(
        (acc, item) => ({
          total: acc.total + item.totalAmountMinor,
          applied: acc.applied + item.appliedAmountMinor,
          open: acc.open + item.openAmountMinor,
        }),
        { total: 0, applied: 0, open: 0 },
      ),
    [items],
  );

  return (
    <div className="overflow-hidden rounded-md border">
      <div className="overflow-x-auto">
        <Table>
          <TableHeader className="bg-muted/50">
            {table.getHeaderGroups().map((headerGroup) => (
              <TableRow key={headerGroup.id} className="hover:bg-transparent">
                {headerGroup.headers.map((header) => {
                  const align = (
                    header.column.columnDef.meta as { align?: string } | undefined
                  )?.align;
                  const sorted = header.column.getIsSorted();
                  return (
                    <TableHead
                      key={header.id}
                      className={cn("h-9 text-xs", align === "right" && "text-right")}
                    >
                      {header.column.getCanSort() ? (
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
                      ) : (
                        flexRender(header.column.columnDef.header, header.getContext())
                      )}
                    </TableHead>
                  );
                })}
              </TableRow>
            ))}
          </TableHeader>
          <TableBody>
            {table.getRowModel().rows.map((row) => (
              <TableRow
                key={row.id}
                data-state={row.getIsSelected() ? "selected" : undefined}
                className="cursor-pointer transition-colors"
                onClick={() => row.toggleSelected()}
              >
                {row.getVisibleCells().map((cell) => {
                  const align = (
                    cell.column.columnDef.meta as { align?: string } | undefined
                  )?.align;
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
          <TableFooter className="bg-muted/40">
            <TableRow className="hover:bg-transparent">
              <TableCell colSpan={8} className="py-2 text-right text-xs font-medium">
                Totals · {items.length} {items.length === 1 ? "invoice" : "invoices"}
              </TableCell>
              <TableCell className="py-2 text-right">
                <AmountDisplay value={totals.total} className="text-xs font-semibold" />
              </TableCell>
              <TableCell className="py-2 text-right">
                <AmountDisplay value={totals.applied} className="text-xs font-semibold" />
              </TableCell>
              <TableCell className="py-2 text-right">
                <AmountDisplay value={totals.open} className="text-xs font-bold" />
              </TableCell>
            </TableRow>
          </TableFooter>
        </Table>
      </div>
    </div>
  );
}
