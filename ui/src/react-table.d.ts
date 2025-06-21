import "@tanstack/react-table";
import type { SelectOption } from "./types/fields";

declare module "@tanstack/react-table" {
  // https://github.com/TanStack/table/issues/44#issuecomment-1377024296
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
    filterType?: "text" | "select" | "date" | "number" | "boolean";
    filterOptions?: SelectOption[];
    defaultFilterOperator?: FilterOperator;

    // Allow for other existing metadata
    [key: string]: any;
  }

  interface FilterFns {
    inDateRange?: FilterFn<any>;
    arrSome?: FilterFn<any>;
  }

  // https://github.com/TanStack/table/discussions/4554
  interface ColumnFiltersOptions<TData extends RowData> {
    filterFns?: Record<string, FilterFn<TData>>;
  }
}
