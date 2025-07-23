/**
 * # Copyright 2023-2025 Eric Moss
 * # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md
 */

import { DataTableConfig } from "@/config/data-table";
import { API_ENDPOINTS } from "@/types/server";
import { IconDefinition } from "@fortawesome/pro-regular-svg-icons";
import type {
  ColumnDef,
  ColumnFilter,
  ColumnFiltersState,
  ColumnSort,
  OnChangeFn,
  PaginationState,
  Row,
  RowSelectionState,
  SortingState,
  Table,
  VisibilityState,
} from "@tanstack/react-table";
import React from "react";
import type { Resource } from "./audit-entry";
import type { LiveModeTableConfig } from "./live-mode";

export type Prettify<T> = {
  [K in keyof T]: T[K];
} & {};

export type StringKeyOf<TData> = Extract<keyof TData, string>;

export interface SearchParams {
  [key: string]: string | string[] | undefined;
}

export interface Option {
  label: string;
  value: string;
  icon?: React.ComponentType<{ className?: string }>;
  count?: number;
}

export interface ExtendedColumnSort<TData> extends Omit<ColumnSort, "id"> {
  id: StringKeyOf<TData>;
}

export type ExtendedSortingState<TData> = ExtendedColumnSort<TData>[];

export type ColumnType = DataTableConfig["columnTypes"][number];

export type FilterOperator = DataTableConfig["globalOperators"][number];

export type JoinOperator = DataTableConfig["joinOperators"][number]["value"];

export interface DataTableFilterField<TData> {
  id: StringKeyOf<TData>;
  label: string;
  placeholder?: string;
  options?: Option[];
}

export interface DataTableAdvancedFilterField<TData>
  extends DataTableFilterField<TData> {
  type: ColumnType;
}

export interface DataTableRowAction<TData> {
  row: Row<TData>;
  type: "update" | "delete";
}

export interface QueryBuilderOpts {
  where?: string;
  orderBy?: string;
  distinct?: boolean;
  nullish?: boolean;
}

type ExtraAction = {
  key: string;
  // * Label to be displayed
  label: string;
  icon?: IconDefinition;
  // * Content to be displayed after the label
  endContent?: React.ReactNode;
  // * Description to be displayed below the label
  description?: string;
  onClick: () => void;
};

export interface ContextMenuAction<TData> {
  id: string;
  label: string | ((row: Row<TData>) => string);
  shortcut?: string;
  variant?: "default" | "destructive";
  disabled?: boolean | ((row: Row<TData>) => boolean);
  hidden?: boolean | ((row: Row<TData>) => boolean);
  onClick?: (row: Row<TData>) => void; // Optional when subActions exist
  separator?: "before" | "after";
  subActions?: ContextMenuAction<TData>[];
}

type DataTableCreateButtonProps = {
  name: string;
  exportModelName: string;
  onCreateClick: () => void;
  isDisabled?: boolean;
  extraActions?: ExtraAction[];
};

export type TableStoreProps<TData extends Record<string, any>> = {
  pagination: PaginationState;
  exportModalOpen: boolean;
  columnVisibility: VisibilityState;
  rowSelection: RowSelectionState;
  currentRecord: TData | undefined;
  columnFilters: ColumnFiltersState;
  sorting: ExtendedSortingState<TData>;
  showCreateModal: boolean;
  showFilterDialog: boolean;
  editModalOpen: boolean;
  initialPageSize: number;
  defaultSort: ExtendedSortingState<TData>;
  showImportModal: boolean;
  setInitialPageSize: OnChangeFn<number>;
  setDefaultSort: OnChangeFn<ExtendedSortingState<TData>>;
  onDataChange?: (data: TData[]) => void;
};

export type TableSheetProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
};

export type EditTableSheetProps<TData extends Record<string, any>> = {
  currentRecord?: TData;
  isLoading?: boolean;
  error?: Error | null;
  // New properties for independent modal data fetching
  useIndependentFetch?: boolean;
  apiEndpoint?: API_ENDPOINTS;
  queryKey?: string;
};

type CurrentRecord<TData extends Record<string, unknown>> = TData | undefined;
type SetCurrentRecord<TData extends Record<string, unknown>> = (
  record: TData | undefined,
) => void;

interface DataTableState<TData extends Record<string, unknown>> {
  pagination: PaginationState;
  setPagination: OnChangeFn<PaginationState>;
  rowSelection: RowSelectionState;
  setRowSelection: OnChangeFn<RowSelectionState>;
  currentRecord: CurrentRecord<TData>;
  setCurrentRecord: SetCurrentRecord<TData>;
  columnVisibility: VisibilityState;
  setColumnVisibility: OnChangeFn<VisibilityState>;
  columnFilters: ColumnFilter[];
  setColumnFilters: OnChangeFn<ColumnFilter[]>;
  sorting: ExtendedSortingState<TData>;
  setSorting: OnChangeFn<ExtendedSortingState<TData>>;
  showFilterDialog: boolean;
  setShowFilterDialog: OnChangeFn<boolean>;
  initialPageSize: number;
  setInitialPageSize: OnChangeFn<number>;
  defaultSort: SortingState;
  setDefaultSort: OnChangeFn<SortingState>;
  onDataChange?: (data: TData[]) => void;
}

type DataTableProps<TData extends Record<string, any>> = {
  columns: ColumnDef<TData>[];
  name: string;
  link: API_ENDPOINTS;
  queryKey: string;
  TableModal?: React.ComponentType<TableSheetProps>;
  TableEditModal?: React.ComponentType<EditTableSheetProps<TData>>;
  exportModelName: string;
  extraSearchParams?: Record<string, any>;
  resource: Resource;
  initialPageSize?: number;
  includeHeader?: boolean;
  includeOptions?: boolean;
  pageSizeOptions?: Readonly<number[]>;
  extraActions?: ExtraAction[];
  getRowClassName?: (row: Row<TData>) => string;
  liveMode?: LiveModeTableConfig;
  contextMenuActions?: ContextMenuAction<TData>[];
};

type DataTableBodyProps<TData extends Record<string, any>> = {
  table: Table<TData>;
  columns: ColumnDef<TData>[];
  liveMode?: {
    enabled: boolean;
    connected: boolean;
    showToggle?: boolean;
    onToggle?: (enabled: boolean) => void;
    autoRefresh?: boolean;
    onAutoRefreshToggle?: (autoRefresh: boolean) => void;
  };
};

export type {
  DataTableBodyProps,
  DataTableCreateButtonProps,
  DataTableProps,
  DataTableState,
  ExtraAction
};

