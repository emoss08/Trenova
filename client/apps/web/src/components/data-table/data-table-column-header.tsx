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
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import type { Column, SortDirection } from "@trenova/shared/types/data-table";
import {
  ArrowDownIcon,
  ArrowUpIcon,
  ChevronSelectorHorizontalIcon,
  EyeOffIcon,
  FlipBackwardIcon,
  Pin01Icon,
  PinOffIcon,
  SwitchVertical01Icon,
} from "@trenova/shared/components/icons";
import { useTableAtom } from "@trenova/shared/hooks/use-table-atom";
import { useOptionalDataTable } from "@/contexts/data-table-context";
import { isColumnResizable } from "@/lib/data-table";

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
  const t = useT();
  const pinned = useTableAtom(column.table.atoms.columnPinning, () => column.getIsPinned());
  const fitColumns = useOptionalDataTable()?.fitColumns;

  const meta = column.columnDef.meta;
  const apiField = meta?.apiField || column.id;
  const isSortable = meta?.sortable !== false;

  const currentSortEntry = currentSort?.find((s) => s.field === apiField);
  const sortDirection = currentSortEntry?.direction;
  const sortIndex = currentSort?.findIndex((s) => s.field === apiField);
  const showSortIndex =
    currentSort && currentSort.length > 1 && sortIndex !== undefined && sortIndex >= 0;

  if (!isSortable) {
    // design-tokens-ignore: column heads are uppercase across every data table by product decision
    return <div className={cn("flex items-center uppercase", className)}>{title}</div>;
  }

  const handleSort = (direction: SortDirection | null) => {
    onSort?.(apiField, direction);
  };

  return (
    <div className={cn("flex items-center gap-2", className)}>
      <DropdownMenu>
        {/* -ml-2.5 cancels the button's own px-2.5 so a sortable title starts
            on the same vertical as an unsortable one and as the cells below it,
            and the type matches TableHead rather than the button's default. */}
        <DropdownMenuTrigger
          render={
            <Button
              variant="ghost"
              size="sm"
              className="data-open:bg-accent text-foreground-subtle hover:text-foreground -ml-2.5 text-xs font-medium"
            >
              {/* design-tokens-ignore: column heads are uppercase across every data table by product decision */}
              <span className="uppercase">{title}</span>
              {showSortIndex && (
                <span className="bg-primary text-primary-foreground ml-1 flex size-4 items-center justify-center rounded-full text-2xs font-medium">
                  {sortIndex + 1}
                </span>
              )}
              {sortDirection === "desc" ? (
                <ArrowDownIcon className="size-3.5" />
              ) : sortDirection === "asc" ? (
                <ArrowUpIcon className="size-3.5" />
              ) : (
                <SwitchVertical01Icon className="size-3.5" />
              )}
            </Button>
          }
        />
        <DropdownMenuContent align="start">
          <DropdownMenuGroup>
            <DropdownMenuItem
              startContent={<ArrowUpIcon className="text-muted-foreground/70 size-3.5" />}
              title={t("Asc")}
              label={t("Asc")}
              onClick={() => handleSort("asc")}
            />
            <DropdownMenuItem
              startContent={<ArrowDownIcon className="text-muted-foreground/70 size-3.5" />}
              title={t("Desc")}
              label={t("Desc")}
              onClick={() => handleSort("desc")}
            />
            {sortDirection && (
              <>
                <DropdownMenuSeparator />
                <DropdownMenuItem
                  startContent={
                    <SwitchVertical01Icon className="text-muted-foreground/70 size-3.5" />
                  }
                  title={t("Clear sort")}
                  label={t("Clear sort")}
                  onClick={() => handleSort(null)}
                />
              </>
            )}
            {column.getCanPin() && (
              <>
                <DropdownMenuSeparator />
                {pinned !== "start" && (
                  <DropdownMenuItem
                    onClick={() => column.pin("start")}
                    startContent={
                      <Pin01Icon className="text-muted-foreground/70 size-3.5 -rotate-45" />
                    }
                    title={t("Pin left")}
                    label={t("Pin left")}
                  />
                )}
                {pinned !== "end" && (
                  <DropdownMenuItem
                    onClick={() => column.pin("end")}
                    startContent={
                      <Pin01Icon className="text-muted-foreground/70 size-3.5 rotate-45" />
                    }
                    title={t("Pin right")}
                    label={t("Pin right")}
                  />
                )}
                {pinned && (
                  <DropdownMenuItem
                    onClick={() => column.pin(false)}
                    startContent={<PinOffIcon className="text-muted-foreground/70 size-3.5" />}
                    title={t("Unpin")}
                    label={t("Unpin")}
                  />
                )}
              </>
            )}
            {fitColumns && isColumnResizable(column) && (
              <>
                <DropdownMenuSeparator />
                <DropdownMenuItem
                  onClick={() => fitColumns([column.id])}
                  startContent={
                    <ChevronSelectorHorizontalIcon className="text-muted-foreground/70 size-3.5" />
                  }
                  title={t("Fit to content")}
                  label={t("Fit to content")}
                />
                <DropdownMenuItem
                  onClick={() => column.resetSize()}
                  startContent={<FlipBackwardIcon className="text-muted-foreground/70 size-3.5" />}
                  title={t("Reset width")}
                  label={t("Reset width")}
                />
              </>
            )}
            {column.getCanHide() && (
              <>
                <DropdownMenuSeparator />
                <DropdownMenuItem
                  onClick={() => column.toggleVisibility(false)}
                  startContent={<EyeOffIcon className="text-muted-foreground/70 size-3.5" />}
                  title={t("Hide")}
                  label={t("Hide")}
                />
              </>
            )}
          </DropdownMenuGroup>
        </DropdownMenuContent>
      </DropdownMenu>
    </div>
  );
}
