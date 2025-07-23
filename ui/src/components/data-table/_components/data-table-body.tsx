/* eslint-disable @typescript-eslint/no-unused-vars */
"use no memo";
import { Icon } from "@/components/ui/icons";
import { Switch } from "@/components/ui/switch";
import { TableBody, TableCell, TableRow } from "@/components/ui/table";
import { cn } from "@/lib/utils";
import { DataTableBodyProps } from "@/types/data-table";
import {
  SortableContext,
  horizontalListSortingStrategy,
} from "@dnd-kit/sortable";
import { faPlay } from "@fortawesome/pro-solid-svg-icons";
import { flexRender, type Row, type Table } from "@tanstack/react-table";
import { DragAlongCell } from "./data-table-draggable";
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
  enableDragging = false,
  contextMenuActions,
}: {
  row: Row<TData>;
  selected?: boolean;
  columnVisibility: Record<string, boolean>;
  table: Table<TData>;
  enableDragging?: boolean;
  contextMenuActions?: ContextMenuAction<TData>[];
}) {
  const columnOrder = table.getState().columnOrder;
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
        "-outline-offset-2 rounded-md outline-muted-foreground transition-colors focus-visible:bg-muted/50 focus-visible:outline data-[state=selected]:outline",
        table.options.meta?.getRowClassName?.(row.original),
      )}
    >
      {enableDragging
        ? row.getVisibleCells().map((cell) => (
            <SortableContext
              key={cell.id}
              items={columnOrder}
              strategy={horizontalListSortingStrategy}
            >
              <DragAlongCell key={cell.id} cell={cell} />
            </SortableContext>
          ))
        : row.getVisibleCells().map((cell) => (
            <TableCell
              className="font-table"
              key={cell.id}
              role="cell"
              aria-label={`${cell.column.id} cell`}
              style={{
                minWidth: cell.column.getSize(),
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
  enableDragging = false,
  contextMenuActions,
}: DataTableBodyProps<TData> & {
  enableDragging?: boolean;
  contextMenuActions?: ContextMenuAction<TData>[];
}) {
  return (
    <TableBody id="content" tabIndex={-1}>
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
              enableDragging={enableDragging}
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
