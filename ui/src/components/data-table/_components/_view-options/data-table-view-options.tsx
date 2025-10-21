import { Button } from "@/components/ui/button";
import { Icon } from "@/components/ui/icons";

import { Input } from "@/components/ui/input";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import type { Resource } from "@/types/audit-entry";
import { DataTableCreateButtonProps } from "@/types/data-table";
import { faPlus, faSearch } from "@fortawesome/pro-regular-svg-icons";
import { faColumns } from "@fortawesome/pro-solid-svg-icons";
import { ChevronDownIcon } from "@radix-ui/react-icons";
import { memo, useCallback, useMemo, useState } from "react";
import { useDataTable } from "../../data-table-provider";
import { DataTableCreateContent } from "../_actions/data-table-create-content";
import { DataTableViewFooter } from "./data-table-view-footer";
import { ColumnSortContent } from "./sorted-columns";

export const DataTableCreateButton = memo(function DataTableCreateButton({
  name,
  isDisabled,
  onCreateClick,
  exportModelName,
  extraActions,
}: DataTableCreateButtonProps) {
  const [isPopoverOpen, setIsPopoverOpen] = useState(false);

  const handleCreateClick = useCallback(() => {
    setIsPopoverOpen(false);

    if (onCreateClick) {
      onCreateClick();
    }
  }, [onCreateClick, setIsPopoverOpen]);

  return (
    <>
      <Popover open={isPopoverOpen} onOpenChange={setIsPopoverOpen}>
        <PopoverTrigger asChild>
          <Button
            title={
              isDisabled
                ? `You do not have permissions to create ${name}s`
                : `Create a new ${name}`
            }
            variant="default"
            disabled={isDisabled}
          >
            <Icon icon={faPlus} className="size-4 text-background" />
            <span>New</span>
          </Button>
        </PopoverTrigger>
        <PopoverContent align="end" side="bottom" className="w-auto p-1">
          <DataTableCreateContent
            name={name}
            handleCreateClick={handleCreateClick}
            extraActions={extraActions}
            exportModelName={exportModelName}
          />
        </PopoverContent>
      </Popover>
    </>
  );
});

export function DataTableViewOptions({ resource }: { resource: Resource }) {
  const [open, setOpen] = useState(false);
  const [searchQuery, setSearchQuery] = useState("");
  const { table } = useDataTable();
  const columnOrder = table.getState().columnOrder;

  const sortedColumns = useMemo(() => {
    const allColumns = table.getAllColumns();
    const hideableColumns = allColumns.filter(
      (column) =>
        typeof column.accessorFn !== "undefined" && column.getCanHide(),
    );

    const sorted = hideableColumns.sort((a, b) => {
      const indexA = columnOrder.indexOf(a.id);
      const indexB = columnOrder.indexOf(b.id);
      return indexA - indexB;
    });

    return sorted;
  }, [columnOrder, table]);

  return (
    <>
      <Popover open={open} onOpenChange={setOpen}>
        <PopoverTrigger asChild>
          <Button
            variant="outline"
            className="h-7 border-dashed"
            aria-label="Toggle column visibility"
          >
            <Icon icon={faColumns} className="size-4" />
            <span className="hidden lg:inline">Configure Columns</span>
            <span className="lg:hidden">Columns</span>
            <span className="sr-only">Toggle column visibility options</span>
            <ChevronDownIcon />
          </Button>
        </PopoverTrigger>
        <PopoverContent align="end" side="bottom" className="w-auto p-2">
          <DataTableViewOptionsInner>
            <Input
              icon={
                <Icon
                  icon={faSearch}
                  className="size-3 text-muted-foreground"
                />
              }
              placeholder="Search columns..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              className="h-7 text-sm bg-background"
            />
            <ColumnSortContent
              sortedColumns={sortedColumns}
              searchQuery={searchQuery}
            />
            <DataTableViewFooter resource={resource} />
          </DataTableViewOptionsInner>
        </PopoverContent>
      </Popover>
    </>
  );
}

function DataTableViewOptionsInner({
  children,
}: {
  children: React.ReactNode;
}) {
  return <div className="flex flex-col">{children}</div>;
}
