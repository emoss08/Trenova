import "@tanstack/react-table";
import type {
  FilterOperator,
  FilterState,
  FilterVariant,
} from "./types/data-table";
import type { SelectOption } from "./types/fields";

declare module "@tanstack/react-table" {
  interface TableMeta<TData extends Record<string, any>> {
    getRowClassName?: (row: Row<TData>) => string;
  }

  interface ColumnMeta {
    headerClassName?: string;
    cellClassName?: string;
    label?: string;
    apiField?: string;
    filterable?: boolean;
    sortable?: boolean;
    filterType?: FilterVariant;
    filterOptions?: SelectOption[];
    defaultFilterOperator?: FilterOperator;
    [key: string]: any;
  }

  interface TableState {
    filters: FilterState;
  }

  interface FilterFns {
    inDateRange?: FilterFn<any>;
    arrSome?: FilterFn<any>;
  }

  interface ColumnFiltersOptions<TData extends RowData> {
    filterFns?: Record<string, FilterFn<TData>>;
  }
}
