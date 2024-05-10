import { Button } from "@/components/ui/button";
import { TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { type StoreType } from "@/lib/useGlobalStore";
import { TableStoreProps } from "@/stores/TableStore";
import type {
  DataTableFacetedFilterListProps,
  FilterConfig,
} from "@/types/tables";
import { faX } from "@fortawesome/pro-solid-svg-icons";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { flexRender, type Table as TableType } from "@tanstack/react-table";
import React, { type ChangeEvent } from "react";
import { Input } from "../fields/input";
import { DataTableImportExportOption } from "./data-table-export-modal";
import { DataTableFacetedFilter } from "./data-table-faceted-filter";
import { DataTableViewOptions } from "./data-table-view-options";

export function DataTableHeader<K extends Record<string, any>>({
  table,
}: {
  table: TableType<K>;
}) {
  return (
    <TableHeader>
      {table.getHeaderGroups().map((headerGroup) => (
        <TableRow key={headerGroup.id}>
          {headerGroup.headers.map((header) => (
            <TableHead key={header.id}>
              {header.isPlaceholder
                ? null
                : flexRender(
                    header.column.columnDef.header,
                    header.getContext(),
                  )}
            </TableHead>
          ))}
        </TableRow>
      ))}
    </TableHeader>
  );
}

export function DataTableFacetedFilterList<TData>({
  table,
  filters,
}: DataTableFacetedFilterListProps<TData>) {
  return (
    <>
      {filters.map((filter) => {
        const column = table.getColumn(filter.columnName as string);
        return (
          column && (
            <DataTableFacetedFilter
              key={filter.columnName as string}
              column={column}
              title={filter.title}
              options={filter.options}
            />
          )
        );
      })}
    </>
  );
}

function AddNewButton({
  selectedRowCount,
  name,
  store,
  isDisabled,
}: {
  selectedRowCount: number;
  name: string;
  store: StoreType<TableStoreProps>;
  isDisabled?: boolean;
}) {
  const buttonLabel =
    selectedRowCount > 0
      ? `Inactivate ${selectedRowCount} records`
      : `Add New ${name}`;

  const buttonVariant = selectedRowCount > 0 ? "destructive" : "default";

  return (
    <Button
      variant={buttonVariant}
      onClick={() => store.set("sheetOpen", true)}
      className="h-8"
      disabled={isDisabled}
    >
      {buttonLabel}
    </Button>
  );
}

export function DataTableTopBar<K>({
  table,
  name,
  selectedRowCount,
  filterColumn,
  tableFacetedFilters,
  userHasPermission,
  addPermissionName,
  store,
}: {
  table: TableType<K>;
  name: string;
  selectedRowCount: number;
  filterColumn: string;
  tableFacetedFilters?: FilterConfig<K>[];
  userHasPermission: (permission: string) => boolean;
  addPermissionName: string;
  store: StoreType<TableStoreProps>;
}) {
  const isFiltered = table.getState().columnFilters.length > 0;

  // Memoize the onChange handler for the filter input
  const handleFilterChange = React.useCallback(
    (event: ChangeEvent<HTMLInputElement>) => {
      const column = table.getColumn(filterColumn);
      if (column) {
        column.setFilterValue(event.target.value);
      }
    },
    [table, filterColumn],
  );

  // Memoize the onClick handler for the reset filters button
  const handleResetFilters = React.useCallback(
    () => table.resetColumnFilters(),
    [table],
  );

  return table.getPageCount() > 0 ? (
    <div className="flex flex-col justify-between sm:flex-row">
      <div className="mr-2 flex flex-1 flex-col space-y-2 sm:flex-row sm:space-x-2 sm:space-y-0">
        <Input
          placeholder="Filter..."
          value={
            (table.getColumn(filterColumn)?.getFilterValue() as string) ?? ""
          }
          onChange={handleFilterChange}
          className="h-8 w-full lg:w-[250px]"
        />
        {tableFacetedFilters && (
          <DataTableFacetedFilterList
            table={table}
            filters={tableFacetedFilters}
          />
        )}
        {isFiltered && (
          <Button
            variant="ghost"
            onClick={handleResetFilters}
            className="h-8 px-2 lg:px-3"
          >
            Reset
            <FontAwesomeIcon icon={faX} className="ml-2 size-4" />
          </Button>
        )}
      </div>
      <div className="mt-2 flex flex-col space-y-2 sm:mt-0 sm:flex-row sm:space-x-2 sm:space-y-0">
        <DataTableViewOptions table={table} />
        <DataTableImportExportOption />
        <AddNewButton
          selectedRowCount={selectedRowCount}
          name={name}
          store={store}
          isDisabled={!userHasPermission(addPermissionName)}
        />
      </div>
    </div>
  ) : (
    <div className="flex flex-col justify-end sm:flex-row">
      <div className="mt-2 flex flex-col space-y-2 sm:mt-0 sm:flex-row sm:space-x-2 sm:space-y-0">
        <AddNewButton
          selectedRowCount={selectedRowCount}
          name={name}
          store={store}
          isDisabled={!userHasPermission(addPermissionName)}
        />
      </div>
    </div>
  );
}
