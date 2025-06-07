"use no memo";
import { useDataTableQuery } from "@/hooks/use-data-table-query";
import { searchParamsParser } from "@/hooks/use-data-table-state";
import { useLiveDataTable } from "@/hooks/use-live-data-table";
import { usePermissions } from "@/hooks/use-permissions";
import { queries } from "@/lib/queries";
import { DataTableProps } from "@/types/data-table";
import { Action } from "@/types/roles-permissions";
import { useQuery } from "@tanstack/react-query";
import {
  getCoreRowModel,
  getPaginationRowModel,
  RowSelectionState,
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
import { LiveModeBanner } from "./_components/live-mode-banner";
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
  liveMode,
}: DataTableProps<TData>) {
  const [searchParams, setSearchParams] = useQueryStates(searchParamsParser);
  const { page, pageSize, entityId, modalType } = searchParams;
  const [rowSelection, setRowSelection] = useState<RowSelectionState>(
    entityId ? { [entityId]: true } : {},
  );
  const { can } = usePermissions();
  const [columnVisibility, setColumnVisibility] =
    useLocalStorage<VisibilityState>(
      `${resource.toLowerCase()}-column-visibility`,
      {},
    );

  // Live mode state management
  const [liveModeEnabled, setLiveModeEnabled] = useLocalStorage(
    `${resource.toLowerCase()}-live-mode-enabled`,
    liveMode?.enabled || false,
  );
  const [autoRefreshEnabled, setAutoRefreshEnabled] = useLocalStorage(
    `${resource.toLowerCase()}-auto-refresh-enabled`,
    liveMode?.autoRefresh || false,
  );

  // Fetch persisted table configuration from the server
  const { data: tableConfig } = useQuery({
    ...queries.tableConfiguration.get(resource),
  });

  // On first successful fetch, hydrate the local column visibility if there is
  // nothing stored locally yet.
  useEffect(() => {
    if (!tableConfig) return;
    // Only overwrite if the current local storage value is empty
    if (Object.keys(columnVisibility || {}).length === 0) {
      setColumnVisibility(
        tableConfig.tableConfig.columnVisibility as VisibilityState,
      );
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [tableConfig]);

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

  // Live mode integration - memoize to prevent unnecessary re-renders
  const liveData = useLiveDataTable({
    queryKey,
    endpoint: liveMode?.endpoint || "",
    enabled: liveModeEnabled && !!liveMode?.endpoint,
    autoRefresh: autoRefreshEnabled,
  });

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
    meta: {
      getRowClassName: (row: any) => {
        let className = getRowClassName?.(row) || "";

        // Add new item highlighting
        if (liveMode && liveData.isNewItem?.(row.id)) {
          className += " animate-new-item";
        }

        return className;
      },
    },
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

  // Memoize modal props to prevent unnecessary re-renders
  const editModalProps = useMemo(
    () => ({
      isLoading: dataQuery.isFetching || dataQuery.isLoading,
      currentRecord: selectedRow?.original,
      error: dataQuery.error,
    }),
    [
      dataQuery.isFetching,
      dataQuery.isLoading,
      selectedRow?.original,
      dataQuery.error,
    ],
  );

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
                  name={name}
                  resource={resource}
                  exportModelName={exportModelName}
                  extraActions={extraActions}
                  handleCreateClick={handleCreateClick}
                  liveModeConfig={liveMode}
                  liveModeEnabled={liveModeEnabled}
                  onLiveModeToggle={setLiveModeEnabled}
                />
              </Suspense>
            </DataTableOptions>
          )}

          {liveMode && !autoRefreshEnabled && (
            <LiveModeBanner
              show={liveData.showNewItemsBanner}
              newItemsCount={liveData.newItemsCount}
              connected={liveData.connected}
              onRefresh={liveData.refreshData}
              onDismiss={liveData.dismissBanner}
            />
          )}

          <Table className="rounded-md border-x border-border border-separate border-spacing-0">
            {includeHeader && <DataTableHeader table={table} />}
            <DataTableBody
              table={table}
              columns={columns}
              liveMode={
                liveMode && {
                  enabled: liveModeEnabled,
                  connected: liveData.connected,
                  showToggle: liveMode.showToggle,
                  onToggle: setLiveModeEnabled,
                  autoRefresh: autoRefreshEnabled,
                  onAutoRefreshToggle: setAutoRefreshEnabled,
                }
              }
            />
          </Table>
          <PaginationInner table={table} />
          {TableModal && isCreateModalOpen && (
            <TableModal
              open={isCreateModalOpen}
              onOpenChange={handleCreateModalClose}
            />
          )}
          {TableEditModal && <TableEditModal {...editModalProps} />}
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
