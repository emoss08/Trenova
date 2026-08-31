"use no memo";
import type { RowData } from "@tanstack/react-table";
import { Button } from "@trenova/shared/components/ui/button";
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@trenova/shared/components/ui/select";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { Spinner } from "@trenova/shared/components/ui/spinner";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@trenova/shared/components/ui/table";
import { useDebounce } from "@trenova/shared/hooks/use-debounce";
import { useOrgCapabilities } from "@trenova/shared/hooks/use-org-capabilities";
import {
  convertFilterItemsToFieldFilters,
  convertFilterItemsToFilterGroups,
  initializeFilterItemsFromFieldFilters,
  initializeFilterItemsFromFilterGroups,
} from "@/lib/data-table";
import { listShipmentsGraphQL } from "@/lib/graphql/shipment";
import { queries } from "@/lib/queries";
import { cn } from "@trenova/shared/lib/utils";
import type {
  FieldFilter,
  FilterItem,
  RowAction,
  ColumnDef,
  Row,
  Table as TanstackTable,
} from "@trenova/shared/types/data-table";
import type { Shipment } from "@trenova/shared/types/shipment";
import type { TableConfig } from "@/types/table-configuration";
import { useQuery } from "@tanstack/react-query";
import { flexRender, useTable, type ColumnVisibilityState } from "@tanstack/react-table";
import { dataTableFeatures } from "@trenova/shared/lib/table-features";
import { ChartGanttIcon, ChevronLeftIcon, ChevronRightIcon, TableIcon } from "lucide-react";
import { lazy, Suspense, useCallback, useEffect, useMemo, useRef, useState } from "react";
import type { ShipmentDocumentUploadContext } from "./expanded-row/document-stack";
import { PanelSkeleton } from "./expanded-row/panel-skeletons";
import { FilterChipRow } from "./filter-chip-row";
import { SavedViewsBar } from "./saved-views-bar";
import { useCommandCenterStore } from "./store";
import {
  isTimelineViewAvailable,
  PAGE_SIZE_OPTIONS,
  resolveCommandCenterViewMode,
  useCommandCenterUrl,
  type CommandCenterPageSize,
  type CommandCenterViewMode,
} from "./url-state";

const DataTableSearch = lazy(() => import("@/components/data-table/data-table-search"));
const DataTableFilterBuilder = lazy(
  () => import("@/components/data-table/data-table-filter-builder"),
);
const DataTableViewOptions = lazy(() => import("@/components/data-table/data-table-view-options"));
const DataTableConfigManager = lazy(
  () => import("@/components/data-table/data-table-config-manager"),
);
const DataTableSaveConfigDialog = lazy(() =>
  import("@/components/data-table/data-table-save-config-dialog").then((m) => ({
    default: m.DataTableSaveConfigDialog,
  })),
);
const ExpandedRow = lazy(() => import("./expanded-row").then((m) => ({ default: m.ExpandedRow })));
const CommandCenterTimeline = lazy(() => import("./timeline"));

function ToolbarButtonSkeleton() {
  return <Skeleton className="h-7 w-20" />;
}

function SearchSkeleton() {
  return <Skeleton className="h-7 w-48" />;
}

function TimelineLoadingFallback() {
  return (
    <div className="flex h-[clamp(420px,58vh,640px)] flex-col gap-2 p-3">
      {Array.from({ length: 8 }).map((_, index) => (
        <Skeleton key={index} className="h-10 w-full" />
      ))}
    </div>
  );
}

function ExpandedRowLoadingFallback() {
  return (
    <div className="grid grid-cols-1 gap-5 px-4 py-3 md:grid-cols-[2fr_1.4fr_1fr_1fr]">
      {Array.from({ length: 4 }).map((_, index) => (
        <PanelSkeleton key={index} />
      ))}
    </div>
  );
}

const SKELETON_CELL_WIDTHS = ["w-3/4", "w-1/2", "w-2/3", "w-3/5"] as const;

