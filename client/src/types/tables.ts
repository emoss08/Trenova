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

import { Column, ColumnDef, Table } from "@tanstack/react-table";
import { QueryKeys } from "./index";

export interface DataTableFacetedFilterProps<TData, TValue> {
  column?: Column<TData, TValue>;
  title?: string;
  options: {
    label: string;
    value: string;
    icon?: React.ComponentType<{ className?: string }>;
  }[];
}

export type DataTableProps<K> = {
  columns: ColumnDef<K>[];
  name: string;
  link: string;
  queryKey: QueryKeys;
  tableFacetedFilters?: FilterConfig<K>[];
  filterColumn: string;
  TableSheet?: React.ComponentType<TableSheetProps>;
};

export type TableSheetProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
};

export type TableOptionProps = {
  label: string;
  value: string;
  icon?: React.ComponentType<{ className?: string }>;
};

export type FilterConfig<TData> = {
  columnName: keyof TData;
  title: string;
  options: TableOptionProps[];
};

export type DataTableFacetedFilterListProps<TData> = {
  table: Table<TData>;
  filters: FilterConfig<TData>[];
};
