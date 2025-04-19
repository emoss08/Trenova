import { API_URL } from "@/constants/env";
import { useDataTableQuery } from "@/hooks/use-data-table-query";
import { useDataTableState } from "@/hooks/use-data-table-state";
import { DataTableProps } from "@/types/data-table";
import { PaginationResponse } from "@/types/server";
import { faSearch } from "@fortawesome/pro-solid-svg-icons";
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
import { parseAsInteger, parseAsString, useQueryState } from "nuqs";
import { useCallback, useEffect, useMemo, useTransition } from "react";
import { Icon } from "../ui/icons";
import { Input } from "../ui/input";
import { Separator } from "../ui/separator";
import { Skeleton } from "../ui/skeleton";
import { Table } from "../ui/table";
import { TooltipProvider } from "../ui/tooltip";
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
  initialPageSize = DEFAULT_PAGE_SIZE,
  pageSizeOptions = DEFAULT_PAGE_SIZE_OPTIONS,
  includeHeader = true,
  includeOptions = true,
  extraActions,
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

  // Entity URL State management
  const [entityId, setEntityId] = useQueryState(
    "entityId",
    entityParams.entityId.withOptions({
      startTransition,
      shallow: false,
    }),
  );
  const [modalType, setModalType] = useQueryState(
    "modal",
    entityParams.modal.withOptions({
      startTransition,
      shallow: false,
    }),
  );

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
        throw new Error("Failed to fetch entity");
      }

      return response.json();
    },
    enabled: !!entityId,
    staleTime: 30000, // 30 seconds
  });

  // Update the handleModalClose function to properly clear both parameters
  const handleEditModalClose = useCallback(async () => {
    await Promise.all([setEntityId(null), setModalType(null)]);
  }, [setEntityId, setModalType]);

  const handleCreateModalClose = useCallback(async () => {
    await setModalType(null);
  }, [setModalType]);

  useEffect(() => {
    // Only handle edit modal consistency
    if (entityId && !modalType) {
      setModalType("edit").catch(console.error);
    }

    // Only clear modal if we're in edit mode and lose the entityId
    if (!entityId && modalType === "edit") {
      setModalType(null).catch(console.error);
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

  const {
    rowSelection,
    setRowSelection,
    columnVisibility,
    setColumnVisibility,
    columnFilters,
    setColumnFilters,
    sorting,
    // setSorting,
    // showFilterDialog,
    // setShowFilterDialog,
  } = useDataTableState<TData>();

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
    data: placeholderData as TData[],
    columns: displayColumns,
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
    defaultColumn: {
      size: 200,
      minSize: 10,
      maxSize: 300,
    },
    onColumnFiltersChange: setColumnFilters,
    onColumnVisibilityChange: setColumnVisibility,
    getCoreRowModel: getCoreRowModel(),
    getFilteredRowModel: getFilteredRowModel(),
    getPaginationRowModel: getPaginationRowModel(),
    getSortedRowModel: getSortedRowModel(),
    getFacetedRowModel: getFacetedRowModel(),
    getFacetedUniqueValues: getFacetedUniqueValues(),
  });

  const isEditModalOpen = Boolean(entityId && modalType === "edit");
  const isCreateModalOpen = Boolean(modalType === "create");
  const isLoading = dataQuery.isLoading || isTransitioning;
  const isEntityLoading = entityQuery.isLoading;
  const isEntityError = entityQuery.error;

  return (
    <TooltipProvider>
      <div className="mt-2 flex flex-col gap-3">
        {includeOptions && (
          <div className="flex justify-between">
            <div className="flex items-center gap-2">
              <Input
                icon={
                  <Icon
                    icon={faSearch}
                    className="size-3 text-muted-foreground"
                  />
                }
                placeholder="Filter..."
                className="w-full"
              />
            </div>
            <div className="flex items-center gap-2">
              <DataTableViewOptions table={table} />
              <Separator
                className="h-6 w-px bg-border"
                orientation="vertical"
              />
              <DataTableCreateButton
                name={name}
                exportModelName={exportModelName}
                extraActions={extraActions}
                onCreateClick={() => {
                  setModalType("create");
                }}
              />
            </div>
          </div>
        )}
        <div className="rounded-md border border-sidebar-border">
          <Table>
            {includeHeader && <DataTableHeader table={table as any} />}
            <DataTableBody table={table as any} />
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
        {isCreateModalOpen && TableModal && (
          <TableModal
            open={isCreateModalOpen}
            onOpenChange={() => {
              handleCreateModalClose();
            }}
          />
        )}
        {isEditModalOpen && TableEditModal && (
          <TableEditModal
            open={isEditModalOpen}
            onOpenChange={() => {
              handleEditModalClose();
            }}
            currentRecord={(entityQuery.data as TData) || undefined}
            isLoading={isEntityLoading}
            error={isEntityError}
          />
        )}
      </div>
    </TooltipProvider>
  );
}
