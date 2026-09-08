import type {
  CellContext as TanStackCellContext,
  Cell as TanStackCell,
  ColumnDef as TanStackColumnDef,
  Column as TanStackColumn,
  HeaderContext as TanStackHeaderContext,
  HeaderGroup as TanStackHeaderGroup,
  Header as TanStackHeader,
  ReactTable as TanStackReactTable,
  Row as TanStackRow,
  CellData,
  RowData,
} from "@tanstack/react-table";
import type { LucideIcon } from "lucide-react";
import { z } from "zod";
import type { CellEditCommitFn } from "../lib/cell-editing-feature";
import type { DataTableFeatures } from "../lib/table-features";
import type { SelectOption } from "./fields";
import type { ResultOf, VariablesOf } from "@graphql-typed-document-node/core";
import type { GraphQLExecutableDocument, TypedGraphQLDocument } from "./graphql";
import type { ConnectionKeys, DataTableRow } from "./graphql-connection";
import type { API_ENDPOINTS } from "./server";

export type {
  ConnectionKeys,
  ConnectionNode,
  DataTableRow,
  UnmaskFragments,
} from "./graphql-connection";

export type ColumnDef<
  TData extends RowData,
  TValue extends CellData = CellData,
> = TanStackColumnDef<DataTableFeatures, TData, TValue>;

export type Column<TData extends RowData, TValue extends CellData = CellData> = TanStackColumn<
  DataTableFeatures,
  TData,
  TValue
>;

export type Row<TData extends RowData> = TanStackRow<DataTableFeatures, TData>;

export type Table<TData extends RowData> = TanStackReactTable<DataTableFeatures, TData>;

export type Cell<TData extends RowData, TValue extends CellData = CellData> = TanStackCell<
  DataTableFeatures,
  TData,
  TValue
>;

export type Header<TData extends RowData, TValue extends CellData = CellData> = TanStackHeader<
  DataTableFeatures,
  TData,
  TValue
>;

export type HeaderGroup<TData extends RowData> = TanStackHeaderGroup<DataTableFeatures, TData>;

export type CellContext<
  TData extends RowData,
  TValue extends CellData = CellData,
> = TanStackCellContext<DataTableFeatures, TData, TValue>;

export type HeaderContext<
  TData extends RowData,
  TValue extends CellData = CellData,
> = TanStackHeaderContext<DataTableFeatures, TData, TValue>;

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

export type PanelMode = "create" | "edit";

export type DataTablePanelProps<TData> = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  mode: PanelMode;
  row: TData | null;
};

type BaseDockAction = {
  id: string;
  label: string;
  loadingLabel?: string;
  icon?: LucideIcon;
  variant?: "default" | "destructive";
  clearSelectionOnSuccess?: boolean;
};

type SimpleDockAction<TData> = BaseDockAction & {
  type?: "simple";
  onClick: (selectedRows: TData[]) => void | Promise<void>;
};

type SelectDockAction<TData> = BaseDockAction & {
  type: "select";
  options: ReadonlyArray<{
    value: string;
    label: string;
    color?: string;
    description?: string;
  }>;
  onSelect: (selectedRows: TData[], value: string) => void | Promise<void>;
  selectPlaceholder?: string;
};

export type DockAction<TData> = SimpleDockAction<TData> | SelectDockAction<TData>;

export type RowAction<TData extends RowData> = {
  id: string;
  label: string;
  icon?: LucideIcon;
  variant?: "default" | "destructive";
  group?: string | { id: string; label: string };
  onClick: (row: Row<TData>) => void;
  hidden?: (row: Row<TData>) => boolean;
  disabled?: (row: Row<TData>) => boolean;
};

export type AddRecordAction = {
  id: string;
  label: string;
  description?: string;
  icon?: LucideIcon;
  onClick: () => void;
};

export type DataTableProps<TData extends Record<string, any>> = {
  columns: ColumnDef<TData>[];
  name: string;
  queryKey: string;
  graphql: DataTableGraphQLSource<TData>;
  resource?: string;
  TableModal?: React.ComponentType<TableSheetProps>;
  TablePanel?: React.ComponentType<DataTablePanelProps<TData>>;
  initialPageSize?: number;
  includeHeader?: boolean;
  includeOptions?: boolean;
  pageSizeOptions?: Readonly<number[]>;
  getRowClassName?: (row: Row<TData>) => string;
  enableRowSelection?: boolean;
  dockActions?: DockAction<TData>[];
  refetchIntervalMs?: number;
  onAddRecord?: () => void;
  addRecordActions?: AddRecordAction[];
  contextMenuActions?: RowAction<TData>[];
  onRowClick?: (row: Row<TData>) => void;
  enableCreateAction?: boolean;
  enableReadOnlyPanel?: boolean;
  initialColumnVisibility?: Record<string, boolean>;
  onCellEditCommit?: CellEditCommitFn<TData>;
  /**
   * Draws the table's own empty state in place of the generic one. It is told
   * whether a search or filter is what emptied the table, and given the
   * table's own clear-filters action to offer.
   */
  renderEmptyState?: (state: DataTableEmptyStateRenderProps) => React.ReactNode;
};

