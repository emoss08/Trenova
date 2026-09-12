"use no memo";
import { useT } from "@trenova/shared/i18n/use-t";
import { DataTableProvider } from "@/contexts/data-table-context";
import { useDataTableFilterSync } from "@/hooks/data-table/use-data-table-filter-sync";
import { useDataTableLiveRefresh } from "@/hooks/data-table/use-data-table-live-refresh";
import { useDataTableQuery } from "@/hooks/data-table/use-data-table-query";
import { searchParamsParser } from "@/hooks/data-table/use-data-table-state";
import { useGuardedRowActions } from "@/hooks/use-pending-actions";
import { usePermissions } from "@/hooks/use-permission";
import {
  columnPinOffsetVar,
  columnSizeVar,
  compileFormatRules,
  emptyTableColumns,
  fromColumnPinningState,
  isTableConfigEqual,
  toColumnPinningState,
  updateSortField,
} from "@/lib/data-table";
import { stableStringify } from "@/lib/stable-stringify";
import { fetchAllRows } from "@/lib/data-table-export";
import { queries } from "@/lib/queries";
import { cn } from "@trenova/shared/lib/utils";
import type {
  DataTableProps,
  FilterItem,
  PanelMode,
  SortDirection,
  SortField,
  Row,
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
import { useTable, type RowSelectionState } from "@tanstack/react-table";
import { dataTableFeatures } from "@trenova/shared/lib/table-features";
import { useQueryStates } from "nuqs";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { toast } from "sonner";
import { Table, TableHeader, TableRow } from "@trenova/shared/components/ui/table";
import { DataTablePagination } from "./_components/data-table-pagination";
import { DataTableBody } from "./data-table-body";
import { DataTableDock } from "./data-table-dock";
import { DataTableEmptyState } from "./data-table-empty-state";
import DataTableFilterChips from "./data-table-filter-chips";
import { DataTableHeaderCell } from "./data-table-header-cell";
import { DataTablePanelContent, DataTablePanelWrapper } from "./data-table-panel";
import { DataTableRefreshPill } from "./data-table-refresh-pill";
import { DataTableSelectionBanner } from "./data-table-selection-banner";
import { createSelectionColumn } from "./data-table-selection-column";
import { DataTableToolbar } from "./data-table-toolbar";

const BULK_SELECT_MAX = 1000;

type CursorState = {
  scopeKey: string;
  cursors: Record<number, string | null>;
  totalCount: number | null;
};

const EMPTY_CURSOR_STATE: CursorState = { scopeKey: "", cursors: { 0: null }, totalCount: null };
const EMPTY_PINNING = { left: [] as string[], right: [] as string[] };
const NO_ROW_ACTIONS: never[] = [];

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
  contextMenuActions: unguardedContextMenuActions,
  onRowClick,
  enableCreateAction = true,
  enableReadOnlyPanel = false,
  initialColumnVisibility,
  graphql,
  refetchIntervalMs,
  onCellEditCommit,
  renderEmptyState,
}: DataTableProps<TData>) {
  "use no memo";
  const t = useT();

  const contextMenuActions = useGuardedRowActions<TData>(
    unguardedContextMenuActions ?? NO_ROW_ACTIONS,
  );
  const permissions = usePermissions(resource ?? "");
  const canCreate = resource ? permissions.canCreate : true;
  const canUpdate = resource ? permissions.canUpdate : true;
  const canExport = resource ? permissions.canExport : true;
  const [searchParams, setSearchParams] = useQueryStates(searchParamsParser);
  const { pageIndex, pageSize, query, fieldFilters, filterGroups, sort, panelType, panelEntityId } =
    searchParams;
  const [rowSelection, setRowSelection] = useState<RowSelectionState>({});
  const [cursorState, setCursorState] = useState<CursorState>(EMPTY_CURSOR_STATE);
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

  const { filterItems, setFilterItems, applyFilterState } = useDataTableFilterSync({
    columns,
    fieldFilters: fieldFilters ?? [],
    filterGroups: filterGroups ?? [],
    setSearchParams,
  });

  const handleFiltersChange = useCallback(
    (items: FilterItem[]) => {
      setFilterItems(items);
    },
    [setFilterItems],
  );

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
      stableStringify({
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
  const scopedCursorState =
    cursorState.scopeKey === cursorScopeKey ? cursorState : EMPTY_CURSOR_STATE;
  const currentCursor = scopedCursorState.cursors[zeroBasedPageIndex];
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
    if (pageInfo?.mode !== "cursor") {
      return;
    }

    const nextPageIndex = zeroBasedPageIndex + 1;
    const nextCursor = pageInfo.hasNextPage && pageInfo.endCursor ? pageInfo.endCursor : null;
    const pageTotalCount = pageInfo.totalCount ?? null;
    setCursorState((current) => {
      const inScope = current.scopeKey === cursorScopeKey;
      const scoped = inScope ? current : EMPTY_CURSOR_STATE;
      const totalCount = pageTotalCount ?? scoped.totalCount;
      const cursorKnown = nextCursor === null || scoped.cursors[nextPageIndex] === nextCursor;
      if (inScope && cursorKnown && totalCount === scoped.totalCount) {
        return current;
      }

      return {
        scopeKey: cursorScopeKey,
        cursors: cursorKnown
          ? scoped.cursors
          : {
              ...scoped.cursors,
              [nextPageIndex]: nextCursor,
            },
        totalCount,
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
    ? (cursorPageInfo.totalCount ?? scopedCursorState.totalCount)
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
  const table = useTable({
    features: dataTableFeatures,
    data: currentPageResults || [],
    columns: tableColumns,
    pageCount,
    rowCount,
    manualPagination: true,
    columnResizeMode: "onChange",
    enableColumnPinning: true,
    getRowId: (row) => row.id,
    manualSorting: true,
    enableRowSelection,
    enableMultiRowSelection: true,
    enableCellEditing: !!onCellEditCommit && canUpdate,
    onCellEditCommit,
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
      toast.error(t("Selection failed"), {
        description: error instanceof Error ? error.message : "Could not load all matching rows.",
      });
    } finally {
      setIsSelectingAll(false);
    }
  }, [graphql, baseQueryOptions, t]);

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

      applyFilterState({ fieldFilters: newFieldFilters, filterGroups: newFilterGroups });

      table.setColumnVisibility(newColumnVisibility);
      table.setColumnOrder(newColumnOrder);
      table.setColumnSizing(newColumnSizing);
      table.setColumnPinning(toColumnPinningState(newColumnPinning));
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
    [setSearchParams, applyFilterState, pageSize, table],
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
  } = table.state;

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
      columnPinning: fromColumnPinningState(liveColumnPinning),
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
  }, [setFilterItems, setSearchParams]);

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
    if (pinned === "start") {
      columnSizeVars[columnPinOffsetVar(column.id, "start")] = `${column.getStart("start")}px`;
    } else if (pinned === "end") {
      columnSizeVars[columnPinOffsetVar(column.id, "end")] = `${column.getAfter("end")}px`;
    }
  }

  const reorderableIds = table.getVisibleLeafColumns().map((col) => col.id);

  // An empty page is drawn as the table it will become rather than as a
  // table with no rows, so the header row and the pager step aside for it.
  const isEmpty = !dataQuery.isLoading && !dataQuery.isError && currentPageRowCount === 0;
  const emptyColumns = useMemo(
    () => emptyTableColumns(table.getVisibleLeafColumns()),
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [table, liveColumnVisibility, liveColumnOrder],
  );
  const defaultCreate = resolvedAddRecordActions.find((action) => action.id === "default-create");

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
            {isEmpty ? (
              <div className="border-border rounded-md border">
                {renderEmptyState ? (
                  renderEmptyState({ hasActiveFilters, onClearFilters: handleClearFilters })
                ) : (
                  <DataTableEmptyState
                    name={name}
                    columns={emptyColumns}
                    hasActiveFilters={hasActiveFilters}
                    onClearFilters={handleClearFilters}
                    onAddRecord={hasActiveFilters ? undefined : defaultCreate?.onClick}
                  />
                )}
              </div>
            ) : (
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
                    <TableHeader className="bg-muted sticky top-0 z-20 backdrop-blur-sm">
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
                    />
                  </Table>
                </DndContext>
              </div>
            )}
            {isEmpty ? null : (
              <DataTablePagination
                table={table}
                onPageChange={handlePageChange}
                onPageSizeChange={handlePageSizeChange}
                mode="cursor"
                hasNextPage={cursorPageInfo?.hasNextPage}
                currentPageRowCount={currentPageRowCount}
                totalCount={totalCount}
              />
            )}
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
