import { useTableStore as store } from "@/stores/table-store";
import { DataTableState } from "@/types/data-table";

export function useDataTableState<
  TData extends Record<string, any>,
>(): DataTableState<TData> {
  const [{ pageIndex, pageSize }, setPagination] = store.use("pagination");
  const [rowSelection, setRowSelection] = store.use("rowSelection");
  const [currentRecord, setCurrentRecord] = store.use("currentRecord");
  const [columnVisibility, setColumnVisibility] = store.use("columnVisibility");
  const [columnFilters, setColumnFilters] = store.use("columnFilters");
  const [sorting, setSorting] = store.use("sorting");
  const [showFilterDialog, setShowFilterDialog] = store.use("showFilterDialog");
  const [initialPageSize, setInitialPageSize] = store.use("initialPageSize");
  const [defaultSort, setDefaultSort] = store.use("defaultSort");

  return {
    pagination: { pageIndex, pageSize },
    setPagination,
    rowSelection,
    setRowSelection,
    currentRecord,
    setCurrentRecord,
    columnVisibility,
    setColumnVisibility,
    columnFilters,
    setColumnFilters,
    sorting,
    setSorting,
    showFilterDialog,
    setShowFilterDialog,
    initialPageSize,
    setInitialPageSize,
    defaultSort,
    setDefaultSort,
  };
}