export type DataTableEmptyStateRenderProps = {
  hasActiveFilters: boolean;
  onClearFilters: () => void;
};

export type DataTableGraphQLExtraVariableParams = {
  pageSize: number;
  options?: DataTableQueryOptions;
};

export type DataTableGraphQLVariableSource<TVariables> =
  | TVariables
  | ((params: DataTableGraphQLExtraVariableParams) => TVariables);

export type DataTableReservedInputKeys =
  | "first"
  | "after"
  | "query"
  | "fieldFilters"
  | "filterGroups"
  | "sort";

export type DataTableExtraVariables<TVariables> = Partial<Omit<TVariables, "input">>;

export type DataTableInputExtraVariables<TVariables> = TVariables extends { input?: infer TInput }
  ? [keyof Omit<NonNullable<TInput>, DataTableReservedInputKeys>] extends [never]
    ? never
    : Partial<Omit<NonNullable<TInput>, DataTableReservedInputKeys>>
  : never;

type DataTableRowMapping<TNode, TData> = [TNode] extends [TData]
  ? { mapNode?: (node: TNode) => TData }
  : { mapNode: (node: TNode) => TData };

export type DataTableGraphQLConfig<
  TDocument extends TypedGraphQLDocument<unknown, never>,
  K extends ConnectionKeys<ResultOf<TDocument>>,
  TData = DataTableRow<TDocument, K>,
> = {
  document: TDocument;
  operationName: string;
  connectionKey: K;
  extraVariables?: DataTableGraphQLVariableSource<DataTableExtraVariables<VariablesOf<TDocument>>>;
  inputExtraVariables?: [DataTableInputExtraVariables<VariablesOf<TDocument>>] extends [never]
    ? never
    : DataTableGraphQLVariableSource<DataTableInputExtraVariables<VariablesOf<TDocument>>>;
} & DataTableRowMapping<DataTableRow<TDocument, K>, TData>;

export type DataTableGraphQLSource<TData> = {
  document: GraphQLExecutableDocument;
  operationName: string;
  connectionKey: string;
  extraVariables?: DataTableGraphQLVariableSource<Record<string, unknown>>;
  inputExtraVariables?: DataTableGraphQLVariableSource<Record<string, unknown>>;
  mapNode?(node: unknown): TData;
};

export type DataTableConfigRow<TConfig> =
  TConfig extends DataTableGraphQLSource<infer TData> ? TData : never;

export type DataTableQueryOptions = {
  query?: string;
  fieldFilters?: FieldFilter[];
  filterGroups?: FilterGroup[];
  sort?: SortField[];
  cursor?: string | null;
};

export type DataTableBodyProps<TData extends Record<string, any>> = {
  table: Table<TData>;
  columns: ColumnDef<TData>[];
  contextMenuActions?: RowAction<TData>[];
  onRowClick?: (row: Row<TData>) => void;
};

export const filterOperatorSchema = z.enum([
  "eq",
  "ne",
  "gt",
  "gte",
  "lt",
  "lte",
  "contains",
  "startswith",
  "endswith",
  "ilike",
  "in",
  "notin",
  "isnull",
  "isnotnull",
  "daterange",
  "lastndays",
  "nextndays",
  "today",
  "yesterday",
  "tomorrow",
]);

export type FilterOperator = z.infer<typeof filterOperatorSchema>;

export const filterVariantSchema = z.enum(["text", "number", "select", "date", "boolean"]);

export type FilterVariant = z.infer<typeof filterVariantSchema>;

export const sortDirectionSchema = z.enum(["asc", "desc"]);
export type SortDirection = z.infer<typeof sortDirectionSchema>;

export const fieldFilterSchema = z.object({
  field: z.string(),
  operator: filterOperatorSchema,
  value: z.unknown(),
});
export type FieldFilter = z.infer<typeof fieldFilterSchema>;

export const filterGroupSchema = z.object({
  filters: z.array(fieldFilterSchema),
});

export type FilterGroup = z.infer<typeof filterGroupSchema>;

export const sortFieldSchema = z.object({
  field: z.string(),
  direction: sortDirectionSchema,
});
export type SortField = z.infer<typeof sortFieldSchema>;

export interface FilterState {
  fieldFilters: FieldFilter[];
  filterGroups: FilterGroup[];
  sort: SortField[];
}

export type FilterConnector = "and" | "or";

interface FilterItemBase {
  id: string;
  connector: FilterConnector;
}

export interface SingleFilterItem extends FilterItemBase {
  type: "filter";
  field: string;
  apiField: string;
  label: string;
  operator: FilterOperator;
  value: unknown;
  filterType: FilterVariant;
  filterOptions?: SelectOption[];
}

export interface FilterGroupItem extends FilterItemBase {
  type: "group";
  items: SingleFilterItem[];
}

export type FilterItem = SingleFilterItem | FilterGroupItem;
