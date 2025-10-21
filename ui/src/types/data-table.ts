import { DataTableConfig } from "@/config/data-table";
import type {
  FilterFieldSchema,
  FilterStateSchema,
  SortFieldSchema,
} from "@/lib/schemas/table-configuration-schema";
import { API_ENDPOINTS } from "@/types/server";
import { IconDefinition } from "@fortawesome/pro-regular-svg-icons";
import type {
  ColumnDef,
  ColumnFiltersState,
  ColumnSort,
  OnChangeFn,
  PaginationState,
  Row,
  RowSelectionState,
  Table,
  VisibilityState,
} from "@tanstack/react-table";
import React from "react";
import type { Resource } from "./audit-entry";
import type { LiveModeTableConfig } from "./live-mode";

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

export type ExtraAction = {
  key: string;
  label: string;
  icon?: IconDefinition;
  endContent?: React.ReactNode;
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
  onClick?: (row: Row<TData>) => void;
  separator?: "before" | "after";
  subActions?: ContextMenuAction<TData>[];
}

export type DataTableCreateButtonProps = {
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
  useIndependentFetch?: boolean;
  apiEndpoint?: API_ENDPOINTS;
  queryKey?: string;
};

export type DataTableProps<TData extends Record<string, any>> = {
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

export type DataTableBodyProps<TData extends Record<string, any>> = {
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

export interface EnhancedQueryParams {
  limit?: number;
  offset?: number;
  query?: string;

  filters?: FilterFieldSchema[];
  sort?: SortFieldSchema[];
}

export type EnhancedColumnDef<TData, TValue = unknown> = ColumnDef<
  TData,
  TValue
>;

export interface Config {
  enableFiltering?: boolean;
  enableSorting?: boolean;
  enableMultiSort?: boolean;
  maxFilters?: number;
  maxSorts?: number;
  searchDebounce?: number;
  showFilterUI?: boolean;
  showSortUI?: boolean;
}

export interface FilterPreset {
  id: string;
  name: string;
  description?: string;
  filters: FilterStateSchema["filters"];
  sort: FilterStateSchema["sort"];
  globalSearch?: string;
  createdAt: Date;
  updatedAt: Date;
}

export interface URLFilterParams {
  [key: string]: string | string[] | undefined;
}

export interface FilterUtils {
  serializeToURL(state: FilterStateSchema): URLFilterParams;
  deserializeFromURL(params: URLFilterParams): FilterStateSchema;
  serializeForAPI(state: FilterStateSchema): EnhancedQueryParams;
  validateFilterState(
    state: FilterStateSchema,
    columns: EnhancedColumnDef<any>[],
  ): boolean;
}
