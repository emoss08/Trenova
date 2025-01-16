import { useDataTableQuery } from "@/hooks/use-data-table-query";
import { useDataTableState } from "@/hooks/use-data-table-state";
import { DataTableProps } from "@/types/data-table";
import { PaginationResponse } from "@/types/server";
import {
  getCoreRowModel,
  getFacetedRowModel,
  getFacetedUniqueValues,
  getFilteredRowModel,
  getPaginationRowModel,
  getSortedRowModel,
  useReactTable,
} from "@tanstack/react-table";
import { parseAsInteger, useQueryState } from "nuqs";
import { useCallback, useMemo, useTransition } from "react";
import { Table } from "../ui/table";
import { DataTableBody } from "./_components/data-table-body";
import { DataTableHeader } from "./_components/data-table-header";
import { DataTablePagination } from "./_components/data-table-pagination";
import {
  DataTableCreateButton,
  DataTableViewOptions,
} from "./_components/data-table-view-options";

const DEFAULT_PAGE_SIZE = 10;
const DEFAULT_PAGE_SIZE_OPTIONS = [10, 20, 30, 40, 50] as const;

const searchParams = {
  page: parseAsInteger.withDefault(1),
  pageSize: parseAsInteger.withDefault(10),
};

export function DataTable<TData extends Record<string, any>>({
  columns,
  link,
  extraSearchParams,
  queryKey,
  name,
  TableModal,
  TableEditModal,
  initialPageSize = DEFAULT_PAGE_SIZE,
  pageSizeOptions = DEFAULT_PAGE_SIZE_OPTIONS,
}: DataTableProps<TData>) {
  const [isTransitioning, startTransition] = useTransition();

  // Use URL state as single source of truth
  const [page, setPage] = useQueryState(
    "page",
    searchParams.page.withOptions({
      startTransition,
      shallow: false,
    }),
  );

  const [pageSize, setPageSize] = useQueryState(
    "pageSize",
    searchParams.pageSize.withOptions({
      startTransition,
      shallow: false,
    }),
  );

  // Derive pagination state from URL
  const pagination = useMemo(
    () => ({
      pageIndex: (page ?? 1) - 1,
      pageSize: pageSize ?? initialPageSize,
    }),
    [page, pageSize, initialPageSize],
  );

  const {
    rowSelection,
    setRowSelection,
    currentRecord,
    setCurrentRecord,
    columnVisibility,
    setColumnVisibility,
    columnFilters,
    setColumnFilters,
    sorting,
    // setSorting,
    showCreateModal,
    setShowCreateModal,
    // showFilterDialog,
    // setShowFilterDialog,
    editModalOpen,
    setEditModalOpen,
  } = useDataTableState<TData>();

  const dataQuery = useDataTableQuery<PaginationResponse<TData>>(
    queryKey,
    link,
    pagination,
    extraSearchParams,
  );

  // Memoize handlers to prevent unnecessary re-renders
  const handlePageChange = useCallback(
    (newPage: number) => {
      startTransition(() => {
        setPage(newPage);
      });
    },
    [setPage],
  );

  const handlePageSizeChange = useCallback(
    (newPageSize: number) => {
      startTransition(() => {
        setPage(1);
        setPageSize(newPageSize);
      });
    },
    [setPage, setPageSize],
  );

  const table = useReactTable({
    data: (dataQuery.data?.results as unknown as TData[]) ?? [],
    columns,
    pageCount: Math.ceil(
      (dataQuery.data?.count ?? 0) / (pageSize ?? initialPageSize),
    ),
    state: {
      pagination,
      sorting,
      columnFilters,
      rowSelection,
      columnVisibility,
    },
    manualPagination: true,
    enableRowSelection: true,
    onRowSelectionChange: setRowSelection,
    // onSortingChange: setSorting,
    onColumnFiltersChange: setColumnFilters,
    onColumnVisibilityChange: setColumnVisibility,
    getCoreRowModel: getCoreRowModel(),
    getFilteredRowModel: getFilteredRowModel(),
    getPaginationRowModel: getPaginationRowModel(),
    getSortedRowModel: getSortedRowModel(),
    getFacetedRowModel: getFacetedRowModel(),
    getFacetedUniqueValues: getFacetedUniqueValues(),
  });

  const isLoading = dataQuery.isLoading || isTransitioning;

  return (
    <div className="mt-2 flex flex-col gap-3 overflow-y-hidden overflow-x-scroll">
      <div className="flex justify-between">
        <div className="flex items-center gap-2"></div>
        <div className="flex items-center gap-2">
          <DataTableViewOptions table={table} />
          <DataTableCreateButton name={name} />
        </div>
      </div>
      <div className="overflow-hidden rounded-md border border-sidebar-border">
        <Table>
          <DataTableHeader table={table} />
          <DataTableBody
            setCurrentRecord={setCurrentRecord}
            setEditModalOpen={setEditModalOpen}
            table={table}
          />
        </Table>
      </div>
      <DataTablePagination
        table={table}
        totalCount={dataQuery.data?.count ?? 0}
        pageSizeOptions={pageSizeOptions}
        isLoading={isLoading}
        onPageChange={handlePageChange}
        onPageSizeChange={handlePageSizeChange}
      />
      {showCreateModal && TableModal && (
        <TableModal open={showCreateModal} onOpenChange={setShowCreateModal} />
      )}
      {editModalOpen && TableEditModal && (
        <TableEditModal
          open={editModalOpen}
          onOpenChange={setEditModalOpen}
          currentRecord={currentRecord as TData}
        />
      )}
    </div>
  );
}
