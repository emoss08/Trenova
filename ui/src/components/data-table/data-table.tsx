import { API_URL } from "@/constants/env";
import { useDataTableQuery } from "@/hooks/use-data-table-query";
import { DataTableSearchParams } from "@/hooks/use-data-table-state";
import { DataTableProps } from "@/types/data-table";
import { PaginationResponse } from "@/types/server";
import { useQuery } from "@tanstack/react-query";
import {
  getCoreRowModel,
  getFacetedRowModel,
  getFacetedUniqueValues,
  getFilteredRowModel,
  getPaginationRowModel,
  getSortedRowModel,
  useReactTable,
} from "@tanstack/react-table";
import { parseAsString, useQueryState } from "nuqs";
import { useCallback, useEffect, useMemo } from "react";
import { useLocation } from "react-router";
import { toast } from "sonner";
import { Skeleton } from "../ui/skeleton";
import { Table } from "../ui/table";
import { DataTableBody } from "./_components/data-table-body";
import { DataTableFilter } from "./_components/data-table-filters";
import { DataTableHeader } from "./_components/data-table-header";
import { DataTableOptions } from "./_components/data-table-options";
import {
  DataTablePagination,
  PaginationInner,
} from "./_components/data-table-pagination";
import { DataTableSearch } from "./_components/data-table-search";

const entityParams = {
  entityId: parseAsString,
  modal: parseAsString,
};

