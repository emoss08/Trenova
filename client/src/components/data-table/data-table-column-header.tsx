import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { cn } from "@/lib/utils";
import type { SortDirection } from "@/types/data-table";
import type { Column } from "@tanstack/react-table";
import {
  ArrowDownIcon,
  ArrowUpDownIcon,
  ArrowUpIcon,
  EyeOffIcon,
} from "lucide-react";

type DataTableColumnHeaderProps<TData, TValue> = {
  column: Column<TData, TValue>;
  title: string;
  currentSort?: { field: string; direction: SortDirection }[];
  onSort?: (field: string, direction: SortDirection | null) => void;
  className?: string;
};

export function DataTableColumnHeader<TData, TValue>({
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
    currentSort &&
    currentSort.length > 1 &&
    sortIndex !== undefined &&
    sortIndex >= 0;

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
            <Button
              variant="ghost"
              size="sm"
              className="-ml-3 h-8 data-open:bg-accent"
            >
              <span className="uppercase">{title}</span>
              {showSortIndex && (
                <span className="ml-1 flex size-4 items-center justify-center rounded-full bg-primary text-[10px] font-medium text-primary-foreground">
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
              startContent={
                <ArrowUpIcon className="size-3.5 text-muted-foreground/70" />
              }
              title="Asc"
              label="Asc"
              onClick={() => handleSort("asc")}
            />
            <DropdownMenuItem
              startContent={
                <ArrowDownIcon className="size-3.5 text-muted-foreground/70" />
              }
              title="Desc"
              label="Desc"
              onClick={() => handleSort("desc")}
            />
            {sortDirection && (
              <>
                <DropdownMenuSeparator />
                <DropdownMenuItem
                  startContent={
                    <ArrowUpDownIcon className="size-3.5 text-muted-foreground/70" />
                  }
                  title="Clear sort"
                  label="Clear sort"
                  onClick={() => handleSort(null)}
                />
              </>
            )}
            {column.getCanHide() && (
              <>
                <DropdownMenuSeparator />
                <DropdownMenuItem
                  onClick={() => column.toggleVisibility(false)}
                  startContent={
                    <EyeOffIcon className="size-3.5 text-muted-foreground/70" />
                  }
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
