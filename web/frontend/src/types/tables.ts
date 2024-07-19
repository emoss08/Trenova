/**
 * COPYRIGHT(c) 2024 Trenova
 *
 * This file is part of Trenova.
 *
 * The Trenova software is licensed under the Business Source License 1.1. You are granted the right
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

import type { Column, ColumnDef, Row, Table } from "@tanstack/react-table";
import React from "react";
import type { QueryKeys, ValuesOf } from "./index";
import { type API_ENDPOINTS } from "./server";

export interface DataTableFacetedFilterProps<TData, TValue> {
  /**
   * The column to filter by.
   * @type Column<TData, TValue>
   * @example column={columns.find((column) => column.id === "name")}
   */
  column?: Column<TData, TValue>;

  /**
   * The title of the filter.
   * @type string
   * @example title="Name"
   * @default ""
   */
  title?: string;

  /**
   * The options to filter by.
   * @type TableOptionProps[]
   * @example options={[{ label: "All", value: "" }, { label: "Active", value: true }, { label: "Inactive", value: false }]}
   * @default []
   */
  options: {
    label: string;
    value: string | boolean;
    icon?: React.ComponentType<{ className?: string }>;
  }[];
}

export type DataTableProps<K> = {
  /**
   * The columns to display in the table.
   * @type ColumnDef<K>[]
   * @example columns={[{ id: "name", Header: "Name", accessor: "name" }, { id: "status", Header: "Status", accessor: "status" }]}
   * @default []
   */
  columns: ColumnDef<K>[];

  /**
   * The name of the table.
   * @type string
   * @example name="commodities"
   */
  name: string;

  /**
   * The endpoint to fetch data from.
   * @type API_ENDPOINTS
   * @example link="/commodities/"
   */
  link: API_ENDPOINTS;

  /**
   * The key to use for the query.
   * @type QueryKeys | string
   * @example queryKey="commodities"
   */
  queryKey: ValuesOf<QueryKeys>;

  tableFacetedFilters?: FilterConfig<K>[];
  filterColumn: string;
  TableSheet?: React.ComponentType<TableSheetProps>;
  TableEditSheet?: React.ComponentType<TableSheetProps>;
  exportModelName: string;
  getRowCanExpand?: (row: Row<K>) => boolean;
  renderSubComponent?: (props: { row: Row<K> }) => React.ReactElement;
  extraSearchParams?: Record<string, any>;
  addPermissionName: string;
  includeHeader?: boolean;
  includeTopBar?: boolean;

  /**
   * The content to render in the floating bar on row selection, at the bottom of the table. When null, the floating bar is not rendered.
   * The datTable instance is passed as a prop to the floating bar content.
   * @default null
   * @type React.ReactNode | null
   * @example floatingBarContent={TasksTableFloatingBarContent(dataTable)}
   */
  floatingBarContent?: React.ReactNode | null;
};

export type TableSheetProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  currentRecord?: Record<string, any>;
};

export type TableOptionProps = {
  label: string;
  value: string | boolean;
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