export function DataTable<TData extends Record<string, any>>({
  columns,
  link,
  extraSearchParams,
  queryKey,
  name,
  exportModelName,
  TableModal,
  TableEditModal,
  initialPageSize = 10,
  includeHeader = true,
  includeOptions = true,
  extraActions,
}: DataTableProps<TData>) {
  // Use URL state as single source of truth
  const location = useLocation();
  const [page] = useQueryState(
    "page",
    DataTableSearchParams.page.withOptions({
      shallow: false,
    }),
  );

  const [pageSize] = useQueryState(
    "pageSize",
    DataTableSearchParams.pageSize.withOptions({
      shallow: false,
    }),
  );

  // Entity URL State management
  const [entityId, setEntityId] = useQueryState(
    "entityId",
    entityParams.entityId.withOptions({
      shallow: false,
    }),
  );
  const [modalType, setModalType] = useQueryState(
    "modal",
    entityParams.modal.withOptions({
      shallow: false,
    }),
  );

  // Process any pending entityId from navigation state
  useEffect(() => {
    const state = location.state as { pendingEntityId?: string } | null;
    if (state?.pendingEntityId) {
      // Set the entityId and modal parameters if we have a pending entityId
      Promise.all([
        setEntityId(state.pendingEntityId, { shallow: false }),
        setModalType("edit", { shallow: false }),
      ]).catch(console.error);

      // Clear the state to prevent reprocessing
      window.history.replaceState(
        { ...state, pendingEntityId: undefined },
        "",
        window.location.href,
      );
    }
  }, [location, setEntityId, setModalType]);

  // Entity Query
  const entityQuery = useQuery({
    queryKey: [queryKey, "entity", link, entityId, extraSearchParams],
    queryFn: async () => {
      if (!entityId) return null;
      const fetchURL = new URL(`${API_URL}${link}${entityId}`);

      if (extraSearchParams) {
        Object.entries(extraSearchParams).forEach(([key, value]) =>
          fetchURL.searchParams.set(key, value),
        );
      }

      const response = await fetch(fetchURL.href, {
        credentials: "include",
      });

      if (!response.ok) {
        toast.error("Failed to fetch entity");
        throw new Error("Failed to fetch entity");
      }

      return response.json();
    },
    enabled: !!entityId,
    staleTime: 30000, // 30 seconds
  });

  // Update the handleModalClose function to properly clear both parameters
  const handleEditModalClose = useCallback(async () => {
    await Promise.all([
      setEntityId(null, { shallow: false }),
      setModalType(null, { shallow: false }),
    ]);
  }, [setEntityId, setModalType]);

  const handleCreateModalClose = useCallback(async () => {
    await setModalType(null, { shallow: false });
  }, [setModalType]);

  useEffect(() => {
    // Only handle edit modal consistency
    if (entityId && !modalType) {
      setModalType("edit", { shallow: false }).catch(console.error);
    }

    // Only clear modal if we're in edit mode and lose the entityId
    if (!entityId && modalType === "edit") {
      setModalType(null, { shallow: false }).catch(console.error);
    }
  }, [entityId, modalType, setModalType]);

  // Derive pagination state from URL
  const pagination = useMemo(
    () => ({
      pageIndex: (page ?? 1) - 1,
      pageSize: pageSize ?? initialPageSize,
    }),
    [page, pageSize, initialPageSize],
  );

  const dataQuery = useDataTableQuery<PaginationResponse<TData>>(
    queryKey,
    link,
    pagination,
    extraSearchParams,
  );

  // Memoized placeholder data with loading skeleton
  const placeholderData = useMemo(
    () =>
      dataQuery.isLoading
        ? Array.from({ length: pagination.pageSize }, () => ({}) as TData)
        : dataQuery.data?.results || [],
    [dataQuery.isLoading, dataQuery.data, pagination.pageSize],
  );

  // Memoized display columns with loading state
  const displayColumns = useMemo(
    () =>
      dataQuery.isLoading
        ? columns.map((column) => ({
            ...column,
            cell: () => <Skeleton className="h-5 w-full" />,
          }))
        : columns,
    [dataQuery.isLoading, columns],
  );

  const table = useReactTable({
    data: placeholderData as TData[],
    columns: displayColumns,
    pageCount: Math.ceil(
      (dataQuery.data?.count ?? 0) / (pageSize ?? initialPageSize),
    ),
    rowCount: dataQuery.data?.count ?? 0,
    state: {
      pagination,
      // sorting,
      // columnFilters,
      // rowSelection,
      // columnVisibility,
    },
    manualPagination: true,
    enableRowSelection: true,
    // onRowSelectionChange: setRowSelection,
    // onColumnFiltersChange: setColumnFilters,
    // onColumnVisibilityChange: setColumnVisibility,
    getCoreRowModel: getCoreRowModel(),
    getFilteredRowModel: getFilteredRowModel(),
    getPaginationRowModel: getPaginationRowModel(),
    getSortedRowModel: getSortedRowModel(),
    getFacetedRowModel: getFacetedRowModel(),
    getFacetedUniqueValues: getFacetedUniqueValues(),
  });

  const isEditModalOpen = Boolean(entityId && modalType === "edit");
  const isCreateModalOpen = Boolean(modalType === "create");
  const isEntityLoading = entityQuery.isLoading;
  const isEntityError = entityQuery.error;

  return (
    <div className="mt-2 flex flex-col gap-3">
      {includeOptions && (
        <DataTableOptions>
          <DataTableSearch />
          <DataTableFilter
            table={table}
            name={name}
            exportModelName={exportModelName}
            extraActions={extraActions}
            setModalType={(type) => setModalType(type, { shallow: false })}
          />
        </DataTableOptions>
      )}
      <DataTableInner>
        <Table>
          {includeHeader && <DataTableHeader table={table as any} />}
          <DataTableBody table={table as any} />
        </Table>
      </DataTableInner>
      <DataTablePagination>
        <PaginationInner table={table} />
      </DataTablePagination>
      {TableModal && isCreateModalOpen && (
        <TableModal
          open={isCreateModalOpen}
          onOpenChange={handleCreateModalClose}
        />
      )}
      {TableEditModal && isEditModalOpen && (
        <TableEditModal
          open={isEditModalOpen}
          onOpenChange={handleEditModalClose}
          isLoading={isEntityLoading}
          currentRecord={entityQuery.data}
          error={isEntityError}
        />
      )}
    </div>
  );
}

export function DataTableInner({ children }: { children: React.ReactNode }) {
  return (
    <div className="rounded-md border border-sidebar-border">{children}</div>
  );
}