function TableBodySkeleton({ columnCount, rowCount }: { columnCount: number; rowCount: number }) {
  return (
    <>
      {Array.from({ length: rowCount }).map((_, rowIndex) => (
        <tr
          key={rowIndex}
          data-testid="command-center-skeleton-row"
          className="border-border/70 h-9 border-b"
        >
          {Array.from({ length: columnCount }).map((_, columnIndex) => (
            <td key={columnIndex} className="px-2.5 py-1.5 align-middle">
              <Skeleton
                className={cn(
                  "h-3.5",
                  SKELETON_CELL_WIDTHS[(rowIndex + columnIndex) % SKELETON_CELL_WIDTHS.length],
                )}
              />
            </td>
          ))}
        </tr>
      ))}
    </>
  );
}

const QUERY_KEY = "shipment-list";
const RESOURCE_NAME = "Shipment";

type CommandCenterTableProps = {
  columns: ColumnDef<Shipment>[];
  rowActions: RowAction<Shipment>[];
  mandatoryFieldFilters: FieldFilter[];
  onUploadDocument: (shipment: Shipment, context?: ShipmentDocumentUploadContext) => void;
  onSummaryChange?: (summary: CommandCenterTableSummary) => void;
};

export type CommandCenterTableSummary = {
  totalCount: number;
  dataUpdatedAt: number;
  backgroundQueriesEnabled: boolean;
};

