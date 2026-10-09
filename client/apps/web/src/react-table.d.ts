import "@tanstack/react-table";
import type {
  DataTableColumnAggregate,
  DataTableFilterField,
  FilterOperator,
  FilterVariant,
} from "@trenova/shared/types/data-table";
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
    /** For a `record` filter, the kind of record the column's values name (see record-filter-sources). */
    filterRecord?: string;
    /** Totals this column across every row the filters match, in the table's totals row. */
    aggregate?: DataTableColumnAggregate;
    /**
     * The server field this column's header counts values of, so the header can filter
     * by a value with its count beside it. Defaults to `apiField` for select and record
     * filters; `false` turns it off.
     */
    facetField?: string | false;
    /** More ways to filter by this column, such as a customer's name beside the customer itself. */
    extraFilters?: DataTableFilterField[];
    defaultFilterOperator?: FilterOperator;
    exportable?: boolean;
    exportValue?: (row: any) => unknown;
    [key: string]: any;
  }
}
