/*
 * COPYRIGHT(c) 2023 MONTA
 *
 * This file is part of Monta.
 *
 * The Monta software is licensed under the Business Source License 1.1. You are granted the right
 * to copy, modify, and redistribute the software, but only for non-production use or with a total
 * of less than three server instances. Starting from the Change Date (November 16, 2026), the
 * software will be made available under version 2 or later of the GNU General Public License.
 * If you use the software in violation of this license, your rights under the license will be
 * terminated automatically. The software is provided "as is," and the Licensor disclaims all
 * warranties and conditions. If you use this license's text or the "Business Source License" name
 * and trademark, you must comply with the Licensor's covenants, which include specifying the
 * Change License as the GPL Version 2.0 or a compatible license, specifying an Additional Use
 * Grant, and not modifying the license in any other way.
 */

import { Button } from "@/components/ui/button";
import { TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { StoreType } from "@/lib/useGlobalStore";
import { TableStoreProps } from "@/stores/TableStore";
import { DataTableFacetedFilterListProps, FilterConfig } from "@/types/tables";
import { Table as TableType, flexRender } from "@tanstack/react-table";
import { Plus, X } from "lucide-react";
import React, { ChangeEvent } from "react";
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

const ButtonConfig: React.FC<{
  selectedRowCount: number;
  name: string;
  store: StoreType<TableStoreProps>;
}> = React.memo(({ selectedRowCount, name, store }) => {
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
    >
      <Plus className="mr-2 h-4 w-4" /> {buttonLabel}
    </Button>
  );
});

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

  return (
    <div className="flex flex-col sm:flex-row justify-between">
      <div className="flex-1 flex flex-col sm:flex-row space-y-2 sm:space-y-0 sm:space-x-2 mr-2">
        <Input
          placeholder="Filter..."
          value={
            (table.getColumn(filterColumn)?.getFilterValue() as string) ?? ""
          }
          onChange={handleFilterChange}
          className="w-full lg:w-[250px] h-8"
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
            <X className="ml-2 h-4 w-4" />
          </Button>
        )}
      </div>
      <div className="flex flex-col sm:flex-row space-y-2 sm:space-y-0 sm:space-x-2 mt-2 sm:mt-0">
        <DataTableViewOptions table={table} />
        <DataTableImportExportOption />
        {userHasPermission(addPermissionName) && (
          <ButtonConfig
            selectedRowCount={selectedRowCount}
            name={name}
            store={store}
          />
        )}
      </div>
    </div>
  );
}
