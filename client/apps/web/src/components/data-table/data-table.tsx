"use no memo";
import { DataTableProvider } from "@/contexts/data-table-context";
import { useDataTableLiveRefresh } from "@/hooks/data-table/use-data-table-live-refresh";
import { useDataTableQuery } from "@/hooks/data-table/use-data-table-query";
import { searchParamsParser } from "@/hooks/data-table/use-data-table-state";
import { useDebounce } from "@/hooks/use-debounce";
import { usePermissions } from "@/hooks/use-permission";
import {
  columnPinOffsetVar,
  columnSizeVar,
  compileFormatRules,
  initializeFilterItemsFromFieldFilters,
  initializeFilterItemsFromFilterGroups,
  isTableConfigEqual,
  updateSortField,
} from "@/lib/data-table";
import { fetchAllRows } from "@/lib/data-table-export";
import { queries } from "@/lib/queries";
import { cn } from "@trenova/shared/lib/utils";
import type {
  DataTableProps,
  FilterGroupItem,
  FilterItem,
  PanelMode,
  SingleFilterItem,
  SortDirection,
  SortField,
} from "@trenova/shared/types/data-table";
import type {
  ActiveTableView,
  TableConfig,
  TableConfiguration,
  TableDensity,
  TableFormatRule,
  TableViewSource,
} from "@/types/table-configuration";
import {
  closestCenter,
  DndContext,
  PointerSensor,
  useSensor,
  useSensors,
  type DragEndEvent,
} from "@dnd-kit/core";
import { restrictToHorizontalAxis } from "@dnd-kit/modifiers";
import { arrayMove, horizontalListSortingStrategy, SortableContext } from "@dnd-kit/sortable";
import { useQuery } from "@tanstack/react-query";
import {
  getCoreRowModel,
  getFilteredRowModel,
  getPaginationRowModel,
  getSortedRowModel,
  useReactTable,
  type Row,
  type RowSelectionState,
} from "@tanstack/react-table";
import { useQueryStates } from "nuqs";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { toast } from "sonner";
import { Table, TableHeader, TableRow } from "@trenova/shared/components/ui/table";
import { DataTablePagination } from "./_components/data-table-pagination";
import { DataTableBody } from "./data-table-body";
import { DataTableDock } from "./data-table-dock";
import DataTableFilterChips from "./data-table-filter-chips";
import { DataTableHeaderCell } from "./data-table-header-cell";
import { DataTablePanelContent, DataTablePanelWrapper } from "./data-table-panel";
import { DataTableRefreshPill } from "./data-table-refresh-pill";
import { DataTableSelectionBanner } from "./data-table-selection-banner";
import { createSelectionColumn } from "./data-table-selection-column";
import { DataTableToolbar } from "./data-table-toolbar";

const BULK_SELECT_MAX = 1000;
const EMPTY_PINNING = { left: [] as string[], right: [] as string[] };

