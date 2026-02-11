/**
 * ACKNOWLEDGMENTS
 *
 * This Table component incorporates design patterns and architectural concepts
 * inspired by the following open-source projects:
 *
 * - SHADCN Table: https://github.com/sadmann7/shadcn-table
 * - OpenStatus Data Table: https://github.com/openstatusHQ/data-table-filters
 *
 * While the implementation is original, we acknowledge the foundational work
 * and innovative approaches demonstrated by these projects in the React table
 * ecosystem.
 */

"use no memo";
import { useDataTableQuery } from "@/hooks/use-data-table-query";
import { searchParamsParser } from "@/hooks/use-data-table-state";
import { useLiveDataTable } from "@/hooks/use-live-data-table";
import { usePermissions } from "@/hooks/use-permission";
import { convertFilterStateToAPIParams } from "@/lib/data-table-api";
import { filterUtils } from "@/lib/data-table-utils";
import { queries } from "@/lib/queries";
import type { FilterStateSchema } from "@/lib/schemas/table-configuration-schema";
import { Config, DataTableProps, EnhancedColumnDef } from "@/types/data-table";
import { Action } from "@/types/roles-permissions";
import { useQuery } from "@tanstack/react-query";
import {
  getCoreRowModel,
  getPaginationRowModel,
  Row,
  useReactTable,
  VisibilityState,
} from "@tanstack/react-table";
import { useQueryStates } from "nuqs";
import React from "react";
import LetterGlitch from "../ui/letter-glitch";
import { DataTablePermissionDeniedSkeleton } from "../ui/permission-skeletons";
import { Table } from "../ui/table";
import { DataTableOptions } from "./_components/_actions/data-table-options";
import { DataTableBody } from "./_components/data-table-body";
import { DataTableHeader } from "./_components/data-table-header";
import { PaginationInner } from "./_components/data-table-pagination";
import { LiveModeBanner } from "./_components/live-mode-banner";
import { DataTableProvider } from "./data-table-provider";

export interface EnhancedDataTableProps<TData extends Record<string, any>>
  extends Omit<DataTableProps<TData>, "columns"> {
  columns: EnhancedColumnDef<TData>[];
  config?: Config;
  defaultFilters?: FilterStateSchema["filters"];
  defaultSort?: FilterStateSchema["sort"];
  onFilterChange?: (state: FilterStateSchema) => void;
  useEnhancedBackend?: boolean;
}

