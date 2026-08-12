import "@tanstack/react-table";
import type { FilterOperator, FilterVariant } from "@trenova/shared/types/data-table";
import type { SelectOption } from "@trenova/shared/types/fields";

declare module "@tanstack/react-table" {
  interface TableMeta<in out TFeatures extends TableFeatures, in out TData extends RowData> {
    getRowClassName?: (row: Row<TFeatures, TData>) => string;
  }

  interface ColumnMeta<
    in out TFeatures extends TableFeatures,
    in out TData extends RowData,
    TValue extends CellData = CellData,
  > {
    headerClassName?: string;
    cellClassName?: string;
    label?: string;
    apiField?: string;
    filterable?: boolean;
    sortable?: boolean;
    filterType?: FilterVariant;
    filterOptions?: SelectOption[];
    defaultFilterOperator?: FilterOperator;
    exportable?: boolean;
    exportValue?: (row: any) => unknown;
    [key: string]: any;
  }
}
