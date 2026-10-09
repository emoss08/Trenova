import { DataTableProvider } from "@/contexts/data-table-context";
import { useDataTableFilterSync } from "@/hooks/data-table/use-data-table-filter-sync";
import { useDataTableLayout } from "@/hooks/data-table/use-data-table-layout";
import { useDataTableLiveRefresh } from "@/hooks/data-table/use-data-table-live-refresh";
import {
  buildDataTableQueryKey,
  fetchDataTablePage,
  useDataTableQuery,
} from "@/hooks/data-table/use-data-table-query";
import { useDataTableRowCursor } from "@/hooks/data-table/use-data-table-row-cursor";
import {
  defaultPageSizeFor,
  resolvePageSize,
  searchParamsParser,
} from "@/hooks/data-table/use-data-table-state";
import { usePageViewRegistration } from "@/hooks/data-table/use-page-view-registration";
import { useGuardedRowActions } from "@/hooks/use-pending-actions";
import { usePermissions } from "@/hooks/use-permission";
import {
  clampFittedWidth,
  columnFacetField,
  columnLayout,
  compileFormatRules,
  emptyTableColumns,
  fromColumnPinningState,
  filterItemFromField,
  getFilterableFields,
  isChangedSince,
  type FilterableField,
  isColumnResizable,
  measureColumnFits,
  isTableConfigEqual,
  toColumnPinningState,
  updateSortField,
  withRequiredPinning,
} from "@/lib/data-table";
import {
  buildCsv,
  buildExportColumns,
  downloadCsv,
  exportFilename,
  fetchAllRows,
} from "@/lib/data-table-export";
import { Download01Icon, Edit05Icon } from "@trenova/shared/components/icons";
import { BulkEditDialog } from "./bulk-edit/bulk-edit-dialog";
import { BulkEditProgress } from "./bulk-edit/bulk-edit-progress";
import type { BulkEditJob } from "@/lib/graphql/bulk-edit";
import type { BulkEditSelectionInput } from "@trenova/graphql/generated/graphql";
import { groupedScope, groupedSort } from "@/lib/data-table-grouping";
import { resolveGraphQLVariableSources } from "@/lib/data-table-variables";
import { queries } from "@/lib/queries";
import { stableStringify } from "@/lib/stable-stringify";
import type {
  ActiveTableView,
  TableConfig,
  TableConfiguration,
  TableDensity,
  TableFormatRule,
  TableLayout,
  TableViewSource,
} from "@/types/table-configuration";
import type { ComposedTableQuery } from "@/types/table-query";
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
import { keepPreviousData, useQuery, useQueryClient } from "@tanstack/react-query";
import { useTable, type RowPinningState, type RowSelectionState } from "@tanstack/react-table";
import { Table, TableHeader, TableRow } from "@trenova/shared/components/ui/table";
import { useT } from "@trenova/shared/i18n/use-t";
import {
  dataTableFeatures,
  hasSelectedRows,
  selectDataTableViewState,
} from "@trenova/shared/lib/table-features";
import { cn, toSentenceFragment } from "@trenova/shared/lib/utils";
import type {
  DataTableFilterField,
  DataTableGroupKey,
  DockAction,
  DataTableProps,
  FieldFilter,
  FilterItem,
  PanelMode,
  Row,
  SortDirection,
  SortField,
} from "@trenova/shared/types/data-table";
import { useLatestCallback } from "@trenova/shared/hooks/use-latest-callback";
import { useDataTableCellWrites } from "@/hooks/data-table/use-data-table-cell-writes";
import { RecordPresence } from "@/components/presence/data-table-viewers";
import { parseAsInteger, useQueryStates } from "nuqs";
import { useCallback, useEffect, useLayoutEffect, useMemo, useRef, useState } from "react";
import { toast } from "sonner";
import { DataTablePagination } from "./_components/data-table-pagination";
import { DataTableBody } from "./data-table-body";
import { DataTableChangesBanner } from "./data-table-changes-banner";
import { aggregateFieldOf, DataTableTotalsRow } from "./data-table-totals-row";
import type { DataTableHeaderFacets } from "./data-table-header-facet-filter";
import { useDataTableInsights } from "@/hooks/data-table/use-data-table-insights";
import { resolveInputExtraVariables } from "@/lib/data-table-variables";
import { clipboardColumns } from "@/lib/data-table-clipboard";
import { MAX_PINNED_ROWS } from "./_components/data-table-context-menu";
import { DataTableDock, type DataTableSelectionTotal } from "./data-table-dock";
import { DataTableEmptyState } from "./data-table-empty-state";
import DataTableFilterChips from "./data-table-filter-chips";
import { DataTableHeaderCell } from "./data-table-header-cell";
import { DataTablePanelContent, DataTablePanelWrapper } from "./data-table-panel";
import { DataTableRefreshPill } from "./data-table-refresh-pill";
import { DataTableSelectionBanner } from "./data-table-selection-banner";
import { createSelectionColumn } from "./data-table-selection-column";
import { DataTableToolbar } from "./data-table-toolbar";
import { translate } from "@trenova/shared/i18n/runtime";

const BULK_SELECT_MAX = 1000;
const COLUMN_DRAG_MODIFIERS = [restrictToHorizontalAxis];

type CursorState = {
  scopeKey: string;
  cursors: Record<number, string | null>;
  totalCount: number | null;
};

