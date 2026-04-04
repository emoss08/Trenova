import type { ColumnDef, Row, Table } from "@tanstack/react-table";
import type { LucideIcon } from "lucide-react";
import { z } from "zod";
import type { SelectOption } from "./fields";
import type { API_ENDPOINTS } from "./server";

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

export type DockAction<TData> =
  | SimpleDockAction<TData>
  | SelectDockAction<TData>;

export type RowAction<TData> = {
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
  link: API_ENDPOINTS;
  queryKey: string;
  resource?: string;
  TableModal?: React.ComponentType<TableSheetProps>;
  TablePanel?: React.ComponentType<DataTablePanelProps<TData>>;
  exportModelName: string;
  extraSearchParams?: Record<string, any>;
  initialPageSize?: number;
  includeHeader?: boolean;
  includeOptions?: boolean;
  pageSizeOptions?: Readonly<number[]>;
  getRowClassName?: (row: Row<TData>) => string;
  enableRowSelection?: boolean;
  dockActions?: DockAction<TData>[];
  onAddRecord?: () => void;
  addRecordActions?: AddRecordAction[];
  contextMenuActions?: RowAction<TData>[];
  onRowClick?: (row: Row<TData>) => void;
  preferDetailRowForEdit?: boolean;
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

export const filterVariantSchema = z.enum([
  "text",
  "number",
  "select",
  "date",
  "boolean",
]);

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