export function DataTable<TData extends Record<string, any>>({
  columns,
  name,
  queryKey,
  resource,
  enableRowSelection = false,
  dockActions = [],
  TablePanel,
  onAddRecord: onAddRecordProp,
  addRecordActions = [],
  contextMenuActions,
  onRowClick,
  enableCreateAction = true,
  enableReadOnlyPanel = false,
  initialColumnVisibility,
  graphql,
  refetchIntervalMs,
}: DataTableProps<TData>) {
  "use no memo";
  const permissions = usePermissions(resource ?? "");
  const canCreate = resource ? permissions.canCreate : true;
  const canUpdate = resource ? permissions.canUpdate : true;
  const canExport = resource ? permissions.canExport : true;
  const [searchParams, setSearchParams] = useQueryStates(searchParamsParser);
  const { pageIndex, pageSize, query, fieldFilters, filterGroups, sort, panelType, panelEntityId } =
    searchParams;
  const [rowSelection, setRowSelection] = useState<RowSelectionState>({});
  const [cursorState, setCursorState] = useState<{
    scopeKey: string;
    cursors: Record<number, string | null>;
  }>({ scopeKey: "", cursors: { 0: null } });
  const [activeView, setActiveView] = useState<ActiveTableView | null>(null);
  const [density, setDensity] = useState<TableDensity>("comfortable");
  const [formatRules, setFormatRules] = useState<TableFormatRule[]>([]);
  const [isSelectingAll, setIsSelectingAll] = useState(false);
  const defaultConfigAppliedRef = useRef(false);
  const selectedRowsMapRef = useRef(new Map<string, TData>());

  const { data: defaultConfig } = useQuery({
    ...queries.tableConfiguration.default(name),
    enabled: !!name,
    retry: false,
    staleTime: Infinity,
  });

  const hasPanel = !!TablePanel;
  const isPanelOpen = !!panelType;
  const panelMode: PanelMode = panelType ?? "create";

  const openPanelCreate = useCallback(() => {
    void setSearchParams({ panelType: "create", panelEntityId: null });
  }, [setSearchParams]);

  const resolvedAddRecordActions = useMemo(() => {
    const actions = [...addRecordActions];
    const defaultOnClick = onAddRecordProp ?? (hasPanel ? openPanelCreate : undefined);

    if (defaultOnClick && !actions.some((action) => action.id === "default-create")) {
      actions.unshift({
        id: "default-create",
        label: `Add ${name}`,
        description: `Create a new ${name.toLowerCase()} from scratch.`,
        onClick: defaultOnClick,
      });
    }

    return enableCreateAction && canCreate ? actions : [];
  }, [
    addRecordActions,
    canCreate,
    enableCreateAction,
    hasPanel,
    name,
    onAddRecordProp,
    openPanelCreate,
  ]);

  const openPanelEdit = useCallback(
    (row: Row<TData>) => {
      const entityId = (row.original as { id?: string }).id;
      if (entityId) {
        void setSearchParams({ panelType: "edit", panelEntityId: entityId });
      }
    },
    [setSearchParams],
  );

  const closePanel = useCallback(() => {
    void setSearchParams({ panelType: null, panelEntityId: null });
  }, [setSearchParams]);

  const handlePanelOpenChange = (open: boolean) => {
    if (!open) {
      closePanel();
    }
  };

  const tableColumns = useMemo(() => {
    if (enableRowSelection) {
      return [createSelectionColumn<TData>(), ...columns];
    }
    return columns;
  }, [columns, enableRowSelection]);

  const [filterItems, setFilterItems] = useState<FilterItem[]>(() => [
    ...initializeFilterItemsFromFieldFilters(fieldFilters ?? [], columns),
    ...initializeFilterItemsFromFilterGroups(
      (filterGroups ?? []).filter((g) => g.filters?.length > 0),
      columns,
    ),
  ]);

  const debouncedFilters = useDebounce(filterItems, 300);
  const urlFiltersRef = useRef({ fieldFilters, filterGroups });
  urlFiltersRef.current = { fieldFilters, filterGroups };

  useEffect(() => {
    const singles = debouncedFilters.filter((f) => f.type === "filter") as SingleFilterItem[];
    const groups = debouncedFilters.filter((f) => f.type === "group") as FilterGroupItem[];

    const newFieldFilters = singles.map((f) => ({
      field: f.apiField,
      operator: f.operator,
      value: f.value,
    }));
    const newFilterGroups = groups.map((g) => ({
      filters: g.items.map((i) => ({
        field: i.apiField,
        operator: i.operator,
        value: i.value,
      })),
    }));

    const { fieldFilters: urlFieldFilters, filterGroups: urlFilterGroups } = urlFiltersRef.current;
    if (
      JSON.stringify(newFieldFilters) === JSON.stringify(urlFieldFilters) &&
      JSON.stringify(newFilterGroups) === JSON.stringify(urlFilterGroups)
    ) {
      return;
    }

    void setSearchParams({
      fieldFilters: newFieldFilters,
      filterGroups: newFilterGroups,
      pageIndex: 1,
    });
  }, [debouncedFilters, setSearchParams]);

  const handleFiltersChange = useCallback((items: FilterItem[]) => {
    setFilterItems(items);
  }, []);

  const handleSearchChange = useCallback(
    (newQuery: string) => {
      void setSearchParams({ query: newQuery, pageIndex: 1 });
    },
    [setSearchParams],
  );

  const handleSortChange = useCallback(
    (field: string, direction: SortDirection | null) => {
      const nextParams: { sort: SortField[]; pageIndex?: number } = {
        sort: updateSortField(sort, field, direction),
        pageIndex: 1,
      };

      void setSearchParams(nextParams);
    },
    [sort, setSearchParams],
  );

  const handleSortArrayChange = useCallback(
    (newSort: SortField[]) => {
      const nextParams: { sort: SortField[]; pageIndex?: number } = {
        sort: newSort,
        pageIndex: 1,
      };

      void setSearchParams(nextParams);
    },
    [setSearchParams],
  );

  const handlePageChange = useCallback(
    (newPageIndex: number) => {
      void setSearchParams({ pageIndex: newPageIndex + 1 });
    },
    [setSearchParams],
  );

  const handlePageSizeChange = useCallback(
    (newPageSize: number) => {
      void setSearchParams({ pageSize: newPageSize, pageIndex: 1 });
    },
    [setSearchParams],
  );

  const zeroBasedPageIndex = pageIndex - 1;
  const effectiveSort = sort;
  const cursorScopeKey = useMemo(
    () =>
      JSON.stringify({
        pageSize,
        query,
        fieldFilters,
        filterGroups,
        sort: effectiveSort,
        graphql: {
          connectionKey: graphql.connectionKey,
          operationName: graphql.operationName,
          extraVariables: graphql.extraVariables ?? null,
        },
      }),
    [
      effectiveSort,
      fieldFilters,
      filterGroups,
      graphql.connectionKey,
      graphql.extraVariables,
      graphql.operationName,
      pageSize,
      query,
    ],
  );
  const pageCursors = cursorState.scopeKey === cursorScopeKey ? cursorState.cursors : { 0: null };
  const currentCursor = pageCursors[zeroBasedPageIndex];
  const canFetchPage = zeroBasedPageIndex === 0 || currentCursor !== undefined;

  const baseQueryOptions = useMemo(
    () => ({
      query,
      fieldFilters,
      filterGroups,
      sort: effectiveSort,
    }),
    [query, fieldFilters, filterGroups, effectiveSort],
  );

  const queryOptions = useMemo(
    () => ({
      ...baseQueryOptions,
      cursor: currentCursor,
    }),
    [baseQueryOptions, currentCursor],
  );

  const pagination = useMemo(
    () => ({ pageIndex: zeroBasedPageIndex, pageSize }),
    [zeroBasedPageIndex, pageSize],
  );

  const dataQuery = useDataTableQuery<TData>(
    queryKey,
    graphql,
    pagination,
    queryOptions,
    canFetchPage,
  );

  const liveRefresh = useDataTableLiveRefresh<TData>({
    intervalMs: refetchIntervalMs,
    enabled: !!refetchIntervalMs && canFetchPage,
    queryKey,
    graphql,
    pagination,
    options: queryOptions,
    currentResults: dataQuery.data?.results,
  });

  useEffect(() => {
    const pageInfo = dataQuery.data?.pageInfo;
    if (pageInfo?.mode !== "cursor" || !pageInfo.hasNextPage || !pageInfo.endCursor) {
      return;
    }

    const nextPageIndex = zeroBasedPageIndex + 1;
    setCursorState((current) => {
      const cursors = current.scopeKey === cursorScopeKey ? current.cursors : { 0: null };
      if (cursors[nextPageIndex] === pageInfo.endCursor) {
        return current;
      }

      return {
        scopeKey: cursorScopeKey,
        cursors: {
          ...cursors,
          [nextPageIndex]: pageInfo.endCursor,
        },
      };
    });
  }, [cursorScopeKey, dataQuery.data?.pageInfo, zeroBasedPageIndex]);

  useEffect(() => {
    if (canFetchPage || pageIndex === 1) {
      return;
    }

    void setSearchParams({ pageIndex: 1 });
  }, [canFetchPage, pageIndex, setSearchParams]);

  const cursorPageInfo = dataQuery.data?.pageInfo ?? null;
  const currentPageResults = dataQuery.data?.results;
  const currentPageRowCount = currentPageResults?.length ?? 0;
  const totalCount = cursorPageInfo
    ? (cursorPageInfo.totalCount ?? null)
    : (dataQuery.data?.count ?? null);
  const rowCount =
    totalCount ??
    (cursorPageInfo
      ? zeroBasedPageIndex * pageSize +
        currentPageRowCount +
        (cursorPageInfo.hasNextPage ? pageSize : 0)
      : 0);
  const pageCount =
    totalCount != null
      ? Math.max(1, Math.ceil(totalCount / pageSize))
      : zeroBasedPageIndex + 1 + (cursorPageInfo?.hasNextPage ? 1 : 0);

  // eslint-disable-next-line react-hooks/incompatible-library
  const table = useReactTable({
    getCoreRowModel: getCoreRowModel(),
    getPaginationRowModel: getPaginationRowModel(),
    getFilteredRowModel: getFilteredRowModel(),
    getSortedRowModel: getSortedRowModel(),
    data: currentPageResults || [],
    columns: tableColumns,
    pageCount,
    rowCount,
    manualPagination: true,
    manualFiltering: true,
    columnResizeMode: "onChange",
    enableColumnPinning: true,
    getRowId: (row) => row.id,
    manualSorting: true,
    enableRowSelection,
    enableMultiRowSelection: true,
    onRowSelectionChange: setRowSelection,
    initialState: initialColumnVisibility
      ? { columnVisibility: initialColumnVisibility }
      : undefined,
    state: {
      pagination,
      rowSelection,
    },
    onPaginationChange: (updater) => {
      const newState =
        typeof updater === "function"
          ? updater({ pageIndex: zeroBasedPageIndex, pageSize })
          : updater;
      handlePageChange(newState.pageIndex);
      if (newState.pageSize !== pageSize) {
        handlePageSizeChange(newState.pageSize);
      }
    },
  });

  useEffect(() => {
    const map = selectedRowsMapRef.current;
    for (const id of map.keys()) {
      if (!rowSelection[id]) map.delete(id);
    }
    if (currentPageResults) {
      for (const row of currentPageResults) {
        const id = (row as { id?: string }).id;
        if (id && rowSelection[id]) map.set(id, row);
      }
    }
  }, [rowSelection, currentPageResults]);

  const selectedCount = useMemo(
    () => Object.values(rowSelection).filter(Boolean).length,
    [rowSelection],
  );

  const getSelectedRows = useCallback(() => Array.from(selectedRowsMapRef.current.values()), []);

  const allPageRowsSelected =
    currentPageRowCount > 0 &&
    (currentPageResults?.every((row) => rowSelection[(row as { id?: string }).id ?? ""]) ?? false);

  const handleSelectAllMatching = useCallback(async () => {
    setIsSelectingAll(true);
    try {
      const rows = await fetchAllRows<TData>({
        graphql,
        options: baseQueryOptions,
        maxRows: BULK_SELECT_MAX,
      });
      const map = selectedRowsMapRef.current;
      const selection: RowSelectionState = {};
      for (const row of rows) {
        const id = (row as { id?: string }).id;
        if (id) {
          selection[id] = true;
          map.set(id, row);
        }
      }
      setRowSelection(selection);
    } catch (error) {
      toast.error("Selection failed", {
        description:
          error instanceof Error ? error.message : "Could not load all matching rows.",
      });
    } finally {
      setIsSelectingAll(false);
    }
  }, [graphql, baseQueryOptions]);

  const handleClearSelection = useCallback(() => {
    setRowSelection({});
  }, []);

  const handleApplyConfig = useCallback(
    (config: TableConfig, source?: TableViewSource) => {
      const newFieldFilters = config.fieldFilters ?? [];
      const newFilterGroups = (config.filterGroups ?? []).filter((g) => g.filters?.length > 0);
      const newSort = config.sort ?? [];
      const newPageSize = config.pageSize ?? pageSize;
      const newColumnVisibility = config.columnVisibility ?? {};
      const newColumnOrder = config.columnOrder ?? [];
      const newColumnSizing = config.columnSizing ?? {};
      const newColumnPinning = config.columnPinning ?? EMPTY_PINNING;
      const newDensity = config.density ?? "comfortable";
      const newFormatRules = config.formatRules ?? [];

      void setSearchParams({
        fieldFilters: newFieldFilters,
        filterGroups: newFilterGroups,
        sort: newSort,
        pageSize: newPageSize,
        pageIndex: 1,
      });

      setFilterItems([
        ...initializeFilterItemsFromFieldFilters(newFieldFilters, columns),
        ...initializeFilterItemsFromFilterGroups(newFilterGroups, columns),
      ]);

      table.setColumnVisibility(newColumnVisibility);
      table.setColumnOrder(newColumnOrder);
      table.setColumnSizing(newColumnSizing);
      table.setColumnPinning({
        left: newColumnPinning.left ?? [],
        right: newColumnPinning.right ?? [],
      });
      setDensity(newDensity);
      setFormatRules(newFormatRules);

      setActiveView(
        source
          ? {
              id: source.id,
              name: source.name,
              config: {
                fieldFilters: newFieldFilters,
                filterGroups: newFilterGroups,
                joinOperator: config.joinOperator ?? "and",
                sort: newSort,
                pageSize: newPageSize,
                columnVisibility: newColumnVisibility,
                columnOrder: newColumnOrder,
                columnSizing: newColumnSizing,
                columnPinning: {
                  left: newColumnPinning.left ?? [],
                  right: newColumnPinning.right ?? [],
                },
                density: newDensity,
                formatRules: newFormatRules,
              },
            }
          : null,
      );
    },
    [setSearchParams, columns, pageSize, table],
  );

  useEffect(() => {
    if (
      defaultConfig?.tableConfig &&
      !defaultConfigAppliedRef.current &&
      pageIndex === 1 &&
      fieldFilters.length === 0 &&
      filterGroups.length === 0 &&
      sort.length === 0 &&
      query === ""
    ) {
      defaultConfigAppliedRef.current = true;
      handleApplyConfig(defaultConfig.tableConfig, {
        id: defaultConfig.id,
        name: defaultConfig.name,
      });
    }
  }, [
    defaultConfig,
    pageIndex,
    fieldFilters.length,
    filterGroups.length,
    sort.length,
    query,
    handleApplyConfig,
  ]);

  const {
    columnVisibility: liveColumnVisibility,
    columnOrder: liveColumnOrder,
    columnSizing: liveColumnSizing,
    columnPinning: liveColumnPinning,
  } = table.getState();

  const currentConfig = useMemo<TableConfig>(() => {
    const columnVisibility: Record<string, boolean> = {};
    for (const col of table.getAllLeafColumns()) {
      columnVisibility[col.id] = liveColumnVisibility[col.id] ?? true;
    }

    return {
      fieldFilters: fieldFilters,
      filterGroups: filterGroups,
      joinOperator: "and",
      sort: effectiveSort,
      pageSize,
      columnVisibility,
      columnOrder: liveColumnOrder,
      columnSizing: liveColumnSizing,
      columnPinning: {
        left: liveColumnPinning.left ?? [],
        right: liveColumnPinning.right ?? [],
      },
      density,
      formatRules,
    };
  }, [
    fieldFilters,
    filterGroups,
    effectiveSort,
    pageSize,
    liveColumnVisibility,
    liveColumnOrder,
    liveColumnSizing,
    liveColumnPinning,
    density,
    formatRules,
    table,
  ]);

  const isViewDirty = useMemo(
    () => (activeView ? !isTableConfigEqual(currentConfig, activeView.config) : false),
    [activeView, currentConfig],
  );

  const handleViewPersisted = useCallback((config: TableConfiguration) => {
    setActiveView({ id: config.id, name: config.name, config: config.tableConfig });
  }, []);

  const handleViewDeleted = useCallback((id: string) => {
    setActiveView((current) => (current?.id === id ? null : current));
  }, []);

  const compiledFormatRules = useMemo(
    () => compileFormatRules<TData>(formatRules, table.getAllLeafColumns()),
    [formatRules, table],
  );

  const hasActiveFilters = filterItems.length > 0 || query !== "";
  const handleClearFilters = useCallback(() => {
    setFilterItems([]);
    void setSearchParams({ query: "", pageIndex: 1 });
  }, [setSearchParams]);

  const sensors = useSensors(useSensor(PointerSensor, { activationConstraint: { distance: 8 } }));

  const handleColumnDragEnd = useCallback(
    (event: DragEndEvent) => {
      const { active, over } = event;
      if (!over || active.id === over.id) return;
      const ids = table.getAllLeafColumns().map((col) => col.id);
      const oldIndex = ids.indexOf(String(active.id));
      const newIndex = ids.indexOf(String(over.id));
      if (oldIndex < 0 || newIndex < 0) return;
      table.setColumnOrder(arrayMove(ids, oldIndex, newIndex));
    },
    [table],
  );

  const listRow = useMemo(() => {
    if (!panelEntityId || panelMode !== "edit") return null;
    const results = currentPageResults || [];
    return results.find((row: TData) => (row as { id?: string }).id === panelEntityId) ?? null;
  }, [panelEntityId, panelMode, currentPageResults]);

  const panelRow = listRow;

  const columnSizeVars: Record<string, string> = {};
  let totalSize = 0;
  for (const header of table.getFlatHeaders()) {
    const { column } = header;
    const size = header.getSize();
    columnSizeVars[columnSizeVar(column.id)] = `${size}px`;
    totalSize += size;

    const pinned = column.getIsPinned();
    if (pinned === "left") {
      columnSizeVars[columnPinOffsetVar(column.id, "left")] = `${column.getStart("left")}px`;
    } else if (pinned === "right") {
      columnSizeVars[columnPinOffsetVar(column.id, "right")] = `${column.getAfter("right")}px`;
    }
  }

  const reorderableIds = table.getVisibleLeafColumns().map((col) => col.id);

  return (
    <DataTableProvider
      isLoading={dataQuery.isLoading}
      table={table}
      columns={tableColumns}
      isPanelOpen={isPanelOpen}
      panelMode={panelMode}
      panelRow={panelRow}
      rowSelection={rowSelection}
      selectedCount={selectedCount}
      getSelectedRows={getSelectedRows}
      openPanelCreate={openPanelCreate}
      openPanelEdit={openPanelEdit}
      closePanel={closePanel}
      hasPanel={hasPanel}
      canOpenPanel={canUpdate || enableReadOnlyPanel}
      canCreate={canCreate}
      canUpdate={canUpdate}
      canExport={canExport}
      pagination={pagination}
    >
      <DataTablePanelWrapper>
        <DataTablePanelContent>
          <div className="flex size-full min-w-0 flex-col gap-2">
            <DataTableToolbar
              table={table}
              columns={columns}
              query={query}
              onSearchChange={handleSearchChange}
              filters={filterItems}
              onFiltersChange={handleFiltersChange}
              sort={sort}
              onSortChange={handleSortArrayChange}
              addRecordActions={resolvedAddRecordActions}
              resource={name}
              currentConfig={currentConfig}
              onApplyConfig={handleApplyConfig}
              activeView={activeView}
              isViewDirty={isViewDirty}
              onViewPersisted={handleViewPersisted}
              onViewDeleted={handleViewDeleted}
              formatRules={formatRules}
              onFormatRulesChange={setFormatRules}
              density={density}
              onDensityChange={setDensity}
              exportContext={{
                graphql,
                queryOptions: baseQueryOptions,
                currentPageRows: currentPageResults ?? [],
                totalCount,
              }}
            />
            <DataTableFilterChips
              filters={filterItems}
              onFiltersChange={handleFiltersChange}
              query={query}
              onClearQuery={() => handleSearchChange("")}
            />
            {enableRowSelection && totalCount != null && (
              <DataTableSelectionBanner
                visible={allPageRowsSelected && totalCount > currentPageRowCount}
                selectedCount={selectedCount}
                totalCount={totalCount}
                maxSelectable={BULK_SELECT_MAX}
                isSelectingAll={isSelectingAll}
                onSelectAllMatching={handleSelectAllMatching}
                onClearSelection={handleClearSelection}
              />
            )}
            <div className="relative min-w-0">
              <DataTableRefreshPill
                visible={liveRefresh.hasPendingUpdate}
                onRefresh={liveRefresh.applyStaged}
                onDismiss={liveRefresh.dismissStaged}
              />
              <DndContext
                sensors={sensors}
                collisionDetection={closestCenter}
                modifiers={[restrictToHorizontalAxis]}
                onDragEnd={handleColumnDragEnd}
              >
                <Table
                  data-density={density}
                  className={cn(
                    "border-separate border-spacing-0",
                    density === "compact" && "[&_td]:py-1 [&_td]:text-xs [&_th]:h-8",
                  )}
                  containerClassName="max-h-[calc(65vh_-_var(--top-bar-height))] rounded-md border border-border"
                  style={{ ...columnSizeVars, minWidth: `${totalSize}px` }}
                >
                  <TableHeader className="sticky top-0 z-20 bg-muted backdrop-blur-sm">
                    {table.getHeaderGroups().map((headerGroup) => (
                      <TableRow key={headerGroup.id} className="hover:bg-transparent">
                        <SortableContext
                          items={reorderableIds}
                          strategy={horizontalListSortingStrategy}
                        >
                          {headerGroup.headers.map((header) => (
                            <DataTableHeaderCell
                              key={header.id}
                              header={header}
                              sort={sort}
                              onSort={handleSortChange}
                            />
                          ))}
                        </SortableContext>
                      </TableRow>
                    ))}
                  </TableHeader>
                  <DataTableBody
                    table={table}
                    columns={tableColumns}
                    isLoading={dataQuery.isLoading}
                    contextMenuActions={contextMenuActions}
                    onRowClick={onRowClick}
                    getFormatClass={compiledFormatRules}
                    hasActiveFilters={hasActiveFilters}
                    onClearFilters={handleClearFilters}
                  />
                </Table>
              </DndContext>
            </div>
            <DataTablePagination
              table={table}
              onPageChange={handlePageChange}
              onPageSizeChange={handlePageSizeChange}
              mode="cursor"
              hasNextPage={cursorPageInfo?.hasNextPage}
              currentPageRowCount={currentPageRowCount}
              totalCount={totalCount}
            />
          </div>
        </DataTablePanelContent>
        {TablePanel && (
          <TablePanel
            open={isPanelOpen}
            onOpenChange={handlePanelOpenChange}
            mode={panelMode}
            row={panelRow}
          />
        )}
      </DataTablePanelWrapper>
      {enableRowSelection && dockActions.length > 0 && (
        <DataTableDock table={table} actions={dockActions} />
      )}
    </DataTableProvider>
  );
}
