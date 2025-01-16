import { Button } from "@/components/ui/button";
import { Icon } from "@/components/ui/icons";

import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { useTableStore as store } from "@/stores/table-store";
import { DataTableAdvancedFilterField } from "@/types/data-table";
import {
  faDownload,
  faHistory,
  faSave,
  faUpload,
} from "@fortawesome/pro-regular-svg-icons";
import { DotsVerticalIcon } from "@radix-ui/react-icons";
import { ColumnFiltersState, SortingState, Table } from "@tanstack/react-table";
import { useQueryState } from "nuqs";
import { useState } from "react";
import { DataTableFilterDialog } from "./data-table-filter-dialog";

type DataTableOptionsProps<TData> = {
  filters: ColumnFiltersState;
  sorting: SortingState;
  filterFields: DataTableAdvancedFilterField<TData>[];
  table: Table<TData>;
};

export function DataTableOptions<TData>({
  filters,
  sorting,
  filterFields,
  table,
}: DataTableOptionsProps<TData>) {
  const [showOptions, setShowOptions] = useState<boolean>(false);
  // eslint-disable-next-line @typescript-eslint/no-unused-vars
  const [filterState, _] = useQueryState("filters");
  const [showFilterDialog, setShowFilterDialog] = store.use("showFilterDialog");

  return (
    <>
      <DropdownMenu
        open={showOptions}
        onOpenChange={(open) => setShowOptions(open)}
      >
        <DropdownMenuTrigger asChild>
          <Button
            variant="outline"
            onClick={() => setShowOptions(!showOptions)}
          >
            <DotsVerticalIcon />
            Options
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent>
          <DropdownMenuGroup>
            <DropdownMenuLabel>Options</DropdownMenuLabel>
            <DropdownMenuSeparator />
            <DropdownMenuItem
              disabled
              startContent={<Icon icon={faUpload} />}
              title="Import"
              description="Import data from a file"
            />
            <DropdownMenuItem
              startContent={<Icon icon={faDownload} />}
              title="Export"
              description="Export the data to a file or email"
              // onClick={() => store.set("exportModalOpen", true)}
            />
            <DropdownMenuSeparator />
            <DropdownMenuItem
              startContent={<Icon icon={faSave} />}
              title="Save Filter"
              description="Save the current filter"
              disabled={!filterState}
              onClick={() => setShowFilterDialog(true)}
            />
            <DropdownMenuItem
              disabled
              startContent={<Icon icon={faHistory} />}
              title="View Audit Log"
              description="View the audit log for the data"
            />
          </DropdownMenuGroup>
        </DropdownMenuContent>
      </DropdownMenu>
      <DataTableFilterDialog<TData>
        open={showFilterDialog}
        onClose={() => setShowFilterDialog(false)}
        sorting={sorting}
        columnFilters={filters}
        table={table}
        filterFields={filterFields}
      />
    </>
  );
}
