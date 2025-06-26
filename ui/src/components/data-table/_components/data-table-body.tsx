/* eslint-disable @typescript-eslint/no-unused-vars */
"use no memo";
import { Icon } from "@/components/ui/icons";
import { Switch } from "@/components/ui/switch";
import { TableBody, TableCell, TableRow } from "@/components/ui/table";
import { cn } from "@/lib/utils";
import { DataTableBodyProps } from "@/types/data-table";
import { faPlay } from "@fortawesome/pro-solid-svg-icons";
import { flexRender, type Row, type Table } from "@tanstack/react-table";
import React from "react";

function LiveModeTableRow({
  columns,
  liveMode,
}: {
  columns: any[];
  liveMode: {
    enabled: boolean;
    connected: boolean;
    showToggle?: boolean;
    onToggle?: (enabled: boolean) => void;
    autoRefresh?: boolean;
    onAutoRefreshToggle?: (autoRefresh: boolean) => void;
  };
}) {
  return (
    <TableRow className="bg-blue-500/10 hover:!bg-blue-500/20 [&:hover_td]:md:!bg-blue-500/10 [&_td]:md:border-blue-500/10">
      <TableCell colSpan={columns.length} className="p-3 select-none">
        <div className="flex justify-between items-center">
          <div className="flex items-center gap-3 text-blue-600">
            <div className="flex items-center gap-1">
              <Icon icon={faPlay} className="size-3 text-blue-600" />
              <span className="text-sm font-medium">Live Mode</span>
            </div>
          </div>

          {liveMode.showToggle && (
            <div className="flex items-center gap-2">
              {liveMode.onAutoRefreshToggle && (
                <div className="flex items-center gap-2">
                  <span className="text-xs">Auto-refresh</span>
                  <Switch
                    checked={liveMode.autoRefresh || false}
                    onCheckedChange={liveMode.onAutoRefreshToggle}
                    size="sm"
                  />
                </div>
              )}
            </div>
          )}
        </div>
      </TableCell>
    </TableRow>
  );
}

function DataTableRow<TData>({
  row,
  selected,
  table,
  // We don't actually use columnVisibility in the component,
  // but we need it for the memo comparison
  // @ts-expect-error - This is a temporary solution to avoid the memo comparison
  columnVisibility,
}: {
  row: Row<TData>;
  selected?: boolean;
  columnVisibility: Record<string, boolean>;
  table: Table<TData>;
}) {
  return (
    <TableRow
      id={row.id}
      tabIndex={0}
      data-state={selected && "selected"}
      onClick={() => row.toggleSelected()}
      onKeyDown={(event) => {
        if (event.key === "Enter") {
          event.preventDefault();
          row.toggleSelected();
        }
      }}
      className={cn(
        "-outline-offset-2 rounded-md outline-muted-foreground transition-colors focus-visible:bg-muted/50 focus-visible:outline data-[state=selected]:outline",
        table.options.meta?.getRowClassName?.(row),
      )}
    >
      {row.getVisibleCells().map((cell) => {
        return (
          <TableCell
            key={cell.id}
            role="cell"
            aria-label={`${cell.column.id} cell`}
          >
            {flexRender(cell.column.columnDef.cell, cell.getContext())}
          </TableCell>
        );
      })}
    </TableRow>
  );
}

const MemoizedRow = React.memo(DataTableRow, (prev, next) => {
  // Check ID and selection state first (fast checks)
  if (prev.row.id !== next.row.id || prev.selected !== next.selected) {
    return false;
  }

  // Check for column visibility changes
  const prevKeys = Object.keys(prev.columnVisibility);
  const nextKeys = Object.keys(next.columnVisibility);

  if (prevKeys.length !== nextKeys.length) {
    return false;
  }

  for (const key of prevKeys) {
    if (prev.columnVisibility[key] !== next.columnVisibility[key]) {
      return false;
    }
  }

  const prevOriginal = prev.row.original as Record<string, any>;
  const nextOriginal = next.row.original as Record<string, any>;

  // Compare updatedAt timestamps for data changes
  return prevOriginal.updatedAt === nextOriginal.updatedAt;
}) as typeof DataTableRow;

export function DataTableBody<TData extends Record<string, any>>({
  table,
  columns,
  liveMode,
}: DataTableBodyProps<TData>) {
  return (
    <TableBody id="content" tabIndex={-1}>
      {liveMode?.enabled && (
        <LiveModeTableRow columns={columns} liveMode={liveMode} />
      )}
      {table.getRowModel().rows?.length ? (
        table.getRowModel().rows.map((row) => {
          return (
            <MemoizedRow
              key={row.id}
              row={row}
              selected={row.getIsSelected()}
              columnVisibility={table.getState().columnVisibility}
              table={table}
            />
          );
        })
      ) : (
        <TableRow>
          <TableCell
            colSpan={columns.length}
            className="h-24 text-center border-b rounded-b-md"
          >
            No results.
          </TableCell>
        </TableRow>
      )}
    </TableBody>
  );
}