export function CommandCenterTable({
  columns,
  rowActions,
  mandatoryFieldFilters,
  onUploadDocument,
  onSummaryChange,
}: CommandCenterTableProps) {
  const [
    { mode: requestedViewMode, expanded: expandedId, page, size: pageSize, q: query },
    setUrl,
  ] = useCommandCenterUrl();
  const capabilities = useOrgCapabilities();
  const timelineAvailable = isTimelineViewAvailable(capabilities);
  const viewMode = resolveCommandCenterViewMode(requestedViewMode, capabilities);
  const pageIndex = Math.max(0, page - 1);
  const setQuery = (next: string) => void setUrl({ q: next.length === 0 ? null : next, page: 1 });
  const setPageIndex = (next: number) => void setUrl({ page: next + 1 });
  const setPageSize = (next: CommandCenterPageSize) =>
    void setUrl({ size: next === 10 ? null : next, page: 1 });
  const setViewMode = (next: CommandCenterViewMode) => void setUrl({ mode: next });
  const toggleExpandedId = (id: string) => void setUrl({ expanded: expandedId === id ? null : id });

  const highlightId = useCommandCenterStore.use.highlightId();
  const setHighlightId = useCommandCenterStore.use.setHighlightId();

  const [filterItems, setFilterItems] = useState<FilterItem[]>([]);
  const [columnVisibility, setColumnVisibility] = useState<ColumnVisibilityState>({
    pickupAppointment: false,
    deliveryAppointment: false,
  });
  const [columnOrder, setColumnOrder] = useState<string[]>([]);
  const [saveDialogOpen, setSaveDialogOpen] = useState(false);
  const cursorCacheRef = useRef(new Map<string, Map<number, string | null>>());

  const debouncedFilterItems = useDebounce(filterItems, 300);

  const userFieldFilters = useMemo(
    () => convertFilterItemsToFieldFilters(debouncedFilterItems) ?? [],
    [debouncedFilterItems],
  );
  const userFilterGroups = useMemo(
    () => convertFilterItemsToFilterGroups(debouncedFilterItems),
    [debouncedFilterItems],
  );

  const mergedFieldFilters = useMemo(
    () => [...mandatoryFieldFilters, ...userFieldFilters],
    [mandatoryFieldFilters, userFieldFilters],
  );

  useEffect(() => {
    void setUrl({ page: 1 });
  }, [mergedFieldFilters, userFilterGroups, query, setUrl]);

  const cursorCacheKey = useMemo(
    () =>
      JSON.stringify({
        pageSize,
        query,
        fieldFilters: mergedFieldFilters,
        filterGroups: userFilterGroups,
      }),
    [pageSize, query, mergedFieldFilters, userFilterGroups],
  );

  const pageCursor = useMemo(() => {
    if (pageIndex === 0) return null;
    return cursorCacheRef.current.get(cursorCacheKey)?.get(pageIndex - 1);
  }, [cursorCacheKey, pageIndex]);

  useEffect(() => {
    if (pageIndex > 0 && pageCursor === undefined) {
      void setUrl({ page: 1 });
    }
  }, [pageCursor, pageIndex, setUrl]);

  const canFetchPage = pageIndex === 0 || pageCursor !== undefined;

  const dataQuery = useQuery({
    queryKey: [
      QUERY_KEY,
      "command-center",
      { pageIndex, pageSize },
      pageCursor,
      mergedFieldFilters,
      userFilterGroups,
      query,
    ],
    queryFn: () =>
      listShipmentsGraphQL({
        limit: pageSize,
        after: pageCursor ?? null,
        query,
        fieldFilters: mergedFieldFilters,
        filterGroups: userFilterGroups,
      }),
    placeholderData: (prev) => prev,
    enabled: canFetchPage && viewMode === "table",
  });

  useEffect(() => {
    const endCursor = dataQuery.data?.pageInfo?.endCursor ?? null;
    if (!dataQuery.data || !endCursor) return;

    let pageCursors = cursorCacheRef.current.get(cursorCacheKey);
    if (!pageCursors) {
      pageCursors = new Map<number, string | null>();
      cursorCacheRef.current.set(cursorCacheKey, pageCursors);
    }
    pageCursors.set(pageIndex, endCursor);
  }, [cursorCacheKey, dataQuery.data, pageIndex]);

  const totalCount = dataQuery.data?.count ?? 0;
  const totalPages = Math.max(1, Math.ceil(totalCount / pageSize));
  const rows = (dataQuery.data?.results ?? []) as Shipment[];
  const isInitialLoading = dataQuery.isPending;
  const backgroundQueriesEnabled = dataQuery.isSuccess && !dataQuery.isFetching;

  const [timelineSummary, setTimelineSummary] = useState<CommandCenterTableSummary | null>(null);
  const handleTimelineSummaryChange = useCallback(
    (summary: CommandCenterTableSummary) => {
      setTimelineSummary(summary);
      onSummaryChange?.(summary);
    },
    [onSummaryChange],
  );

  useEffect(() => {
    if (!dataQuery.data || viewMode !== "table") return;
    onSummaryChange?.({
      totalCount,
      dataUpdatedAt: dataQuery.dataUpdatedAt,
      backgroundQueriesEnabled,
    });
  }, [
    backgroundQueriesEnabled,
    dataQuery.data,
    dataQuery.dataUpdatedAt,
    onSummaryChange,
    totalCount,
    viewMode,
  ]);

  const table = useTable({
    features: dataTableFeatures,
    data: rows,
    columns,
    state: { columnVisibility, columnOrder },
    onColumnVisibilityChange: setColumnVisibility,
    onColumnOrderChange: setColumnOrder,
    manualPagination: true,
    pageCount: totalPages,
    rowCount: totalCount,
    getRowId: (row) => row.id ?? "",
  });

  const { data: defaultConfig } = useQuery({
    ...queries.tableConfiguration.default(RESOURCE_NAME),
    retry: false,
    staleTime: Infinity,
  });
  const appliedDefaultRef = useRef(false);

  const handleApplyConfig = useMemo(
    () => (config: TableConfig) => {
      const fieldFilters = config.fieldFilters ?? [];
      const filterGroups = (config.filterGroups ?? []).filter((g) => g.filters?.length > 0);
      const fromFields = initializeFilterItemsFromFieldFilters(fieldFilters, columns);
      const fromGroups = initializeFilterItemsFromFilterGroups(filterGroups, columns);
      setFilterItems([...fromFields, ...fromGroups]);
      if (config.columnVisibility) setColumnVisibility(config.columnVisibility);
      if (config.columnOrder?.length) setColumnOrder(config.columnOrder);
      void setUrl({ page: 1 });
    },
    [columns, setUrl],
  );

  useEffect(() => {
    if (defaultConfig?.tableConfig && !appliedDefaultRef.current && filterItems.length === 0) {
      appliedDefaultRef.current = true;
      handleApplyConfig(defaultConfig.tableConfig);
    }
  }, [defaultConfig, filterItems.length, handleApplyConfig]);

  const currentConfig = useMemo<TableConfig>(() => {
    const visibility: Record<string, boolean> = {};
    for (const col of table.getAllLeafColumns()) {
      visibility[col.id] = columnVisibility[col.id] ?? true;
    }
    return {
      fieldFilters: userFieldFilters,
      filterGroups: userFilterGroups,
      joinOperator: "and",
      sort: [],
      pageSize,
      columnVisibility: visibility,
      columnOrder,
      columnSizing: {},
      columnPinning: { left: [], right: [] },
      density: "comfortable",
      formatRules: [],
    };
  }, [userFieldFilters, userFilterGroups, pageSize, table, columnVisibility, columnOrder]);

  const handleRowClick = (row: Row<Shipment>) => {
    if (row.original.id) toggleExpandedId(row.original.id);
  };

  const rightSlot = (
    <>
      {timelineAvailable && <ViewModeToggle viewMode={viewMode} setViewMode={setViewMode} />}
      <Suspense fallback={<ToolbarButtonSkeleton />}>
        <DataTableConfigManager
          resource={RESOURCE_NAME}
          onApplyConfig={handleApplyConfig}
          onSaveConfig={() => setSaveDialogOpen(true)}
        />
      </Suspense>
    </>
  );

  const countsEnabled =
    viewMode === "table"
      ? backgroundQueriesEnabled
      : (timelineSummary?.backgroundQueriesEnabled ?? false);

  const tableBody = (
    <>
      <div className="relative overflow-x-auto">
        <Table>
          <colgroup>
            {table.getVisibleFlatColumns().map((col) => (
              <col key={col.id} style={{ width: `${col.getSize()}px` }} />
            ))}
          </colgroup>
          <TableHeader>
            {table.getHeaderGroups().map((hg) => (
              <TableRow key={hg.id}>
                {hg.headers.map((header) => (
                  <TableHead
                    key={header.id}
                    className="bg-muted"
                    style={{ width: `${header.getSize()}px` }}
                  >
                    {header.isPlaceholder
                      ? null
                      : flexRender(header.column.columnDef.header, header.getContext())}
                  </TableHead>
                ))}
              </TableRow>
            ))}
          </TableHeader>
          <TableBody>
            {isInitialLoading ? (
              <TableBodySkeleton
                columnCount={table.getVisibleFlatColumns().length}
                rowCount={Math.min(pageSize, 10)}
              />
            ) : table.getRowModel().rows.length === 0 ? (
              <TableRow>
                <TableCell colSpan={table.getVisibleFlatColumns().length}>
                  No shipments match the current view.
                </TableCell>
              </TableRow>
            ) : (
              table.getRowModel().rows.map((row) => {
                const isExpanded = expandedId === row.original.id;
                const isHighlighted =
                  !!highlightId && highlightId === row.original.id && !isExpanded;
                return (
                  <RowFragment
                    key={row.id}
                    row={row}
                    isExpanded={isExpanded}
                    isHighlighted={isHighlighted}
                    onClick={() => handleRowClick(row)}
                    onMouseEnter={() => row.original.id && setHighlightId(row.original.id)}
                    onMouseLeave={() => setHighlightId(null)}
                    rowActions={rowActions}
                    onUploadDocument={onUploadDocument}
                  />
                );
              })
            )}
          </TableBody>
        </Table>
        {dataQuery.isFetching && !isInitialLoading && (
          <div className="bg-background/70 text-muted-foreground pointer-events-none absolute top-2 right-2 inline-flex items-center gap-1 rounded px-2 py-1 text-[10px] backdrop-blur-sm">
            <Spinner className="size-3" />
            Refreshing
          </div>
        )}
      </div>

      <CommandCenterFooter
        totalCount={totalCount}
        pageIndex={pageIndex}
        totalPages={totalPages}
        rowCount={rows.length}
        pageSize={pageSize as CommandCenterPageSize}
        isLoading={isInitialLoading}
        onPageSizeChange={setPageSize}
        onPrev={() => setPageIndex(Math.max(0, pageIndex - 1))}
        onNext={() => setPageIndex(Math.min(totalPages - 1, pageIndex + 1))}
      />
    </>
  );

  return (
    <section className="border-border bg-card flex flex-col overflow-hidden rounded-md border">
      <SavedViewsBar rightSlot={rightSlot} countsEnabled={countsEnabled} />
      <div className="border-border flex items-center gap-2 border-b px-3 py-1.5">
        <Suspense fallback={<SearchSkeleton />}>
          <DataTableSearch value={query} onChange={setQuery} placeholder="Search shipments..." />
        </Suspense>
        <Suspense fallback={<ToolbarButtonSkeleton />}>
          <DataTableFilterBuilder
            columns={columns as ColumnDef<RowData>[]}
            filters={filterItems}
            onFiltersChange={setFilterItems}
          />
        </Suspense>
        <div className="bg-border mx-1 h-4 w-px" />
        <FilterChipRow />
        {viewMode === "table" && (
          <>
            {isInitialLoading ? (
              <Skeleton className="ml-auto h-3.5 w-24 shrink-0" />
            ) : (
              <p className="font-table text-muted-foreground ml-auto shrink-0 text-[10.5px] tabular-nums">
                {rows.length} of {totalCount} results
              </p>
            )}
            <Suspense fallback={<ToolbarButtonSkeleton />}>
              <DataTableViewOptions table={table as unknown as TanstackTable<RowData>} />
            </Suspense>
          </>
        )}
      </div>

      {viewMode === "timeline" ? (
        <Suspense fallback={<TimelineLoadingFallback />}>
          <CommandCenterTimeline
            fieldFilters={mergedFieldFilters}
            filterGroups={userFilterGroups}
            query={query}
            onSummaryChange={handleTimelineSummaryChange}
          />
        </Suspense>
      ) : (
        tableBody
      )}

      <Suspense fallback={null}>
        <DataTableSaveConfigDialog
          open={saveDialogOpen}
          onOpenChange={setSaveDialogOpen}
          resource={RESOURCE_NAME}
          currentConfig={currentConfig}
        />
      </Suspense>
    </section>
  );
}

