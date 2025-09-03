/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

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
import { useLocalStorage } from "@/hooks/use-local-storage";
import { usePermissions } from "@/hooks/use-permissions";
import {
  convertFilterStateToAPIParams,
  getDataTableEndpoint,
} from "@/lib/enhanced-data-table-api";
import { filterUtils } from "@/lib/enhanced-data-table-utils";
import { queries } from "@/lib/queries";
import type { FilterStateSchema } from "@/lib/schemas/table-configuration-schema";
import { DataTableProps } from "@/types/data-table";
import type {
  EnhancedColumnDef,
  EnhancedDataTableConfig,
} from "@/types/enhanced-data-table";
import { Action } from "@/types/roles-permissions";
import type { API_ENDPOINTS } from "@/types/server";
import { useQuery } from "@tanstack/react-query";
import {
  getCoreRowModel,
  getPaginationRowModel,
  Row,
  useReactTable,
  VisibilityState,
} from "@tanstack/react-table";
import { useQueryStates } from "nuqs";
import React, { lazy, useCallback, useEffect, useMemo } from "react";
import { toast } from "sonner";
import { LazyLoader } from "../error-boundary";
import LetterGlitch from "../ui/letter-glitch";
import { DataTablePermissionDeniedSkeleton } from "../ui/permission-skeletons";
import { Table } from "../ui/table";
import { DataTableActionsSkeleton } from "./_components/data-table-actions";
import { DataTableBody } from "./_components/data-table-body";
import { DataTableHeader } from "./_components/data-table-header";
import { DataTableOptions } from "./_components/data-table-options";
import { PaginationInner } from "./_components/data-table-pagination";
import { EnhancedDataTableFilters } from "./_components/enhanced-data-table-filters";
import { EnhancedDataTableSearch } from "./_components/enhanced-data-table-search";
import { EnhancedDataTableSort } from "./_components/enhanced-data-table-sort";
import { LiveModeBanner } from "./_components/live-mode-banner";
import { DataTableProvider } from "./data-table-provider";

const DataTableActions = lazy(() => import("./_components/data-table-actions"));

export interface EnhancedDataTableProps<TData extends Record<string, any>>
  extends Omit<DataTableProps<TData>, "columns"> {
  columns: EnhancedColumnDef<TData>[];
  config?: EnhancedDataTableConfig;
  defaultFilters?: FilterStateSchema["filters"];
  defaultSort?: FilterStateSchema["sort"];
  onFilterChange?: (state: FilterStateSchema) => void;
  useEnhancedBackend?: boolean;
}

