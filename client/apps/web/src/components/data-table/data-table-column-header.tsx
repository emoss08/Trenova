import type { RowData } from "@tanstack/react-table";
import { Button } from "@trenova/shared/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@trenova/shared/components/ui/dropdown-menu";
import { cn } from "@trenova/shared/lib/utils";
import type { SortDirection, Column } from "@trenova/shared/types/data-table";
import {
  ArrowDownIcon,
  ArrowUpDownIcon,
  ArrowUpIcon,
  EyeOffIcon,
  PinIcon,
  PinOffIcon,
} from "lucide-react";

type DataTableColumnHeaderProps<TData extends RowData, TValue> = {
  column: Column<TData, TValue>;
  title: string;
  currentSort?: { field: string; direction: SortDirection }[];
  onSort?: (field: string, direction: SortDirection | null) => void;
  className?: string;
};

export function DataTableColumnHeader<TData extends RowData, TValue>({
  column,
  title,
  currentSort,
  onSort,
  className,
}: DataTableColumnHeaderProps<TData, TValue>) {
  const meta = column.columnDef.meta;
  const apiField = meta?.apiField || column.id;
  const isSortable = meta?.sortable !== false;

  const currentSortEntry = currentSort?.find((s) => s.field === apiField);
  const sortDirection = currentSortEntry?.direction;
  const sortIndex = currentSort?.findIndex((s) => s.field === apiField);
  const showSortIndex =
    currentSort && currentSort.length > 1 && sortIndex !== undefined && sortIndex >= 0;

  if (!isSortable) {
    return <div className={cn("flex items-center", className)}>{title}</div>;
  }

  const handleSort = (direction: SortDirection | null) => {
    onSort?.(apiField, direction);
  };

  return (
    <div className={cn("flex items-center gap-2", className)}>
      <DropdownMenu>
        <DropdownMenuTrigger
          render={
            <Button variant="ghost" size="sm" className="data-open:bg-accent -ml-3">
              <span className="uppercase">{title}</span>
              {showSortIndex && (
                <span className="bg-primary text-primary-foreground ml-1 flex size-4 items-center justify-center rounded-full text-[10px] font-medium">
                  {sortIndex + 1}
                </span>
              )}
              {sortDirection === "desc" ? (
                <ArrowDownIcon className="size-3.5" />
              ) : sortDirection === "asc" ? (
                <ArrowUpIcon className="size-3.5" />
              ) : (
                <ArrowUpDownIcon className="size-3.5" />
              )}
            </Button>
          }
        />
        <DropdownMenuContent align="start">
          <DropdownMenuGroup>
            <DropdownMenuItem
              startContent={<ArrowUpIcon className="text-muted-foreground/70 size-3.5" />}
              title="Asc"
              label="Asc"
              onClick={() => handleSort("asc")}
            />
            <DropdownMenuItem
              startContent={<ArrowDownIcon className="text-muted-foreground/70 size-3.5" />}
              title="Desc"
              label="Desc"
              onClick={() => handleSort("desc")}
            />
            {sortDirection && (
              <>
                <DropdownMenuSeparator />
                <DropdownMenuItem
                  startContent={<ArrowUpDownIcon className="text-muted-foreground/70 size-3.5" />}
                  title="Clear sort"
                  label="Clear sort"
                  onClick={() => handleSort(null)}
                />
              </>
            )}
            {column.getCanPin() && (
              <>
                <DropdownMenuSeparator />
                {column.getIsPinned() !== "start" && (
                  <DropdownMenuItem
                    onClick={() => column.pin("start")}
                    startContent={
                      <PinIcon className="text-muted-foreground/70 size-3.5 -rotate-45" />
                    }
                    title="Pin left"
                    label="Pin left"
                  />
                )}
                {column.getIsPinned() !== "end" && (
                  <DropdownMenuItem
                    onClick={() => column.pin("end")}
                    startContent={
                      <PinIcon className="text-muted-foreground/70 size-3.5 rotate-45" />
                    }
                    title="Pin right"
                    label="Pin right"
                  />
                )}
                {column.getIsPinned() && (
                  <DropdownMenuItem
                    onClick={() => column.pin(false)}
                    startContent={<PinOffIcon className="text-muted-foreground/70 size-3.5" />}
                    title="Unpin"
                    label="Unpin"
                  />
                )}
              </>
            )}
            {column.getCanHide() && (
              <>
                <DropdownMenuSeparator />
                <DropdownMenuItem
                  onClick={() => column.toggleVisibility(false)}
                  startContent={<EyeOffIcon className="text-muted-foreground/70 size-3.5" />}
                  title="Hide"
                  label="Hide"
                />
              </>
            )}
          </DropdownMenuGroup>
        </DropdownMenuContent>
      </DropdownMenu>
    </div>
  );
}