function ViewModeToggle({
  viewMode,
  setViewMode,
}: {
  viewMode: "table" | "timeline";
  setViewMode: (m: "table" | "timeline") => void;
}) {
  return (
    <div
      role="group"
      aria-label="View mode"
      className="border-border inline-flex overflow-hidden rounded-md border"
    >
      <button
        type="button"
        onClick={() => setViewMode("table")}
        aria-pressed={viewMode === "table"}
        className={cn(
          "flex items-center gap-1 px-2 py-1 text-[11px] transition-colors",
          viewMode === "table"
            ? "bg-muted text-foreground"
            : "bg-background text-muted-foreground hover:text-foreground",
        )}
      >
        <TableIcon className="size-3" />
        Table
      </button>
      <button
        type="button"
        onClick={() => setViewMode("timeline")}
        aria-pressed={viewMode === "timeline"}
        className={cn(
          "border-border flex items-center gap-1 border-l px-2 py-1 text-[11px] transition-colors",
          viewMode === "timeline"
            ? "bg-muted text-foreground"
            : "bg-background text-muted-foreground hover:text-foreground",
        )}
      >
        <ChartGanttIcon className="size-3" />
        Timeline
      </button>
    </div>
  );
}

function RowFragment({
  row,
  isExpanded,
  isHighlighted,
  onClick,
  onMouseEnter,
  onMouseLeave,
  rowActions,
  onUploadDocument,
}: {
  row: Row<Shipment>;
  isExpanded: boolean;
  isHighlighted: boolean;
  onClick: () => void;
  onMouseEnter: () => void;
  onMouseLeave: () => void;
  rowActions: RowAction<Shipment>[];
  onUploadDocument: (shipment: Shipment, context?: ShipmentDocumentUploadContext) => void;
}) {
  return (
    <>
      <tr
        className={cn(
          "group/row border-border/70 hover:bg-muted/30 h-9 cursor-pointer border-b transition-colors",
          isExpanded && "bg-brand/10 outline-brand hover:bg-brand/20 outline-1 -outline-offset-1",
          isHighlighted && "bg-muted/50",
        )}
        onClick={onClick}
        onMouseEnter={onMouseEnter}
        onMouseLeave={onMouseLeave}
      >
        {row.getVisibleCells().map((cell) => (
          <td key={cell.id} className="px-2.5 py-1.5 align-middle text-[11.5px]">
            {flexRender(cell.column.columnDef.cell, cell.getContext())}
          </td>
        ))}
      </tr>
      {isExpanded && (
        <tr className="border-border bg-background border-b">
          <td colSpan={row.getVisibleCells().length} className="p-0">
            <div className="cc-fade-in">
              <Suspense fallback={<ExpandedRowLoadingFallback />}>
                <ExpandedRow
                  row={row}
                  shipment={row.original}
                  rowActions={rowActions}
                  onUploadDocument={onUploadDocument}
                />
              </Suspense>
            </div>
          </td>
        </tr>
      )}
    </>
  );
}

