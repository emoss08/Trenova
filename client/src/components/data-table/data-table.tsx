"use no memo";
import { DataTableProvider } from "@/contexts/data-table-context";
import { useDataTableQuery } from "@/hooks/data-table/use-data-table-query";
import { searchParamsParser } from "@/hooks/data-table/use-data-table-state";
import { useDebounce } from "@/hooks/use-debounce";
import { usePermissions } from "@/hooks/use-permission";
import {
  initializeFilterItemsFromFieldFilters,
  initializeFilterItemsFromFilterGroups,
  updateSortField,
} from "@/lib/data-table";
import { api } from "@/lib/api";
import { queries } from "@/lib/queries";
import type {
  DataTableProps,
  FilterGroupItem,
  FilterItem,
  PanelMode,
  SingleFilterItem,
  SortDirection,
  SortField,
} from "@/types/data-table";
import type { TableConfig } from "@/types/table-configuration";
import { useQuery } from "@tanstack/react-query";
import {
  flexRender,
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
import { Table, TableHead, TableHeader, TableRow } from "../ui/table";
import { DataTablePagination } from "./_components/data-table-pagination";
import { DataTableBody } from "./data-table-body";
import { DataTableColumnHeader } from "./data-table-column-header";
import { DataTableDock } from "./data-table-dock";
import { DataTablePanelContent, DataTablePanelWrapper } from "./data-table-panel";
import { createSelectionColumn } from "./data-table-selection-column";
import { DataTableToolbar } from "./data-table-toolbar";

export function DataTable<TData extends Record<string, any>>({
  columns,
  name,
  link,
  detailLink,
  queryKey,
  resource,
  enableRowSelection = false,
  dockActions = [],
  TablePanel,
  onAddRecord: onAddRecordProp,
  addRecordActions = [],
  extraSearchParams,
  contextMenuActions,
  onRowClick,
  preferDetailRowForEdit = false,
  enableCreateAction = true,
  enableReadOnlyPanel = false,
  initialColumnVisibility,
  graphql,
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
  const defaultConfigAppliedRef = useRef(false);

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

  useEffect(() => {
    const singles = debouncedFilters.filter((f) => f.type === "filter") as SingleFilterItem[];
    const groups = debouncedFilters.filter((f) => f.type === "group") as FilterGroupItem[];

    const newFieldFilters = singles.map((f) => ({
      field: f.apiField,
      operator: f.operator,
      value: f.value,
    }));

    void setSearchParams({
      fieldFilters: newFieldFilters,
      filterGroups: groups.map((g) => ({
        filters: g.items.map((i) => ({
          field: i.apiField,
          operator: i.operator,
          value: i.value,
        })),
      })),
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
      };
      if (graphql) {
        nextParams.pageIndex = 1;
      }

      void setSearchParams(nextParams);
    },
    [graphql, sort, setSearchParams],
  );

  const handleSortArrayChange = useCallback(
    (newSort: SortField[]) => {
      const nextParams: { sort: SortField[]; pageIndex?: number } = {
        sort: newSort,
      };
      if (graphql) {
        nextParams.pageIndex = 1;
      }

      void setSearchParams(nextParams);
    },
    [graphql, setSearchParams],
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
  const cursorScopeKey = useMemo(
    () =>
      JSON.stringify({
        pageSize,
        query,
        fieldFilters,
        filterGroups,
        sort,
        variables: graphql?.variables ?? null,
    }),
    [fieldFilters, filterGroups, graphql?.variables, pageSize, query, sort],
  );
  const useGraphQLOffsetPagination = Boolean(graphql && sort.length > 0);
  const pageCursors =
    graphql && !useGraphQLOffsetPagination && cursorState.scopeKey === cursorScopeKey
      ? cursorState.cursors
      : { 0: null };
  const currentCursor =
    graphql && !useGraphQLOffsetPagination ? pageCursors[zeroBasedPageIndex] : null;
  const canFetchPage =
    !graphql ||
    useGraphQLOffsetPagination ||
    zeroBasedPageIndex === 0 ||
    currentCursor !== undefined;

  const dataQuery = useDataTableQuery<TData>(
    queryKey,
    link,
    { pageIndex: zeroBasedPageIndex, pageSize },
    { query, fieldFilters, filterGroups, sort, cursor: currentCursor, extraSearchParams },
    graphql,
    canFetchPage,
  );

  useEffect(() => {
    const pageInfo = dataQuery.data?.pageInfo;
    if (!graphql || pageInfo?.mode !== "cursor" || !pageInfo.hasNextPage || !pageInfo.endCursor) {
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
  }, [cursorScopeKey, dataQuery.data?.pageInfo, graphql, zeroBasedPageIndex]);

  useEffect(() => {
    if (!graphql || canFetchPage || pageIndex === 1) {
      return;
    }

    void setSearchParams({ pageIndex: 1 });
  }, [canFetchPage, graphql, pageIndex, setSearchParams]);

  const cursorPageInfo =
    graphql && dataQuery.data?.pageInfo?.mode === "cursor" ? dataQuery.data.pageInfo : null;
  const currentPageRowCount = dataQuery.data?.results.length ?? 0;
  const rowCount = cursorPageInfo
    ? zeroBasedPageIndex * pageSize +
      currentPageRowCount +
      (cursorPageInfo.hasNextPage ? pageSize : 0)
    : (dataQuery.data?.count ?? 0);
  const pageCount = cursorPageInfo
    ? zeroBasedPageIndex + 1 + (cursorPageInfo.hasNextPage ? 1 : 0)
    : Math.ceil((dataQuery.data?.count ?? 0) / pageSize);

  // eslint-disable-next-line react-hooks/incompatible-library
  const table = useReactTable({
    getCoreRowModel: getCoreRowModel(),
    getPaginationRowModel: getPaginationRowModel(),
    getFilteredRowModel: getFilteredRowModel(),
    getSortedRowModel: getSortedRowModel(),
    data: dataQuery.data?.results || [],
    columns: tableColumns,
    pageCount,
    rowCount,
    manualPagination: true,
    manualFiltering: true,
    columnResizeMode: "onChange",
    getRowId: (row) => row.id,
    manualSorting: true,
    enableRowSelection,
    enableMultiRowSelection: true,
    onRowSelectionChange: setRowSelection,
    initialState: initialColumnVisibility
      ? { columnVisibility: initialColumnVisibility }
      : undefined,
    state: {
      pagination: {
        pageIndex: zeroBasedPageIndex,
        pageSize,
      },
      rowSelection,
    },
    onPaginationChange: (updater) => {
      const newState =
        typeof updater === "function" ? updater({ pageIndex: zeroBasedPageIndex, pageSize }) : updater;
      handlePageChange(newState.pageIndex);
      if (newState.pageSize !== pageSize) {
        handlePageSizeChange(newState.pageSize);
      }
    },
  });

  const handleApplyConfig = useCallback(
    (config: TableConfig) => {
      const newFieldFilters = config.fieldFilters ?? [];
      const newFilterGroups = (config.filterGroups ?? []).filter((g) => g.filters?.length > 0);

      const filterItemsFromFields = initializeFilterItemsFromFieldFilters(newFieldFilters, columns);
      const filterItemsFromGroups = initializeFilterItemsFromFilterGroups(newFilterGroups, columns);

      void setSearchParams({
        fieldFilters: newFieldFilters,
        filterGroups: newFilterGroups,
        sort: config.sort,
        pageSize: config.pageSize,
        pageIndex: 1,
      });

      setFilterItems([...filterItemsFromFields, ...filterItemsFromGroups]);

      if (config.columnVisibility) {
        table.setColumnVisibility(config.columnVisibility);
      }
      if (config.columnOrder && config.columnOrder.length > 0) {
        table.setColumnOrder(config.columnOrder);
      }
    },
    [setSearchParams, columns, table],
  );

  useEffect(() => {
    if (
      defaultConfig?.tableConfig &&
      !defaultConfigAppliedRef.current &&
      fieldFilters.length === 0 &&
      filterGroups.length === 0 &&
      sort.length === 0 &&
      query === ""
    ) {
      defaultConfigAppliedRef.current = true;
      handleApplyConfig(defaultConfig.tableConfig);
    }
  }, [
    defaultConfig,
    fieldFilters.length,
    filterGroups.length,
    sort.length,
    query,
    handleApplyConfig,
  ]);

  const currentConfig = useMemo<TableConfig>(() => {
    const columnVisibility: Record<string, boolean> = {};
    for (const col of table.getAllColumns()) {
      columnVisibility[col.id] = col.getIsVisible();
    }

    return {
      fieldFilters: fieldFilters,
      filterGroups: filterGroups,
      joinOperator: "and",
      sort,
      pageSize,
      columnVisibility,
      columnOrder: table.getState().columnOrder,
    };
  }, [fieldFilters, filterGroups, sort, pageSize, table]);

  const listRow = useMemo(() => {
    if (!panelEntityId || panelMode !== "edit") return null;
    const results = dataQuery.data?.results || [];
    return results.find((row: TData) => (row as { id?: string }).id === panelEntityId) ?? null;
  }, [panelEntityId, panelMode, dataQuery.data?.results]);

  const extraParams = extraSearchParams
    ? "?" +
      new URLSearchParams(
        Object.entries(extraSearchParams).map(([k, v]) => [k, String(v)]),
      ).toString()
    : "";

  const { data: detailRow } = useQuery({
    queryKey: [queryKey, "detail", detailLink ?? link, extraParams, panelEntityId],
    queryFn: () => api.get<TData>(`${detailLink ?? link}${panelEntityId}/${extraParams}`),
    enabled: !!panelEntityId && panelMode === "edit" && (preferDetailRowForEdit || !listRow),
    staleTime: 0,
  });

  const panelRow = preferDetailRowForEdit ? (detailRow ?? null) : (listRow ?? detailRow ?? null);

  const { columnSizeVars, totalSize } = useMemo(() => {
    const headers = table.getFlatHeaders();
    const vars: Record<string, string> = {};
    let total = 0;
    for (const header of headers) {
      const size = header.getSize();
      vars[`--col-${header.column.id.replace(".", "-")}-size`] = `${size}px`;
      total += size;
    }
    return { columnSizeVars: vars, totalSize: total };
  }, [table]);

  return (
    <DataTableProvider
      isLoading={dataQuery.isLoading}
      table={table}
      columns={tableColumns}
      isPanelOpen={isPanelOpen}
      panelMode={panelMode}
      panelRow={panelRow}
      rowSelection={rowSelection}
      openPanelCreate={openPanelCreate}
      openPanelEdit={openPanelEdit}
      closePanel={closePanel}
      hasPanel={hasPanel}
      canOpenPanel={canUpdate || enableReadOnlyPanel}
      canCreate={canCreate}
      canUpdate={canUpdate}
      canExport={canExport}
      pagination={{
        pageIndex: pageIndex - 1,
        pageSize: pageSize,
      }}
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
            />
            <Table
              className="border-separate border-spacing-0"
              containerClassName="max-h-[calc(65vh_-_var(--top-bar-height))] rounded-md border border-border"
              style={{ ...columnSizeVars, minWidth: `${totalSize}px` }}
            >
              <TableHeader className="sticky top-0 z-10 bg-muted backdrop-blur-sm">
                {table.getHeaderGroups().map((headerGroup) => (
                  <TableRow key={headerGroup.id} className="hover:bg-transparent">
                    {headerGroup.headers.map((header) => {
                      const meta = header.column.columnDef.meta;
                      const isSortable = meta?.sortable !== false;
                      return (
                        <TableHead
                          key={header.id}
                          className="border-b border-border"
                          style={{
                            width: `var(--col-${header.column.id.replace(".", "-")}-size)`,
                          }}
                        >
                          {header.isPlaceholder ? null : isSortable ? (
                            <DataTableColumnHeader
                              column={header.column}
                              title={
                                typeof header.column.columnDef.header === "string"
                                  ? header.column.columnDef.header
                                  : meta?.label || header.column.id
                              }
                              currentSort={sort}
                              onSort={handleSortChange}
                            />
                          ) : (
                            flexRender(header.column.columnDef.header, header.getContext())
                          )}
                        </TableHead>
                      );
                    })}
                  </TableRow>
                ))}
              </TableHeader>
              <DataTableBody
                table={table}
                columns={tableColumns}
                isLoading={dataQuery.isLoading}
                contextMenuActions={contextMenuActions}
                onRowClick={onRowClick}
              />
            </Table>
            <DataTablePagination
              table={table}
              onPageChange={handlePageChange}
              onPageSizeChange={handlePageSizeChange}
              mode={cursorPageInfo ? "cursor" : "offset"}
              hasNextPage={cursorPageInfo?.hasNextPage}
              currentPageRowCount={currentPageRowCount}
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
