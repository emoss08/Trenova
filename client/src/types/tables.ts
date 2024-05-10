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