const EMPTY_CURSOR_STATE: CursorState = { scopeKey: "", cursors: { 0: null }, totalCount: null };
const EMPTY_PINNING = { left: [] as string[], right: [] as string[] };
const EMPTY_GROUP_KEYS: DataTableGroupKey[] = [];
const NO_ROW_ACTIONS: never[] = [];
const NO_ROWS: never[] = [];
const NO_ROW_PINNING: RowPinningState = { top: [], bottom: [] };
const NO_SEEN_ROWS: ReadonlySet<string> = new Set();
const NO_VALUES: readonly string[] = [];
const NO_FILTER_FIELDS: DataTableFilterField[] = [];
const NO_SCOPE_FILTERS: FieldFilter[] = [];
const noop = () => {};
const DENSITY_COMPACT_ROW = "[--row-h:var(--row-h-compact)]";

export function DataTable<TData extends Record<string, any>>({
  columns,
  filterFields: tableFilterFields = NO_FILTER_FIELDS,
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
  initialColumnPinning,
  graphql,
  refetchIntervalMs,
  onCellEditCommit,
  renderEmptyState,
  emptyTitle,
  scopeFilters: ownScopeFilters = NO_SCOPE_FILTERS,
  enableExport = true,
  pageSizeOptions,
  getRowClassName,
  grouping,
  expansion,
  keyboard,
  toolbar,
  alternateView,
  initialDensity = "comfortable",
  footerLeading,
}: DataTableProps<TData>) {
  const t = useT();

  const contextMenuActions = useGuardedRowActions<TData>(
    unguardedContextMenuActions ?? NO_ROW_ACTIONS,
  );
  const permissions = usePermissions(resource ?? "");
  const canCreate = resource ? permissions.canCreate : true;
  const canUpdate = resource ? permissions.canUpdate : true;
  const canExport = enableExport && (resource ? permissions.canExport : true);
  const defaultPageSize = defaultPageSizeFor(pageSizeOptions);
  const tableSearchParamsParser = useMemo(
    () => ({ ...searchParamsParser, pageSize: parseAsInteger.withDefault(defaultPageSize) }),
    [defaultPageSize],
  );
  const [searchParams, setSearchParams] = useQueryStates(tableSearchParamsParser);
  const { pageIndex, query, fieldFilters, filterGroups, sort, panelType, panelEntityId } =
    searchParams;
  const pageSize = resolvePageSize(searchParams.pageSize, pageSizeOptions);
  const [cursorState, setCursorState] = useState<CursorState>(EMPTY_CURSOR_STATE);
  const [activeView, setActiveView] = useState<ActiveTableView | null>(null);
  const [density, setDensity] = useState<TableDensity>(initialDensity);
  const [formatRules, setFormatRules] = useState<TableFormatRule[]>([]);
  const [isSelectingAll, setIsSelectingAll] = useState(false);
  const [initialStateApplied, setInitialStateApplied] = useState(false);
  const [rowPinning, setRowPinning] = useState<RowPinningState>(NO_ROW_PINNING);
  const [pinnedRowsCollapsed, setPinnedRowsCollapsed] = useState(false);
  // Rows changed after this moment (the person's last visit) are marked, until
  // they open one or mark them all seen. Zero marks nothing: a first visit.
  const [changesSince, setChangesSince] = useState(0);
  const [hideChangesSinceLastVisit, setHideChangesSinceLastVisit] = useState(false);
  const [seenRowIds, setSeenRowIds] = useState<ReadonlySet<string>>(NO_SEEN_ROWS);
  const [hideTotals, setHideTotals] = useState(false);
  const [virtualized, setVirtualized] = useState(false);
  const queryClient = useQueryClient();
  const selectedRowsMapRef = useRef(new Map<string, TData>());

  const { data: defaultConfig, isPending: defaultConfigPending } = useQuery({
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
        label: t("New {0}", toSentenceFragment(name)),
        description: t("Create a new record from scratch."),
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
    t,
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

  const filterFields = useMemo(
    () => getFilterableFields(columns, tableFilterFields),
    [columns, tableFilterFields],
  );

  const { filterItems, setFilterItems, applyFilterState } = useDataTableFilterSync({
    fields: filterFields,
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

  /**
   * A question answered as filters, put where a hand-built filter goes.
   *
   * Nothing is applied that is not shown: the composed filters land in the
   * same builder and the same chips, so the person sees each one and can
   * change or drop it before reading a single row. Text the composer could not
   * place on a field becomes the search term rather than being thrown away.
   */
  const handleAskApplied = useCallback(
    (composed: ComposedTableQuery) => {
      applyFilterState({ fieldFilters: composed.fieldFilters, filterGroups: [] });
      void setSearchParams({
        query: composed.query,
        sort: composed.sort,
        pageIndex: 1,
      });
    },
    [applyFilterState, setSearchParams],
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
  const effectiveSort = useMemo(() => groupedSort(grouping, sort), [grouping, sort]);
  const collapsedGroupScope = useMemo(
    () => groupedScope(grouping),
    // oxlint-disable-next-line react-hooks/exhaustive-deps
    [grouping?.field, grouping?.collapsedKeys, grouping?.collapsedScope],
  );
  const scopeFilters = useMemo(
    () =>
      collapsedGroupScope.fieldFilters.length > 0
        ? [...ownScopeFilters, ...collapsedGroupScope.fieldFilters]
        : ownScopeFilters,
    [collapsedGroupScope, ownScopeFilters],
  );
  const effectiveFilterGroups = useMemo(
    () =>
      collapsedGroupScope.filterGroups.length > 0
        ? [...collapsedGroupScope.filterGroups, ...filterGroups]
        : filterGroups,
    [collapsedGroupScope, filterGroups],
  );
  const cursorScopeKey = useMemo(
    () =>
      stableStringify({
        pageSize,
        query,
        scopeFilters,
        fieldFilters,
        filterGroups: effectiveFilterGroups,
        sort: effectiveSort,
        graphql: {
          connectionKey: graphql.connectionKey,
          operationName: graphql.operationName,
          ...resolveGraphQLVariableSources(graphql, pageSize),
        },
      }),
    [effectiveSort, fieldFilters, effectiveFilterGroups, graphql, pageSize, query, scopeFilters],
  );
  const scopedCursorState =
    cursorState.scopeKey === cursorScopeKey ? cursorState : EMPTY_CURSOR_STATE;
  const currentCursor = scopedCursorState.cursors[zeroBasedPageIndex];
  const canFetchPage = zeroBasedPageIndex === 0 || currentCursor !== undefined;

  const baseQueryOptions = useMemo(
    () => ({
      query,
      fieldFilters: scopeFilters.length > 0 ? [...scopeFilters, ...fieldFilters] : fieldFilters,
      filterGroups: effectiveFilterGroups,
      sort: effectiveSort,
    }),
    [query, scopeFilters, fieldFilters, effectiveFilterGroups, effectiveSort],
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
    queryKey: buildDataTableQueryKey(queryKey, graphql, pagination, queryOptions),
    scopeKey: `${cursorScopeKey}:${zeroBasedPageIndex}`,
    results: dataQuery.data?.results,
    isPlaceholderData: dataQuery.isPlaceholderData,
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

  const collapsedGroupKeys = grouping?.collapsedKeys ?? EMPTY_GROUP_KEYS;
  const [settledCollapsedKeys, setSettledCollapsedKeys] = useState(collapsedGroupKeys);
  const groupOrderKey = grouping
    ? [grouping.field, ...(grouping.tieBreakers ?? []).map((entry) => entry.field)].join("\u0000")
    : "";
  const [settledGroupOrderKey, setSettledGroupOrderKey] = useState(groupOrderKey);
  useEffect(() => {
    if (dataQuery.data && !dataQuery.isPlaceholderData) {
      setSettledCollapsedKeys(collapsedGroupKeys);
      setSettledGroupOrderKey(groupOrderKey);
    }
  }, [collapsedGroupKeys, groupOrderKey, dataQuery.data, dataQuery.isPlaceholderData]);
  // The previous page is kept on screen while the next loads, but rows sorted
  // for one grouping cannot be laid out under another's headers.
  const regrouping = dataQuery.isPlaceholderData && settledGroupOrderKey !== groupOrderKey;
  const isLoadingPage = dataQuery.isLoading || regrouping || !initialStateApplied;
  const loadingGroupKeys = useMemo(
    () =>
      dataQuery.isPlaceholderData
        ? settledCollapsedKeys.filter((key) => !collapsedGroupKeys.includes(key))
        : EMPTY_GROUP_KEYS,
    [collapsedGroupKeys, dataQuery.isPlaceholderData, settledCollapsedKeys],
  );

  // Pinned rows are fetched by ID, so they stay at the top whatever page, filter or
  // search is showing; only the filters the table's host imposes still apply.
  const pinnedRowIds = rowPinning.top;
  // The source is named in the key by its operation; the config itself carries a
  // document and functions that do not belong in a hashed key.
  // oxlint-disable-next-line @tanstack/query/exhaustive-deps
  const pinnedQuery = useQuery({
    queryKey: [queryKey, "pinned", graphql.operationName, scopeFilters, pinnedRowIds],
    queryFn: ({ signal }) =>
      fetchDataTablePage<TData>({
        pageSize: pinnedRowIds.length,
        options: {
          fieldFilters: [...scopeFilters, { field: "id", operator: "in", value: pinnedRowIds }],
        },
        graphql,
        signal,
      }),
    enabled: initialStateApplied && pinnedRowIds.length > 0,
    placeholderData: keepPreviousData,
    staleTime: 30_000,
  });
  const pinnedResults = pinnedRowIds.length > 0 ? pinnedQuery.data?.results : undefined;

  const cursorPageInfo = dataQuery.data?.pageInfo ?? null;
  const currentPageResults = regrouping ? undefined : liveRefresh.results;
  const tableData = useMemo(() => {
    const page = currentPageResults ?? NO_ROWS;
    if (!pinnedResults || pinnedResults.length === 0) return page;
    const onPage = new Set(page.map((row) => (row as { id?: string }).id));
    const extra = pinnedResults.filter((row) => !onPage.has((row as { id?: string }).id));
    return extra.length > 0 ? [...extra, ...page] : page;
  }, [currentPageResults, pinnedResults]);
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
  const table = useTable(
    {
      features: dataTableFeatures,
      // Rows wait for the person's layout, so the table never draws a page in its
      // default columns and then jumps to theirs.
      data: initialStateApplied ? tableData : NO_ROWS,
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
      meta: getRowClassName ? { getRowClassName } : undefined,
      initialState: {
        ...(initialColumnVisibility ? { columnVisibility: initialColumnVisibility } : {}),
        ...(initialColumnPinning
          ? { columnPinning: toColumnPinningState(initialColumnPinning) }
          : {}),
      },
      state: {
        pagination,
        rowPinning,
      },
      onRowPinningChange: setRowPinning,
      enableRowPinning: !!name,
      keepPinnedRows: true,
      // Cells are selected with Alt held, so a plain click still opens or expands
      // the row and a plain drag still selects text.
      enableCellSelection: (cell) => cell.column.id !== "select",
      enableCellSelectionDrag: true,
      isCellRangeSelectionEvent: (event) => (event as MouseEvent).shiftKey,
      isMultiCellRangeSelectionEvent: (event) =>
        (event as MouseEvent).metaKey || (event as MouseEvent).ctrlKey,
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
    },
    selectDataTableViewState,
  );

  // Selection is the table's own state, read where it is drawn; the shell only keeps
  // the selected records in step with it, outside render, for the dock's actions.
  const selectionAtom = table.atoms.rowSelection;
  useEffect(() => {
    const sync = (selection: RowSelectionState) => {
      const map = selectedRowsMapRef.current;
      for (const id of map.keys()) {
        if (!selection[id]) map.delete(id);
      }
      if (tableData.length > 0) {
        for (const row of tableData) {
          const id = (row as { id?: string }).id;
          if (id && selection[id]) map.set(id, row);
        }
      }
    };
    sync(selectionAtom.get());
    const subscription = selectionAtom.subscribe(sync);
    return () => subscription.unsubscribe();
  }, [selectionAtom, tableData]);

  // Read when asked, from the rows the table holds now, so a total or an action never
  // sees a selection from before the latest tick. A row picked by "select all matching"
  // that is not on this page comes from the records that selection fetched.
  const getSelectedRows = useLatestCallback((): TData[] => {
    const selection = table.atoms.rowSelection.get();
    const rowsById = table.getCoreRowModel().rowsById;
    const kept = selectedRowsMapRef.current;
    const rows: TData[] = [];
    for (const id in selection) {
      if (!selection[id]) continue;
      const row = rowsById[id]?.original ?? kept.get(id);
      if (row) rows.push(row);
    }
    return rows;
  });

  const handleSelectAllMatching = useLatestCallback(async () => {
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
      table.setRowSelection(selection);
    } catch (error) {
      toast.error(t("Selection failed"), {
        description:
          error instanceof Error ? error.message : translate("Could not load all matching rows."),
      });
    } finally {
      setIsSelectingAll(false);
    }
  });

  const handleClearSelection = useLatestCallback(() => {
    table.setRowSelection({});
  });

  const pageRowIds = useMemo(
    () =>
      (currentPageResults ?? [])
        .map((row) => (row as { id?: string }).id)
        .filter((id): id is string => !!id),
    [currentPageResults],
  );
  const handleTogglePin = useLatestCallback((rowId: string) => {
    setRowPinning((current) => {
      if (current.top.includes(rowId)) {
        return { ...current, top: current.top.filter((id) => id !== rowId) };
      }
      if (current.top.length >= MAX_PINNED_ROWS) return current;
      return { ...current, top: [...current.top, rowId] };
    });
  });

  const openedRowId =
    keyboard?.cursorRowId ?? expansion?.expandedRowId ?? panelEntityId ?? null;
  useEffect(() => {
    if (!openedRowId) return;
    setSeenRowIds((current) => {
      if (current.has(openedRowId)) return current;
      const next = new Set(current);
      next.add(openedRowId);
      return next;
    });
  }, [openedRowId]);

  const visibleLeafColumns = table.getVisibleLeafColumns();
  const aggregateFields = useMemo(
    () =>
      visibleLeafColumns
        .map((column) => aggregateFieldOf(column))
        .filter((field): field is string => !!field),
    [visibleLeafColumns],
  );
  const insightOptions = useMemo(
    () => resolveInputExtraVariables(graphql, pageSize, baseQueryOptions),
    [graphql, pageSize, baseQueryOptions],
  );
  const insights = useDataTableInsights({
    resource,
    queryOptions: baseQueryOptions,
    options: insightOptions,
    aggregateFields,
    showTotals: !hideTotals,
    dataVersion: dataQuery.dataUpdatedAt,
  });
  const hasTotals = aggregateFields.some((field) => insights.summable.has(field));
  const applyFacetFilter = useLatestCallback(
    (field: FilterableField, values: readonly string[]) => {
      const kept = filterItems.filter(
        (item) =>
          !(item.type === "filter" && item.apiField === field.apiField && item.operator === "in"),
      );
      setFilterItems(
        values.length === 0
          ? kept
          : [...kept, filterItemFromField(field, { operator: "in", value: [...values] }, "and")],
      );
    },
  );
  const headerFacets = useMemo<DataTableHeaderFacets | undefined>(
    () =>
      resource && insights.facetable.size > 0
        ? {
            resource,
            facetable: insights.facetable,
            queryOptions: baseQueryOptions,
            options: insightOptions,
            activeValues: (apiField) => {
              for (const item of filterItems) {
                const matches =
                  item.type === "filter" && item.apiField === apiField && item.operator === "in";
                if (matches) return Array.isArray(item.value) ? (item.value as string[]) : [];
              }
              return NO_VALUES;
            },
            apply: applyFacetFilter,
          }
        : undefined,
    [resource, insights.facetable, baseQueryOptions, insightOptions, filterItems, applyFacetFilter],
  );
  const selectionTotals = useMemo(() => {
    const textColumns = clipboardColumns(table.getAllLeafColumns());
    const totals: DataTableSelectionTotal<TData>[] = [];
    for (const column of visibleLeafColumns) {
      const aggregate = column.columnDef.meta?.aggregate;
      const text = textColumns.get(column.id);
      if (!aggregate || !text) continue;
      totals.push({
        id: column.id,
        label: column.columnDef.meta?.label ?? text.header,
        format: aggregate.format ?? "number",
        getValue: text.getValue,
      });
    }
    return totals;
  }, [table, visibleLeafColumns]);
  const showTotalsRow = !hideTotals && hasTotals;

  const showChanges = changesSince > 0 && !hideChangesSinceLastVisit;
  const unseenCount = useMemo(() => {
    if (!showChanges) return 0;
    let count = 0;
    for (const row of tableData) {
      if (isChangedSince(row, changesSince) && !seenRowIds.has((row as unknown as { id: string }).id)) {
        count += 1;
      }
    }
    return count;
  }, [showChanges, tableData, changesSince, seenRowIds]);
  const markAllSeen = useCallback(() => {
    setChangesSince(Math.floor(Date.now() / 1000));
    setSeenRowIds(NO_SEEN_ROWS);
  }, []);

  const cursorRowIds = useMemo(() => {
    if (pinnedRowsCollapsed || pinnedRowIds.length === 0) return pageRowIds;
    const pinned = new Set(pinnedRowIds);
    return [...pinnedRowIds, ...pageRowIds.filter((id) => !pinned.has(id))];
  }, [pageRowIds, pinnedRowIds, pinnedRowsCollapsed]);

  const hasRowSelection = useCallback(
    () => hasSelectedRows(selectionAtom.get()),
    [selectionAtom],
  );

  const handleToggleRowSelection = useLatestCallback((rowId: string) => {
    table.setRowSelection((current) => {
      if (!current[rowId]) return { ...current, [rowId]: true };
      const { [rowId]: _removed, ...rest } = current;
      return rest;
    });
  });

  const cursorRowId = keyboard?.cursorRowId ?? null;
  const rowShortcuts = useMemo(
    () =>
      keyboard?.rowShortcuts?.map((shortcut) => ({
        key: shortcut.key,
        mod: shortcut.mod,
        alt: shortcut.alt,
        run: (rowId: string) => {
          const row = tableData.find((entry) => (entry as { id?: string }).id === rowId);
          if (row) shortcut.run(row);
        },
      })),
    [keyboard?.rowShortcuts, tableData],
  );
  useDataTableRowCursor({
    enabled: !!keyboard?.enabled && !alternateView?.active,
    rowIds: cursorRowIds,
    onTogglePin: name ? handleTogglePin : undefined,
    cursorRowId,
    onCursorRowIdChange: keyboard?.onCursorRowIdChange ?? noop,
    expandedRowId: expansion?.expandedRowId ?? null,
    onExpandedRowIdChange: expansion?.onExpandedRowIdChange,
    hasSelection: hasRowSelection,
    onToggleSelect: enableRowSelection ? handleToggleRowSelection : undefined,
    onClearSelection: handleClearSelection,
    shortcuts: rowShortcuts,
  });

  const viewportRef = useRef<HTMLDivElement | null>(null);
  const attachViewport = useCallback((element: HTMLDivElement | null) => {
    viewportRef.current = element;
    if (!element || typeof ResizeObserver === "undefined") return;
    let width = -1;
    const apply = () => {
      const next = element.clientWidth;
      if (next === width) return;
      width = next;
      element.style.setProperty("--dt-viewport-w", `${next}px`);
    };
    apply();
    const observer = new ResizeObserver(apply);
    observer.observe(element);
    return () => {
      observer.disconnect();
      if (viewportRef.current === element) viewportRef.current = null;
    };
  }, []);

  useEffect(() => {
    const focusId = cursorRowId ?? expansion?.expandedRowId ?? null;
    if (!focusId) return;
    const row = viewportRef.current?.querySelector<HTMLElement>(
      `tr[data-row-index][id="${CSS.escape(focusId)}"]`,
    );
    row?.scrollIntoView?.({ block: "nearest" });
  }, [cursorRowId, expansion?.expandedRowId]);

  const toggleExpandedRow = useLatestCallback((row: Row<TData>) => {
    if (!expansion) return;
    expansion.onExpandedRowIdChange(expansion.expandedRowId === row.id ? null : row.id);
    keyboard?.onCursorRowIdChange(row.id);
  });
  const handleRowClick = onRowClick ?? (expansion ? toggleExpandedRow : undefined);

  const handleApplyConfig = useCallback(
    (config: TableConfig, source?: TableViewSource) => {
      const newFieldFilters = config.fieldFilters ?? [];
      const newFilterGroups = (config.filterGroups ?? []).filter((g) => g.filters?.length > 0);
      const newSort = config.sort ?? [];
      const newPageSize = config.pageSize ?? pageSize;
      const newColumnVisibility = config.columnVisibility ?? {};
      const newColumnOrder = config.columnOrder ?? [];
      const newColumnSizing = config.columnSizing ?? {};
      const newColumnPinning = withRequiredPinning(
        config.columnPinning ?? EMPTY_PINNING,
        initialColumnPinning,
      );
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
    [setSearchParams, applyFilterState, pageSize, table, initialColumnPinning],
  );

  const layout = useDataTableLayout({
    resource: name,
    table,
    density,
    formatRules,
    activeViewId: activeView?.id ?? null,
    pinnedRowsCollapsed,
    hideChangesSinceLastVisit,
    hideTotals,
    virtualized,
  });

  const restoreView = useLatestCallback(async (id: string) => {
    try {
      const view = await queryClient.fetchQuery({
        ...queries.tableConfiguration.detail(id),
        staleTime: 60_000,
      });
      if (view) setActiveView({ id: view.id, name: view.name, config: view.tableConfig });
    } catch {
      setActiveView((current) => (current?.id === id ? null : current));
    }
  });

  const applyLayout = useLatestCallback((saved: TableLayout) => {
    table.setColumnVisibility({ ...initialColumnVisibility, ...saved.columnVisibility });
    table.setColumnOrder(saved.columnOrder);
    table.setColumnSizing(saved.columnSizing);
    table.setColumnPinning(
      toColumnPinningState(withRequiredPinning(saved.columnPinning, initialColumnPinning)),
    );
    if (saved.density) setDensity(saved.density);
    setFormatRules(saved.formatRules);
    table.setRowPinning({ top: saved.pinnedRowIds, bottom: [] });
    setPinnedRowsCollapsed(saved.pinnedRowsCollapsed);
    setChangesSince(saved.lastSeenAt);
    setHideChangesSinceLastVisit(saved.hideChangesSinceLastVisit);
    setHideTotals(saved.hideTotals);
    setVirtualized(saved.virtualized);
    if (!saved.activeViewId) {
      setActiveView(null);
    } else if (saved.activeViewId !== defaultConfig?.id) {
      void restoreView(saved.activeViewId);
    }
  });

  // The table opens once, before it is drawn: the default view when the address
  // asks for nothing else, then the person's own arrangement over it.
  const defaultConfigSettled = !name || !defaultConfigPending;
  useLayoutEffect(() => {
    if (initialStateApplied || !layout.settled || !defaultConfigSettled) return;
    const addressIsClean =
      pageIndex === 1 &&
      fieldFilters.length === 0 &&
      filterGroups.length === 0 &&
      sort.length === 0 &&
      query === "";
    if (defaultConfig?.tableConfig && addressIsClean) {
      handleApplyConfig(defaultConfig.tableConfig, {
        id: defaultConfig.id,
        name: defaultConfig.name,
      });
    }
    const saved = layout.readSaved();
    if (saved) applyLayout(saved);
    setInitialStateApplied(true);
    layout.arm();
  }, [
    initialStateApplied,
    layout,
    defaultConfigSettled,
    defaultConfig,
    pageIndex,
    fieldFilters.length,
    filterGroups.length,
    sort.length,
    query,
    handleApplyConfig,
    applyLayout,
  ]);

  const handleResetLayout = useCallback(() => {
    void layout.reset(() => {
      table.setColumnVisibility(initialColumnVisibility ?? {});
      table.setColumnOrder([]);
      table.setColumnSizing({});
      table.setColumnPinning(
        toColumnPinningState(withRequiredPinning(EMPTY_PINNING, initialColumnPinning)),
      );
      setDensity(initialDensity);
      setFormatRules([]);
      table.setRowPinning({ top: [], bottom: [] });
      setPinnedRowsCollapsed(false);
      setHideChangesSinceLastVisit(false);
      setHideTotals(false);
      setVirtualized(false);
      setActiveView(null);
      if (defaultConfig?.tableConfig) {
        handleApplyConfig(defaultConfig.tableConfig, {
          id: defaultConfig.id,
          name: defaultConfig.name,
        });
      }
    });
  }, [
    layout,
    table,
    initialColumnVisibility,
    initialColumnPinning,
    initialDensity,
    defaultConfig,
    handleApplyConfig,
  ]);

  const {
    columnVisibility: liveColumnVisibility,
    columnOrder: liveColumnOrder,
    columnPinning: liveColumnPinning,
  } = table.state;
  // Mid-drag the shell is not told each width; it compares a view against the
  // widths as they last settled, which is what a drag leaves behind.
  const liveColumnSizing = table.state.columnSizing ?? table.atoms.columnSizing.get();

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

  // Bulk edit is offered where the server has an editor for the table and the person
  // can change its rows; the server checks both again when the edit starts.
  const bulkEditFieldsQuery = useQuery({
    ...queries.bulkEdit.fields(resource ?? ""),
    enabled: !!resource && canUpdate && !!enableRowSelection,
    staleTime: Infinity,
    retry: false,
  });
  const canBulkEdit = (bulkEditFieldsQuery.data?.length ?? 0) > 0;
  const [bulkEditSelection, setBulkEditSelection] = useState<BulkEditSelectionInput | null>(null);
  const [bulkEditJobId, setBulkEditJobId] = useState<string | null>(null);
  const editSelectedRows = useLatestCallback(() => {
    const ids = Object.keys(selectionAtom.get()).filter((id) => selectionAtom.get()[id]);
    if (ids.length > 0) setBulkEditSelection({ ids });
  });
  const editAllMatching = useLatestCallback(() => {
    setBulkEditSelection({
      filter: {
        query: baseQueryOptions.query || undefined,
        fieldFilters: baseQueryOptions.fieldFilters ?? [],
        filterGroups: baseQueryOptions.filterGroups ?? [],
      },
      options: insightOptions,
    });
  });
  const handleBulkEditDone = useLatestCallback((job: BulkEditJob) => {
    void queryClient.invalidateQueries({ queryKey: [queryKey] });
    if (job.status === "Completed" && job.failedCount === 0) table.setRowSelection({});
  });

  const resolvedDockActions = useMemo(() => {
    const actions: DockAction<TData>[] = [...dockActions];
    if (canBulkEdit) {
      actions.unshift({
        id: "bulk-edit",
        label: t("Edit"),
        icon: Edit05Icon,
        onClick: () => editSelectedRows(),
      });
    }
    if (canExport && dockActions.length > 0) {
      actions.push({
        id: "export-selected",
        label: t("Export"),
        icon: Download01Icon,
        onClick: (rows: TData[]) =>
          downloadCsv(
            buildCsv(rows, buildExportColumns(table.getAllLeafColumns(), true)),
            exportFilename(name),
          ),
      });
    }
    return actions;
  }, [canBulkEdit, canExport, dockActions, editSelectedRows, name, t, table]);

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
    const results = tableData;
    return results.find((row: TData) => (row as { id?: string }).id === panelEntityId) ?? null;
  }, [panelEntityId, panelMode, tableData]);

  const panelRow = listRow;

  // Column widths and pinned offsets are written onto the table element as CSS
  // variables rather than rendered, so a resize drag moves every cell without a
  // React render: the sizing atom is followed directly, and each shell render
  // (columns shown, ordered or pinned differently) writes them again.
  const tableElementRef = useRef<HTMLTableElement | null>(null);
  const writtenLayoutVarsRef = useRef<readonly string[]>([]);
  const writeColumnLayout = useLatestCallback(() => {
    const element = tableElementRef.current;
    if (!element) return;
    const { vars, totalSize } = columnLayout(table.getFlatHeaders());
    for (const name of writtenLayoutVarsRef.current) {
      if (!(name in vars)) element.style.removeProperty(name);
    }
    for (const name in vars) {
      element.style.setProperty(name, vars[name]);
    }
    element.style.minWidth = `${totalSize}px`;
    writtenLayoutVarsRef.current = Object.keys(vars);
  });
  useLayoutEffect(() => {
    writeColumnLayout();
  });
  const fitColumns = useLatestCallback((columnIds?: readonly string[]) => {
    const element = tableElementRef.current;
    if (!element) return;
    const columns = table
      .getVisibleLeafColumns()
      .filter(
        (column) => isColumnResizable(column) && (!columnIds || columnIds.includes(column.id)),
      );
    if (columns.length === 0) return;

    const fits = measureColumnFits(
      element,
      columns.map((column) => column.id),
    );
    const sizing: Record<string, number> = {};
    for (const column of columns) {
      const width = fits[column.id];
      if (width) sizing[column.id] = clampFittedWidth(column, width);
    }
    table.setColumnSizing((previous) => ({ ...previous, ...sizing }));
  });

  const { fillDown, pasteFromClipboard } = useDataTableCellWrites({ table, queryKey });

  const columnSizingAtom = table.atoms.columnSizing;
  useLayoutEffect(() => {
    const subscription = columnSizingAtom.subscribe(writeColumnLayout);
    return () => subscription.unsubscribe();
  }, [columnSizingAtom, writeColumnLayout]);

  const reorderableIdsKey = table
    .getVisibleLeafColumns()
    .map((col) => col.id)
    .join("\u0000");
  const reorderableIds = useMemo(
    () => (reorderableIdsKey ? reorderableIdsKey.split("\u0000") : []),
    [reorderableIdsKey],
  );

  // An empty page is drawn as the table it will become rather than as a
  // table with no rows, so the header row and the pager step aside for it.
  const isEmpty = !isLoadingPage && !dataQuery.isError && currentPageRowCount === 0;
  const hasCollapsedGroups = (grouping?.collapsedKeys.length ?? 0) > 0;
  const emptyColumns = useMemo(
    () => emptyTableColumns(table.getVisibleLeafColumns()),
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [table, liveColumnVisibility, liveColumnOrder],
  );
  const defaultCreate = resolvedAddRecordActions.find((action) => action.id === "default-create");

  return (
    <DataTableProvider
      isLoading={isLoadingPage}
      table={table}
      columns={tableColumns}
      isPanelOpen={isPanelOpen}
      panelMode={panelMode}
      panelRow={panelRow}
      getSelectedRows={getSelectedRows}
      fitColumns={fitColumns}
      fillDown={fillDown}
      pasteFromClipboard={pasteFromClipboard}
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
      <PageViewRegistration
        resource={resource ?? name}
        query={query}
        fieldFilters={baseQueryOptions.fieldFilters}
        filterGroups={filterGroups}
        sort={effectiveSort}
        selection={selectionAtom}
        columnVisibility={liveColumnVisibility}
        columnIds={table.getAllLeafColumns().map((column) => column.id)}
        rowCount={totalCount}
      />
      <DataTablePanelWrapper>
        <DataTablePanelContent>
          <div className="bleed:gap-0 bleed:min-h-0 flex size-full min-w-0 flex-col gap-2">
            <DataTableToolbar
              table={table}
              columns={columns}
              filterFields={filterFields}
              query={query}
              onSearchChange={handleSearchChange}
              filters={filterItems}
              onFiltersChange={handleFiltersChange}
              sort={sort}
              onSortChange={handleSortArrayChange}
              fieldFilters={fieldFilters ?? []}
              onAskApplied={handleAskApplied}
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
              hasSavedLayout={layout.hasSavedLayout}
              highlightChanges={!hideChangesSinceLastVisit}
              showTotals={!hideTotals}
              virtualized={virtualized}
              onVirtualizedChange={name && !grouping ? setVirtualized : undefined}
              onShowTotalsChange={
                name && hasTotals ? (on: boolean) => setHideTotals(!on) : undefined
              }
              onHighlightChangesChange={
                name ? (on: boolean) => setHideChangesSinceLastVisit(!on) : undefined
              }
              onFitColumns={fitColumns}
              onResetLayout={name ? handleResetLayout : undefined}
              exportContext={{
                permissionResource: resource,
                graphql,
                queryOptions: baseQueryOptions,
                currentPageRows: currentPageResults ?? [],
                totalCount,
              }}
              slots={toolbar}
            />
            <DataTableFilterChips
              filters={filterItems}
              onFiltersChange={handleFiltersChange}
              query={query}
              onClearQuery={() => handleSearchChange("")}
              extraChips={toolbar?.chips}
            />
            {unseenCount > 0 && (
              <DataTableChangesBanner count={unseenCount} onMarkAllSeen={markAllSeen} />
            )}
            {enableRowSelection && totalCount != null && (
              <DataTableSelectionBanner
                selection={selectionAtom}
                pageRowIds={pageRowIds}
                totalCount={totalCount}
                maxSelectable={BULK_SELECT_MAX}
                onEditAllMatching={canBulkEdit ? editAllMatching : undefined}
                isSelectingAll={isSelectingAll}
                onSelectAllMatching={handleSelectAllMatching}
                onClearSelection={handleClearSelection}
              />
            )}
            {alternateView?.active ? (
              <div
                data-density={density}
                className={cn(
                  "bleed:min-h-0 bleed:flex-1 relative min-w-0",
                  density === "compact" && DENSITY_COMPACT_ROW,
                )}
              >
                {alternateView.render({ queryOptions: baseQueryOptions })}
              </div>
            ) : isEmpty && !hasCollapsedGroups ? (
              <div className="border-border bleed:rounded-none bleed:border-0 rounded-lg border">
                {renderEmptyState ? (
                  renderEmptyState({ hasActiveFilters, onClearFilters: handleClearFilters })
                ) : (
                  <DataTableEmptyState
                    columns={emptyColumns}
                    hasActiveFilters={hasActiveFilters}
                    title={emptyTitle}
                    onClearFilters={handleClearFilters}
                    addRecord={hasActiveFilters ? undefined : defaultCreate}
                  />
                )}
              </div>
            ) : (
              <div ref={attachViewport} className="bleed:min-h-0 bleed:flex-1 relative min-w-0">
                <DataTableRefreshPill
                  visible={liveRefresh.hasPendingUpdate}
                  onRefresh={liveRefresh.applyStaged}
                  onDismiss={liveRefresh.dismissStaged}
                />
                <DndContext
                  sensors={sensors}
                  collisionDetection={closestCenter}
                  modifiers={COLUMN_DRAG_MODIFIERS}
                  onDragEnd={handleColumnDragEnd}
                >
                  <Table
                    role="grid"
                    aria-label={name}
                    aria-rowcount={
                      totalCount != null ? totalCount + 1 + table.getTopRows().length : -1
                    }
                    aria-colcount={visibleLeafColumns.length}
                    aria-multiselectable
                    aria-busy={isLoadingPage || undefined}
                    data-density={density}
                    className={cn(
                      "border-separate border-spacing-0",
                      // Density repoints the row-height token; the cells read it,
                      // so the two densities stay the same table at two sizes
                      // rather than one table with padding patched over it.
                      density === "compact" && cn(DENSITY_COMPACT_ROW, "[&_td]:py-0.5"),
                    )}
                    // The vertical track starts below the sticky column header, so the
                    // scrollbar runs beside the rows and never over the header.
                    containerClassName="bleed:h-full bleed:max-h-none bleed:rounded-none bleed:border-0 max-h-[calc(65vh_-_var(--top-bar-height))] rounded-lg border border-border [&>[data-slot=scroll-area-scrollbar][data-orientation=vertical]]:top-(--row-head-h)!"
                    ref={tableElementRef}
                  >
                    <TableHeader className="sticky top-0 z-20">
                      {table.getHeaderGroups().map((headerGroup) => (
                        <TableRow
                          key={headerGroup.id}
                          aria-rowindex={1}
                          className="hover:bg-transparent"
                        >
                          <SortableContext
                            items={reorderableIds}
                            strategy={horizontalListSortingStrategy}
                          >
                            {headerGroup.headers.map((header) => (
                              <DataTableHeaderCell
                                key={header.id}
                                header={header}
                                facets={headerFacets}
                                facetField={
                                  headerFacets
                                    ? columnFacetField(
                                        header.column,
                                        filterFields,
                                        headerFacets.facetable,
                                      )
                                    : null
                                }
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
                      isLoading={isLoadingPage}
                      contextMenuActions={contextMenuActions}
                      onRowClick={handleRowClick}
                      getFormatClass={compiledFormatRules}
                      grouping={grouping}
                      loadingGroupKeys={loadingGroupKeys}
                      expansion={expansion}
                      cursorRowId={cursorRowId}
                      isFirstPage={zeroBasedPageIndex === 0}
                      isLastPage={!cursorPageInfo?.hasNextPage}
                      rowChanges={liveRefresh.changes}
                      pinnedRowsCollapsed={pinnedRowsCollapsed}
                      changesSince={showChanges ? changesSince : 0}
                      seenRowIds={seenRowIds}
                      onPinnedRowsCollapsedChange={setPinnedRowsCollapsed}
                      rowOffset={zeroBasedPageIndex * pageSize}
                      virtualized={virtualized}
                      tableRef={tableElementRef}
                      estimatedRowHeight={density === "compact" ? 30 : 40}
                    />
                    {showTotalsRow ? (
                      <DataTableTotalsRow
                        columns={visibleLeafColumns}
                        totals={insights.totals}
                        loading={insights.totalsLoading}
                        summable={insights.summable}
                      />
                    ) : null}
                  </Table>
                </DndContext>
              </div>
            )}
            {isEmpty || alternateView?.active ? null : (
              <DataTablePagination
                table={table}
                onPageChange={handlePageChange}
                onPageSizeChange={handlePageSizeChange}
                mode="cursor"
                hasNextPage={cursorPageInfo?.hasNextPage}
                currentPageRowCount={currentPageRowCount}
                totalCount={totalCount}
                pageSizeOptions={pageSizeOptions}
                leading={footerLeading}
              />
            )}
          </div>
        </DataTablePanelContent>
        {TablePanel && (
          <RecordPresence
            resource={resource}
            recordId={isPanelOpen && panelMode === "edit" ? (panelRow?.id as string | undefined) : null}
          >
            <TablePanel
              open={isPanelOpen}
              onOpenChange={handlePanelOpenChange}
              mode={panelMode}
              row={panelRow}
            />
          </RecordPresence>
        )}
      </DataTablePanelWrapper>
      {resource && bulkEditSelection ? (
        <BulkEditDialog
          open
          onOpenChange={(open) => {
            if (!open) setBulkEditSelection(null);
          }}
          resource={resource}
          selection={bulkEditSelection}
          onStarted={(job) => setBulkEditJobId(job.id)}
        />
      ) : null}
      {bulkEditJobId ? (
        <BulkEditProgress
          jobId={bulkEditJobId}
          onDismiss={() => setBulkEditJobId(null)}
          onDone={handleBulkEditDone}
        />
      ) : null}
      {enableRowSelection && resolvedDockActions.length > 0 && (
        <DataTableDock
          table={table}
          actions={resolvedDockActions}
          totals={selectionTotals}
        />
      )}
    </DataTableProvider>
  );
}

/**
 * Tells the page view what the table shows. It follows the selection itself, so
 * ticking a row updates what the assistant reads without redrawing the table.
 */
function PageViewRegistration(props: Parameters<typeof usePageViewRegistration>[0]) {
  usePageViewRegistration(props);
  return null;
}
