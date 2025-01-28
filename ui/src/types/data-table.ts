import { DataTableConfig } from "@/config/data-table";
import { filterSchema } from "@/lib/parsers";
import { API_ENDPOINTS } from "@/types/server";
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
import { z } from "zod";

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

export type Filter<TData> = Prettify<
  Omit<z.infer<typeof filterSchema>, "id"> & {
    id: StringKeyOf<TData>;
  }
>;

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
type TableRowAction<TData> = {
  row: Row<TData>;
  type: "update" | "delete";
};

type DataTableCreateButtonProps = {
  name: string;
  exportModelName: string;
  isDisabled?: boolean;
  onCreateClick?: () => void;
};

export type DataTableViewOptionsProps<TData> = {
  table: Table<TData>;
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

export type EditTableSheetProps<TData extends Record<string, any>> =
  TableSheetProps & {
    currentRecord?: TData;
    isLoading?: boolean;
    error?: Error | null;
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
  // filterFields: DataTableAdvancedFilterField<TData>[];
  // filterColumn: string;
  TableModal?: React.ComponentType<TableSheetProps>;
  TableEditModal?: React.ComponentType<EditTableSheetProps<TData>>;
  exportModelName: string;
  extraSearchParams?: Record<string, any>;
  // permissionName: string;
  initialPageSize?: number;
  defaultSort?: SortingState;
  // onDataChange?: (data: TData[]) => void;
  pageSizeOptions?: Readonly<number[]>;
};

type DataTableBodyProps<TData extends Record<string, any>> = {
  table: Table<TData>;
};

export type {
  DataTableBodyProps,
  DataTableCreateButtonProps,
  DataTableProps,
  DataTableState,
  TableRowAction
};