const defaultConfig: Config = {
  enableFiltering: true,
  enableSorting: true,
  enableMultiSort: true,
  maxFilters: 10,
  maxSorts: 3,
  searchDebounce: 300,
  showFilterUI: true,
  showSortUI: true,
};

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
  config = defaultConfig,
  defaultFilters = [],
  defaultSort = [],
  onFilterChange,
  // useEnhancedBackend = false,
  contextMenuActions,
}: EnhancedDataTableProps<TData>) {
  const [searchParams, setSearchParams] = useQueryStates(searchParamsParser);
  const { page, pageSize, entityId, modalType } = searchParams;

  const rowSelection = React.useMemo(
    () => (entityId ? { [entityId]: true } : {}),
    [entityId],
  );

  const [columnOrder, setColumnOrder] = React.useState<string[]>([]);
  const [columnVisibility, setColumnVisibility] =
    React.useState<VisibilityState>({});

  const { can } = usePermissions();
  const topBarRef = React.useRef<HTMLDivElement>(null);
  const [topBarHeight, setTopBarHeight] = React.useState(0);

  React.useEffect(() => {
    const observer = new ResizeObserver(() => {
      const rect = topBarRef.current?.getBoundingClientRect();
      if (rect) {
        setTopBarHeight(rect.height);
      }
    });

    const topBar = topBarRef.current;
    if (!topBar) return;

    observer.observe(topBar);
    return () => observer.unobserve(topBar);
  }, [topBarRef]);

  const filterState = React.useMemo<FilterStateSchema>(() => {
    const deserialized = filterUtils.deserializeFromURL({
      query: searchParams.query || "",
      filters: searchParams.filters || "",
      sort: searchParams.sort || "",
    });

    const isFirstLoad =
      !searchParams.query && !searchParams.filters && !searchParams.sort;

    return {
      globalSearch: deserialized.globalSearch || "",
      filters:
        deserialized.filters.length > 0
          ? deserialized.filters
          : isFirstLoad
            ? defaultFilters
            : [],
      sort:
        deserialized.sort.length > 0
          ? deserialized.sort
          : isFirstLoad
            ? defaultSort
            : [],
    };
  }, [
    searchParams.query,
    searchParams.filters,
    searchParams.sort,
    defaultFilters,
    defaultSort,
  ]);

  const parsedSearchParams = React.useMemo(
    () =>
      convertFilterStateToAPIParams(filterState, {
        additionalParams: extraSearchParams || {},
      }),
    [filterState, extraSearchParams],
  );

  const [liveModeEnabled, setLiveModeEnabled] = React.useState(
    liveMode?.enabled || false,
  );
  const [autoRefreshEnabled, setAutoRefreshEnabled] = React.useState(
    liveMode?.autoRefresh || false,
  );

  const {
    data: tableConfig,
    isLoading: isTableConfigLoading,
    isError,
  } = useQuery({
    ...queries.tableConfiguration.getDefaultOrLatestConfiguration(resource),
  });

  React.useEffect(() => {
    if (isError || isTableConfigLoading || !tableConfig) {
      return;
    }

    if (tableConfig.tableConfig?.columnVisibility) {
      setColumnVisibility(
        tableConfig.tableConfig.columnVisibility as VisibilityState,
      );
    }

    if (tableConfig.tableConfig?.columnOrder) {
      setColumnOrder(tableConfig.tableConfig.columnOrder);
    }

    const hasUrlFilters =
      searchParams.filters || searchParams.sort || searchParams.query;
    if (!hasUrlFilters && tableConfig.tableConfig) {
      handleFilterChange({
        globalSearch: "",
        filters: tableConfig.tableConfig.filters || [],
        sort: tableConfig.tableConfig.sort || [],
      });
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [tableConfig, isTableConfigLoading, isError]);

  const pagination = React.useMemo(
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
    parsedSearchParams,
  );

  const liveData = useLiveDataTable({
    queryKey,
    endpoint: liveMode?.endpoint || "",
    enabled: liveModeEnabled && !!liveMode?.endpoint,
    autoRefresh: autoRefreshEnabled,
    batchWindow: liveMode?.options?.batchWindow || 100,
    debounceDelay: liveMode?.options?.debounceDelay || 300,
    onNewData: liveMode?.options?.onNewData,
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
      columnOrder,
      filters: filterState,
    },
    onColumnVisibilityChange: setColumnVisibility,
    onColumnOrderChange: setColumnOrder,
    enableMultiRowSelection: false,
    columnResizeMode: "onChange",
    manualPagination: true,
    enableRowSelection: true,
    getRowId: (row) => row.id,
    onRowSelectionChange: (updater) => {
      const newSelection =
        typeof updater === "function" ? updater(rowSelection) : updater;
      const selectedId = Object.keys(newSelection)[0];

      if (selectedId) {
        setSearchParams({ entityId: selectedId, modalType: "edit" });
      } else {
        setSearchParams({ entityId: null, modalType: null });
      }
    },
    getCoreRowModel: getCoreRowModel(),
    getPaginationRowModel: getPaginationRowModel(),
    meta: {
      getRowClassName: (row: Row<TData>) => {
        let className = getRowClassName?.(row) || "";

        if (liveModeEnabled && liveData.isNewItem?.(row.id)) {
          className += " animate-new-item";
        }

        return className;
      },
    },
  });

  const tableState = table.getState();
  const columnSizeVars = React.useMemo(() => {
    const headers = table.getFlatHeaders();
    const colSizes: { [key: string]: string } = {};

    for (const header of headers) {
      const sanitizedId = header.id.replace(".", "-");
      colSizes[`--header-${sanitizedId}-size`] = `${header.getSize()}px`;
      colSizes[`--col-${header.column.id.replace(".", "-")}-size`] =
        `${header.column.getSize()}px`;
    }

    return colSizes;
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [
    table,
    tableState.columnSizingInfo,
    tableState.columnSizing,
    tableState.columnVisibility,
  ]);

  const selectedRow = React.useMemo(() => {
    if (
      (dataQuery.isLoading || dataQuery.isFetching) &&
      !dataQuery.data?.results.length
    ) {
      return undefined;
    }

    const selectedRowKey = Object.keys(rowSelection)[0];
    if (!selectedRowKey) return undefined;

    return table
      .getCoreRowModel()
      .flatRows.find((row) => row.id === selectedRowKey);
  }, [
    rowSelection,
    dataQuery.isLoading,
    dataQuery.isFetching,
    dataQuery.data,
    table,
  ]);

  const handleFilterChange = React.useCallback(
    (newFilterState: FilterStateSchema) => {
      const urlParams = filterUtils.serializeToURL(newFilterState);

      setSearchParams({
        query: typeof urlParams.query === "string" ? urlParams.query : null,
        filters:
          typeof urlParams.filters === "string" ? urlParams.filters : null,
        sort: typeof urlParams.sort === "string" ? urlParams.sort : null,
        page: 1,
      });

      onFilterChange?.(newFilterState);
    },
    [setSearchParams, onFilterChange],
  );

  const isCreateModalOpen = Boolean(modalType === "create");

  React.useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === "u" && (e.metaKey || e.ctrlKey)) {
        e.preventDefault();
        setColumnOrder([]);
        setColumnVisibility({});
      }
    };

    document.addEventListener("keydown", handleKeyDown);
    return () => document.removeEventListener("keydown", handleKeyDown);
  }, []);

  const handleCreateModalClose = React.useCallback(() => {
    setSearchParams({ modalType: null, entityId: null });
  }, [setSearchParams]);

  return (
    <DataTableProvider
      table={table}
      columns={columns}
      isLoading={dataQuery.isFetching || dataQuery.isLoading}
      pagination={pagination}
      rowSelection={rowSelection}
      columnVisibility={columnVisibility}
      columnOrder={columnOrder}
    >
      <div
        className="flex size-full flex-col gap-2"
        style={
          {
            ...columnSizeVars,
            "--top-bar-height": `${topBarHeight}px`,
          } as React.CSSProperties
        }
      >
        {can(resource, Action.Read) ? (
          <>
            {includeOptions && (
              <DataTableOptions
                config={config}
                handleFilterChange={handleFilterChange}
                setSearchParams={setSearchParams}
                filterState={filterState}
                columns={columns}
                name={name}
                resource={resource}
                exportModelName={exportModelName}
                extraActions={extraActions}
                liveMode={liveMode}
                liveModeEnabled={liveModeEnabled}
                setLiveModeEnabled={setLiveModeEnabled}
              />
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
            {!dataQuery.isLoading && dataQuery.data?.count === 0 ? (
              <div className="flex max-h-[300px] flex-col items-center justify-center overflow-hidden rounded-md border border-border p-0.5">
                <div className="relative size-full">
                  <LetterGlitch
                    glitchColors={["#9c9c9c", "#696969", "#424242"]}
                    glitchSpeed={50}
                    centerVignette={true}
                    outerVignette={false}
                    smooth={true}
                  />
                  <div className="pointer-events-none absolute inset-0 flex flex-col items-center justify-center gap-1">
                    <p className="bg-amber-300 px-1 py-0.5 text-center font-table text-sm/none font-medium text-amber-950 uppercase select-none dark:bg-amber-400 dark:text-neutral-900">
                      No data available
                    </p>
                    <p className="bg-neutral-900 px-1 py-0.5 text-center font-table text-sm/none font-medium text-white uppercase select-none dark:bg-neutral-500 dark:text-neutral-900">
                      Try adjusting your filters or search query
                    </p>
                  </div>
                </div>
              </div>
            ) : (
              <Table
                className="border-separate border-spacing-0"
                containerClassName="max-h-[calc(65vh_-_var(--top-bar-height))] border border-border rounded-md"
              >
                {includeHeader && <DataTableHeader table={table} />}
                <DataTableBody
                  isLoading={true}
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
                  contextMenuActions={contextMenuActions}
                />
              </Table>
            )}

            <PaginationInner table={table} />
            {TableModal && isCreateModalOpen && (
              <TableModal
                open={isCreateModalOpen}
                onOpenChange={handleCreateModalClose}
              />
            )}
            {TableEditModal && selectedRow?.original && (
              <TableEditModal
                isLoading={dataQuery.isFetching || dataQuery.isLoading}
                currentRecord={selectedRow?.original}
                error={dataQuery.error}
                apiEndpoint={link}
                queryKey={queryKey}
              />
            )}
          </>
        ) : (
          <DataTablePermissionDeniedSkeleton
            resource={resource}
            action={Action.Read}
          />
        )}
      </div>
    </DataTableProvider>
  );
}
