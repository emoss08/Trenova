"use no memo";
import { useDataTableQuery } from "@/hooks/use-data-table-query";
import { searchParamsParser } from "@/hooks/use-data-table-state";
import { usePermissions } from "@/hooks/use-permissions";
import { DataTableProps } from "@/types/data-table";
import { Action } from "@/types/roles-permissions";
import {
  getCoreRowModel,
  getPaginationRowModel,
  RowSelectionState,
  Table as TableType,
  useReactTable,
  VisibilityState,
} from "@tanstack/react-table";
import { useLocalStorage } from "@uidotdev/usehooks";
import { useQueryStates } from "nuqs";
import {
  lazy,
  Suspense,
  useCallback,
  useEffect,
  useMemo,
  useState,
} from "react";
import { DataTablePermissionDeniedSkeleton } from "../ui/permission-skeletons";
import { Table } from "../ui/table";
import { DataTableBody } from "./_components/data-table-body";
import { DataTableHeader } from "./_components/data-table-header";
import { DataTableOptions } from "./_components/data-table-options";
import { PaginationInner } from "./_components/data-table-pagination";
import { DataTableSearch } from "./_components/data-table-search";
import { DataTableProvider } from "./data-table-provider";

const DataTableActions = lazy(() => import("./_components/data-table-actions"));

export function DataTable<TData extends Record<string, any>>({
  columns,
  link,
  queryKey,
  extraSearchParams,
  name,
  exportModelName,
  TableModal,
  TableEditModal,
  initialPageSize = 10,
  includeHeader = true,
  includeOptions = true,
  extraActions,
  resource,
  getRowClassName,
}: DataTableProps<TData>) {
  const [searchParams, setSearchParams] = useQueryStates(searchParamsParser);
  const { page, pageSize, entityId, modalType } = searchParams;
  const [rowSelection, setRowSelection] = useState<RowSelectionState>(
    entityId ? { [entityId]: true } : {},
  );
  const { can } = usePermissions();
  const [columnVisibility, setColumnVisibility] =
    useLocalStorage<VisibilityState>(
      `${name.toLowerCase()}-column-visibility`,
      {},
    );

  // Derive pagination state from URL
  const pagination = useMemo(
    () => ({
      pageIndex: (page ?? 1) - 1,
      pageSize: pageSize ?? initialPageSize,
    }),
    [page, pageSize, initialPageSize],
  );

  const dataQuery = useDataTableQuery<TData>(
    queryKey,
    link,
    pagination,
    extraSearchParams,
  );

  const table = useReactTable({
    data: dataQuery.data?.results || [],
    columns: columns,
    pageCount: Math.ceil(
      (dataQuery.data?.count ?? 0) / (pageSize ?? initialPageSize),
    ),
    rowCount: dataQuery.data?.count ?? 0,
    state: {
      pagination,
      rowSelection,
      columnVisibility,
    },
    onColumnVisibilityChange: setColumnVisibility,
    enableMultiRowSelection: false,
    columnResizeMode: "onChange",
    manualPagination: true,
    enableRowSelection: true,
    getRowId: (row) => row.id,
    onRowSelectionChange: setRowSelection,
    getCoreRowModel: getCoreRowModel(),
    getPaginationRowModel: getPaginationRowModel(),
    meta: { getRowClassName },
  });

  const selectedRow = useMemo(() => {
    if (
      (dataQuery.isLoading || dataQuery.isFetching) &&
      !dataQuery.data?.results.length
    )
      return;
    const selectedRowKey = Object.keys(rowSelection)?.[0];

    return table
      .getCoreRowModel()
      .flatRows.find((row) => row.id === selectedRowKey);
  }, [
    rowSelection,
    table,
    dataQuery.isLoading,
    dataQuery.isFetching,
    dataQuery.data?.results,
  ]);

  useEffect(() => {
    if (dataQuery.isLoading || dataQuery.isFetching) return;
    if (modalType === "create") return; // * Don't override "create" modalType
    if (Object.keys(rowSelection)?.length && !selectedRow) {
      setSearchParams({ entityId: null, modalType: null });
      setRowSelection({});
    } else {
      setSearchParams({
        entityId: selectedRow?.id || null,
        modalType: selectedRow ? "edit" : null,
      });
    }
  }, [
    rowSelection,
    selectedRow,
    setSearchParams,
    dataQuery.isLoading,
    dataQuery.isFetching,
    modalType,
  ]);

  const handleCreateClick = useCallback(() => {
    setSearchParams({ modalType: "create", entityId: null });
  }, [setSearchParams]);

  const handleCreateModalClose = useCallback(() => {
    setSearchParams({ modalType: null, entityId: null });
  }, [setSearchParams]);

  const isCreateModalOpen = Boolean(modalType === "create");

  return (
    <DataTableProvider
      table={table}
      columns={columns}
      isLoading={dataQuery.isFetching || dataQuery.isLoading}
      pagination={pagination}
      rowSelection={rowSelection}
      columnVisibility={columnVisibility}
    >
      {can(resource, Action.Read) ? (
        <>
          {includeOptions && (
            <DataTableOptions>
              <DataTableSearch />
              <Suspense fallback={<div>Loading...</div>}>
                <DataTableActions
                  table={table as unknown as TableType<unknown>}
                  name={name}
                  resource={resource}
                  exportModelName={exportModelName}
                  extraActions={extraActions}
                  handleCreateClick={handleCreateClick}
                />
              </Suspense>
            </DataTableOptions>
          )}
          <Table className="rounded-md border-x border-border border-separate border-spacing-0">
            {includeHeader && <DataTableHeader table={table} />}
            <DataTableBody table={table} columns={columns} />
          </Table>
          <PaginationInner table={table} />
          {TableModal && isCreateModalOpen && (
            <TableModal
              open={isCreateModalOpen}
              onOpenChange={handleCreateModalClose}
            />
          )}
          {TableEditModal && (
            <TableEditModal
              isLoading={dataQuery.isFetching || dataQuery.isLoading}
              currentRecord={selectedRow?.original}
              error={dataQuery.error}
            />
          )}
        </>
      ) : (
        <DataTablePermissionDeniedSkeleton
          resource={resource}
          action={Action.Read}
        />
      )}
    </DataTableProvider>
  );
}