function CommandCenterFooter({
  totalCount,
  pageIndex,
  totalPages,
  rowCount,
  pageSize,
  isLoading,
  onPageSizeChange,
  onPrev,
  onNext,
}: {
  totalCount: number;
  pageIndex: number;
  totalPages: number;
  rowCount: number;
  pageSize: CommandCenterPageSize;
  isLoading: boolean;
  onPageSizeChange: (size: CommandCenterPageSize) => void;
  onPrev: () => void;
  onNext: () => void;
}) {
  return (
    <div className="border-border text-muted-foreground flex items-center justify-between border-t px-3 py-1.5 text-[11px]">
      {isLoading ? (
        <Skeleton className="h-3.5 w-44" />
      ) : (
        <p className="font-table tabular-nums">
          {rowCount} rows · page {pageIndex + 1} of {totalPages} · {totalCount} total
        </p>
      )}
      <div className="flex items-center gap-3">
        <div className="flex items-center gap-1.5">
          <span>Rows</span>
          <Select
            value={String(pageSize)}
            onValueChange={(value) => onPageSizeChange(Number(value) as CommandCenterPageSize)}
          >
            <SelectTrigger className="h-6 w-14.5 py-0 text-[11px]">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectGroup>
                {PAGE_SIZE_OPTIONS.map((size) => (
                  <SelectItem key={size} value={String(size)} className="text-[11px]">
                    {size}
                  </SelectItem>
                ))}
              </SelectGroup>
            </SelectContent>
          </Select>
        </div>
        <div className="flex items-center gap-1">
          <Button
            variant="ghost"
            size="icon-xs"
            aria-label="Previous page"
            disabled={pageIndex === 0}
            onClick={onPrev}
          >
            <ChevronLeftIcon className="size-3.5" />
          </Button>
          <span className="font-table tabular-nums">
            {pageIndex + 1} / {totalPages}
          </span>
          <Button
            variant="ghost"
            size="icon-xs"
            aria-label="Next page"
            disabled={pageIndex >= totalPages - 1}
            onClick={onNext}
          >
            <ChevronRightIcon className="size-3.5" />
          </Button>
        </div>
      </div>
    </div>
  );
}
