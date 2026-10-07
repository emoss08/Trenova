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
import type { IconComponent } from "@trenova/shared/components/icons";
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
  icon?: IconComponent;
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
  icon?: IconComponent;
  /** A keyboard hint shown beside the action, already formatted for the platform. */
  shortcut?: string;
  variant?: "default" | "destructive";
  group?: string | { id: string; label: string };
  onClick: (row: Row<TData>) => void | Promise<unknown>;
  hidden?: (row: Row<TData>) => boolean;
  disabled?: (row: Row<TData>) => boolean;
  isPending?: (row: Row<TData>) => boolean;
};

export type AddRecordAction = {
  id: string;
  label: string;
  description?: string;
  icon?: IconComponent;
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
  /** Columns pinned from the first paint and kept pinned under any saved view. */
  initialColumnPinning?: { left: string[]; right: string[] };
  onCellEditCommit?: CellEditCommitFn<TData>;
  /**
   * Draws the table's own empty state in place of the generic one. It is told
   * whether a search or filter is what emptied the table, and given the
   * table's own clear-filters action to offer.
   */
  renderEmptyState?: (state: DataTableEmptyStateRenderProps) => React.ReactNode;
  /**
   * Filters the table's surroundings put on it: a range or a record chosen
   * beside the table rather than in its filter builder. They are ANDed ahead
   * of the person's own filters in every query the table makes, and are not
   * shown as chips, because the control that set them is already on screen.
   */
  scopeFilters?: FieldFilter[];
  /**
   * Offers the browser-side CSV export to people who may export the
   * resource. Off for a table whose export is a server-side record of its
   * own, such as an audited, signed file.
   */
  enableExport?: boolean;
  grouping?: DataTableGrouping<TData>;
  expansion?: DataTableExpansion<TData>;
  keyboard?: DataTableKeyboard<TData>;
  toolbar?: DataTableToolbarSlots;
  alternateView?: DataTableAlternateView;
  initialDensity?: "comfortable" | "compact";
  /** Placed at the start of the pagination footer. */
  footerLeading?: React.ReactNode;
};

export type DataTableGroupKey = string | number;

export type DataTableGroup = {
  key: DataTableGroupKey;
  label: string;
  /** Class for the group's colour square; a token utility such as `bg-danger`. */
  swatchClassName?: string;
  /** Rows in the whole group under the current filters, not on this page. */
  count?: number;
  /** Right-aligned summary for the whole group, such as its revenue. */
  aggregate?: React.ReactNode;
};

/**
 * Rows grouped by a server-sortable field. The table sorts by the field ahead
 * of the person's own sort, so pagination runs over the grouped order, and
 * drops collapsed groups with a `notin` filter, so a collapsed group costs no
 * rows. Group totals come from the caller, who can count past the page.
 */
/** The filters that drop collapsed groups from the server's pages. */
export type DataTableGroupScope = {
  fieldFilters: FieldFilter[];
  filterGroups: FilterGroup[];
};

export type DataTableGrouping<TData> = {
  field: string;
  direction?: SortDirection;
  /**
   * Sorted right after the group field, ahead of the person's own sort, so
   * rows whose group field ties but whose key differs (two customers with one
   * name) never interleave.
   */
  tieBreakers?: SortField[];
  /**
   * How collapsed groups are dropped when a key is not the field's own value,
   * such as the day of a timestamp. Defaults to `field notin keys`.
   */
  collapsedScope?: (keys: readonly DataTableGroupKey[]) => DataTableGroupScope;
  groups: DataTableGroup[];
  getGroupKey: (row: TData) => DataTableGroupKey;
  collapsedKeys: readonly DataTableGroupKey[];
  onToggleGroup: (key: DataTableGroupKey) => void;
};

export type DataTableExpandedRowContext = {
  collapse: () => void;
};

/** One row at a time opens inline beneath itself. */
export type DataTableExpansion<TData extends RowData> = {
  expandedRowId: string | null;
  onExpandedRowIdChange: (rowId: string | null) => void;
  renderExpandedRow: (row: Row<TData>, context: DataTableExpandedRowContext) => React.ReactNode;
};

/**
 * Page-level keyboard control of the table: a row cursor moved with J/K or
 * the arrows, Enter to expand, X to select, Esc to back out one step at a
 * time. Only one table on a page should take it.
 */
export type DataTableKeyboard<TData = unknown> = {
  enabled: boolean;
  cursorRowId: string | null;
  onCursorRowIdChange: (rowId: string | null) => void;
  /** Keys that act on the cursor row, or the open row when there is no cursor. */
  rowShortcuts?: DataTableRowShortcut<TData>[];
};

export type DataTableRowShortcut<TData> = {
  /** Lowercase key; an Alt chord is matched by physical key. */
  key: string;
  mod?: boolean;
  alt?: boolean;
  run: (row: TData) => void;
};

export type DataTableViewContext = {
  queryOptions: Omit<DataTableQueryOptions, "cursor">;
};

/** One entry the search field offers while it is focused and empty. */
export type DataTableSearchSuggestion = {
  key: string;
  label: string;
  count?: number;
  /** A token class for the dot drawn ahead of the label. */
  dotClassName?: string;
  /** Already applied, so the entry reads as chosen. */
  selected?: boolean;
  onSelect: () => void;
};

export type DataTableSearchSuggestions = {
  title: string;
  items: DataTableSearchSuggestion[];
  /** Told when the list opens and closes, so counts can be fetched only while it shows. */
  onOpenChange?: (open: boolean) => void;
};

/** Filters a table applies outside the field filters, shown and cleared with the filter chips. */
export type DataTableExtraChips = {
  items: { key: string; label: string; onRemove: () => void }[];
  onClear: () => void;
};

export type DataTableToolbarSlots = {
  /** Entries the search field offers while it is focused and empty. */
  searchSuggestions?: DataTableSearchSuggestions;
  /** A key that focuses the search field from anywhere on the page. */
  searchShortcut?: string;
  /** Filters the host applies itself, drawn as chips beside the table's own. */
  chips?: DataTableExtraChips;
  /** Controls placed before the display menu. */
  trailing?: React.ReactNode;
  /** Controls placed after the saved views. */
  end?: React.ReactNode;
  /**
   * Classes that let a narrow host collapse the built-in controls: `label` is
   * put on button labels so they can give way to icons, `secondary` on the
   * display menu and saved views so they can hide.
   */
  responsive?: { label: string; secondary: string };
};

/** Draws the rows another way, such as a timeline or a map, under the same toolbar and filters. */
export type DataTableAlternateView = {
  active: boolean;
  render: (context: DataTableViewContext) => React.ReactNode;
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
