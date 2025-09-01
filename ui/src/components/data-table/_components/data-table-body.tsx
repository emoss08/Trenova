/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

/* eslint-disable @typescript-eslint/no-unused-vars */
"use no memo";
import { Icon } from "@/components/ui/icons";
import { Switch } from "@/components/ui/switch";
import { TableBody, TableCell, TableRow } from "@/components/ui/table";
import { cn } from "@/lib/utils";
import { DataTableBodyProps } from "@/types/data-table";
import { faPlay } from "@fortawesome/pro-solid-svg-icons";
import { flexRender, type Row, type Table } from "@tanstack/react-table";
import {
  DataTableContextMenu,
  type ContextMenuAction,
} from "./data-table-context-menu";

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
    <TableRow
      className="bg-blue-500/10 hover:!bg-blue-500/20 [&:hover_td]:md:!bg-blue-500/10 [&_td]:md:border-blue-500/10"
      // Respect header presence using CSS var set on the scroll container
      style={{ top: "var(--header-h, 0px)" }}
    >
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
  contextMenuActions,
}: {
  row: Row<TData>;
  selected?: boolean;
  columnVisibility: Record<string, boolean>;
  table: Table<TData>;
  contextMenuActions?: ContextMenuAction<TData>[];
}) {
  const tableRow = (
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
        "-outline-offset-2 outline-muted-foreground transition-colors focus-visible:bg-muted/50 focus-visible:outline data-[state=selected]:outline",
        table.options.meta?.getRowClassName?.(row),
      )}
    >
      {row.getVisibleCells().map((cell) => (
        <TableCell
          className={cn("font-sans truncate border-b border-border", {
            // If the last row remove the border
            "border-b-0": row.index === table.getRowModel().rows.length - 1,
          })}
          key={cell.id}
          role="cell"
          aria-label={`${cell.column.id} cell`}
          style={{
            width: `var(--col-${cell.column.id.replace(".", "-")}-size)`,
            maxWidth: `var(--col-${cell.column.id.replace(".", "-")}-size)`,
          }}
        >
          {flexRender(cell.column.columnDef.cell, cell.getContext())}
        </TableCell>
      ))}
    </TableRow>
  );

  if (contextMenuActions && contextMenuActions.length > 0) {
    return (
      <DataTableContextMenu row={row} actions={contextMenuActions}>
        {tableRow}
      </DataTableContextMenu>
    );
  }

  return tableRow;
}

export function DataTableBody<TData extends Record<string, any>>({
  table,
  columns,
  liveMode,
  contextMenuActions,
}: DataTableBodyProps<TData> & {
  contextMenuActions?: ContextMenuAction<TData>[];
}) {
  return (
    <TableBody
      id="content"
      tabIndex={-1}
      // REMINDER: avoids scroll (skipping the table header) when using skip to content
      style={{
        scrollMarginTop: "calc(var(--top-bar-height) + 40px)",
      }}
    >
      {liveMode?.enabled && (
        <LiveModeTableRow columns={columns} liveMode={liveMode} />
      )}
      {table.getRowModel().rows?.length ? (
        table.getRowModel().rows.map((row) => {
          return (
            <DataTableRow
              key={row.id}
              row={row}
              selected={row.getIsSelected()}
              columnVisibility={table.getState().columnVisibility}
              table={table}
              contextMenuActions={contextMenuActions}
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