const defaultConfig: EnhancedDataTableConfig = {
  enableFiltering: true,
  enableSorting: true,
  enableMultiSort: true,
  maxFilters: 10,
  maxSorts: 3,
  searchDebounce: 300,
  showFilterUI: true,
  showSortUI: true,
  enableFilterPresets: false,
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
  useEnhancedBackend = false,
  contextMenuActions,
}: EnhancedDataTableProps<TData>) {
  const [searchParams, setSearchParams] = useQueryStates(searchParamsParser);
  const { page, pageSize, entityId, modalType } = searchParams;

  // Derive rowSelection from entityId - URL is the single source of truth
  const rowSelection = React.useMemo(
    () => (entityId ? { [entityId]: true } : {}),
    [entityId],
  );

  // Initialize column order with empty array like infinite table
  const [columnOrder, setColumnOrder] = useLocalStorage<string[]>(
    `trenova-${resource.toLowerCase()}-column-order`,
    [],
  );

  const { can } = usePermissions();
  const [columnVisibility, setColumnVisibility] =
    useLocalStorage<VisibilityState>(
      `trenova-${resource.toLowerCase()}-column-visibility`,
      {},
    );
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

  // Derive filter state from URL parameters
  const filterState = useMemo<FilterStateSchema>(() => {
    const deserialized = filterUtils.deserializeFromURL({
      query: searchParams.query || "",
      filters: searchParams.filters || "",
      sort: searchParams.sort || "",
    });

    // Check if this is the very first load (no URL params at all)
    // vs. user has interacted and explicitly set empty state
    const isFirstLoad =
      !searchParams.query && !searchParams.filters && !searchParams.sort;

    const result = {
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

    return result;
  }, [
    searchParams.query,
    searchParams.filters,
    searchParams.sort,
    defaultFilters,
    defaultSort,
  ]);

  // Convert filter state to API parameters
  const enhancedSearchParams = useMemo(() => {
    const apiParams = convertFilterStateToAPIParams(filterState, {
      useLegacyMode: !useEnhancedBackend,
      additionalParams: extraSearchParams || {},
    });

    // Don't add timestamp - it causes infinite loops!
    return apiParams;
  }, [filterState, extraSearchParams, useEnhancedBackend]);

  // Get the appropriate endpoint
  const apiEndpoint = useMemo((): API_ENDPOINTS => {
    if (useEnhancedBackend) {
      return getDataTableEndpoint(resource, true);
    }
    return link;
  }, [resource, link, useEnhancedBackend]);

  // Live mode state management
  const [liveModeEnabled, setLiveModeEnabled] = useLocalStorage(
    `trenova-${resource.toLowerCase()}-live-mode-enabled`,
    liveMode?.enabled || false,
  );
  const [autoRefreshEnabled, setAutoRefreshEnabled] = useLocalStorage(
    `trenova-${resource.toLowerCase()}-auto-refresh-enabled`,
    liveMode?.autoRefresh || false,
  );

  // Fetch persisted table configuration from the server
  const {
    data: tableConfig,
    isLoading: isTableConfigLoading,
    isError,
  } = useQuery({
    ...queries.tableConfiguration.getDefaultOrLatestConfiguration(resource),
  });

  // On first successful fetch, hydrate the local column visibility
  useEffect(() => {
    if (isError) {
      toast.error("Unable to fetch table configuration", {
        description: "Please try again later or contact support",
      });
      return;
    }

    // Don't do anything while still loading
    if (isTableConfigLoading) return;

    // * Check if there is no table configuration only after the query is done loading
    if (!tableConfig) {
      return;
    }

    // * Set column visibility from table configuration
    if (tableConfig.tableConfig?.columnVisibility) {
      setColumnVisibility(
        tableConfig.tableConfig.columnVisibility as VisibilityState,
      );
    }

    // * Set column order from table configuration
    if (tableConfig.tableConfig?.columnOrder) {
      setColumnOrder(tableConfig.tableConfig.columnOrder);
    }

    handleFilterChange({
      globalSearch: "",
      filters: tableConfig.tableConfig.filters || [],
      sort: tableConfig.tableConfig.sort || [],
    });

    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [tableConfig, isTableConfigLoading, isError, setColumnOrder]);

  // Derive pagination state from URL
  const pagination = useMemo(
    () => ({
      pageIndex: (page ?? 1) - 1,
      pageSize: pageSize ?? initialPageSize,
    }),
    [page, pageSize, initialPageSize],
  );

  // Use the data table query with enhanced parameters
  const dataQuery = useDataTableQuery<TData>(
    queryKey,
    apiEndpoint as API_ENDPOINTS,
    pagination,
    enhancedSearchParams,
  );

  // Live mode integration with performance optimization
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
      // When row selection changes, update the URL
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

        // Add new item highlighting
        if (liveModeEnabled && liveData.isNewItem?.(row.id)) {
          className += " animate-new-item";
        }

        return className;
      },
    },
  });

  /**
   * Calculate column sizes as CSS variables for performance optimization
   * This approach from infinite table prevents expensive column.getSize() calls on every render
   */
  const tableState = table.getState();
  const columnSizeVars = useMemo(() => {
    const headers = table.getFlatHeaders();
    const colSizes: { [key: string]: string } = {};
    for (let i = 0; i < headers.length; i++) {
      const header = headers[i]!;
      // Replace "." with "-" to avoid invalid CSS variable names
      colSizes[`--header-${header.id.replace(".", "-")}-size`] =
        `${header.getSize()}px`;
      colSizes[`--col-${header.column.id.replace(".", "-")}-size`] =
        `${header.column.getSize()}px`;
    }
    return colSizes;
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [
    tableState.columnSizingInfo,
    tableState.columnSizing,
    tableState.columnVisibility,
  ]);

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
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [
    rowSelection,
    dataQuery.isLoading,
    dataQuery.isFetching,
    dataQuery.data?.results,
  ]);

  // No longer need effects to sync rowSelection with entityId
  // The rowSelection is now derived directly from entityId

  const handleFilterChange = useCallback(
    (newFilterState: FilterStateSchema) => {
      const urlParams = filterUtils.serializeToURL(newFilterState);

      setSearchParams({
        ...searchParams,
        query: typeof urlParams.query === "string" ? urlParams.query : null,
        filters:
          typeof urlParams.filters === "string" ? urlParams.filters : null,
        sort: typeof urlParams.sort === "string" ? urlParams.sort : null,
        page: 1,
      });

      onFilterChange?.(newFilterState);
    },
    [searchParams, setSearchParams, onFilterChange],
  );

  const handleCreateClick = useCallback(() => {
    setSearchParams({ modalType: "create", entityId: null });
  }, [setSearchParams]);

  const handleCreateModalClose = useCallback(() => {
    setSearchParams({ modalType: null, entityId: null });
  }, [setSearchParams]);

  const isCreateModalOpen = Boolean(modalType === "create");

  // Keyboard shortcut to reset column visibility and order (like infinite table)
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === "u" && (e.metaKey || e.ctrlKey)) {
        e.preventDefault();
        setColumnOrder([]);
        setColumnVisibility({});
      }
    };

    document.addEventListener("keydown", handleKeyDown);
    return () => document.removeEventListener("keydown", handleKeyDown);
  }, [setColumnOrder, setColumnVisibility]);

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
        className="flex flex-col gap-2 size-full"
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
              <DataTableOptions>
                <div className="flex flex-col size-full">
                  <div className="flex justify-between items-center">
                    {(config.showFilterUI || config.showSortUI) && (
                      <div className="flex flex-col lg:flex-row gap-2">
                        <EnhancedDataTableSearch
                          filterState={filterState}
                          onFilterChange={handleFilterChange}
                          placeholder={`Search ${name.toLowerCase()}...`}
                        />
                        {config.showFilterUI && (
                          <div className="flex-1">
                            <EnhancedDataTableFilters
                              columns={columns}
                              filterState={filterState}
                              onFilterChange={handleFilterChange}
                              config={config}
                            />
                          </div>
                        )}
                        {config.showSortUI && (
                          <div className="w-full lg:w-auto">
                            <EnhancedDataTableSort
                              columns={columns}
                              sortState={filterState.sort}
                              onSortChange={(newSort) => {
                                const newFilterState = {
                                  ...filterState,
                                  sort: newSort,
                                };
                                handleFilterChange(newFilterState);
                              }}
                              config={config}
                            />
                          </div>
                        )}
                      </div>
                    )}
                    <LazyLoader fallback={<DataTableActionsSkeleton />}>
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
                    </LazyLoader>
                  </div>
                </div>
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
            {!dataQuery.isLoading && dataQuery.data?.count === 0 ? (
              <div className="flex flex-col items-center justify-center max-h-[300px] border border-border rounded-md p-0.5 overflow-hidden">
                <div className="relative size-full">
                  <LetterGlitch
                    glitchColors={["#9c9c9c", "#696969", "#424242"]}
                    glitchSpeed={50}
                    centerVignette={false}
                    outerVignette={true}
                    smooth={true}
                  />
                  <div className="absolute inset-0 flex flex-col gap-1 items-center justify-center pointer-events-none">
                    <p className="text-sm/none px-1 py-0.5 text-center font-medium uppercase select-none font-table dark:text-neutral-900 bg-amber-300 text-amber-950 dark:bg-amber-400">
                      No data available
                    </p>
                    <p className="text-sm/none px-1 py-0.5 text-center font-medium uppercase select-none font-table dark:text-neutral-900 bg-neutral-900 text-white dark:bg-neutral-500">
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
            {TableEditModal && (
              <TableEditModal
                isLoading={dataQuery.isFetching || dataQuery.isLoading}
                currentRecord={selectedRow?.original}
                error={dataQuery.error}
                apiEndpoint={apiEndpoint as API_ENDPOINTS}
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
