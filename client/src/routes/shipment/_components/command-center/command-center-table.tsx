"use no memo";
import { Button } from "@/components/ui/button";
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Skeleton } from "@/components/ui/skeleton";
import { Spinner } from "@/components/ui/spinner";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { useDebounce } from "@/hooks/use-debounce";
import {
  convertFilterItemsToFieldFilters,
  convertFilterItemsToFilterGroups,
  initializeFilterItemsFromFieldFilters,
  initializeFilterItemsFromFilterGroups,
} from "@/lib/data-table";
import { listShipmentsGraphQL } from "@/lib/graphql/shipment";
import { queries } from "@/lib/queries";
import { cn } from "@/lib/utils";
import type { FieldFilter, FilterItem, RowAction } from "@/types/data-table";
import type { Shipment } from "@/types/shipment";
import type { TableConfig } from "@/types/table-configuration";
import { useQuery } from "@tanstack/react-query";
import {
  flexRender,
  getCoreRowModel,
  useReactTable,
  type ColumnDef,
  type Row,
  type Table as TanstackTable,
  type VisibilityState,
} from "@tanstack/react-table";
import { ChartGanttIcon, ChevronLeftIcon, ChevronRightIcon, TableIcon } from "lucide-react";
import { lazy, Suspense, useCallback, useEffect, useMemo, useRef, useState } from "react";
import type { ShipmentDocumentUploadContext } from "./expanded-row/document-stack";
import { PanelSkeleton } from "./expanded-row/panel-skeletons";
import { FilterChipRow } from "./filter-chip-row";
import { SavedViewsBar } from "./saved-views-bar";
import { useCommandCenterStore } from "./store";
import {
  PAGE_SIZE_OPTIONS,
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
  const [{ mode: viewMode, expanded: expandedId, page, size: pageSize, q: query }, setUrl] =
    useCommandCenterUrl();
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
  const [columnVisibility, setColumnVisibility] = useState<VisibilityState>({
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

  const table = useReactTable({
    data: rows,
    columns,
    getCoreRowModel: getCoreRowModel(),
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
    if (
      defaultConfig?.tableConfig &&
      !appliedDefaultRef.current &&
      filterItems.length === 0
    ) {
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
      <ViewModeToggle viewMode={viewMode} setViewMode={setViewMode} />
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
            {table.getRowModel().rows.length === 0 && !dataQuery.isLoading ? (
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
        {dataQuery.isFetching && (
          <div className="pointer-events-none absolute top-2 right-2 inline-flex items-center gap-1 rounded bg-background/70 px-2 py-1 text-[10px] text-muted-foreground backdrop-blur-sm">
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
        onPageSizeChange={setPageSize}
        onPrev={() => setPageIndex(Math.max(0, pageIndex - 1))}
        onNext={() => setPageIndex(Math.min(totalPages - 1, pageIndex + 1))}
      />
    </>
  );

  return (
    <section className="flex flex-col overflow-hidden rounded-md border border-border bg-card">
      <SavedViewsBar rightSlot={rightSlot} countsEnabled={countsEnabled} />
      <div className="flex items-center gap-2 border-b border-border px-3 py-1.5">
        <Suspense fallback={<SearchSkeleton />}>
          <DataTableSearch value={query} onChange={setQuery} placeholder="Search shipments..." />
        </Suspense>
        <Suspense fallback={<ToolbarButtonSkeleton />}>
          <DataTableFilterBuilder
            columns={columns as ColumnDef<unknown>[]}
            filters={filterItems}
            onFiltersChange={setFilterItems}
          />
        </Suspense>
        <div className="mx-1 h-4 w-px bg-border" />
        <FilterChipRow />
        {viewMode === "table" && (
          <>
            <p className="ml-auto shrink-0 font-table text-[10.5px] text-muted-foreground tabular-nums">
              {rows.length} of {totalCount} results
            </p>
            <Suspense fallback={<ToolbarButtonSkeleton />}>
              <DataTableViewOptions table={table as unknown as TanstackTable<unknown>} />
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
      className="inline-flex overflow-hidden rounded-md border border-border"
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
          "flex items-center gap-1 border-l border-border px-2 py-1 text-[11px] transition-colors",
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
          "group/row h-9 cursor-pointer border-b border-border/70 transition-colors hover:bg-muted/30",
          isExpanded && "bg-brand/10 outline-1 -outline-offset-1 outline-brand hover:bg-brand/20",
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
        <tr className="border-b border-border bg-background">
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
  onPageSizeChange,
  onPrev,
  onNext,
}: {
  totalCount: number;
  pageIndex: number;
  totalPages: number;
  rowCount: number;
  pageSize: CommandCenterPageSize;
  onPageSizeChange: (size: CommandCenterPageSize) => void;
  onPrev: () => void;
  onNext: () => void;
}) {
  return (
    <div className="flex items-center justify-between border-t border-border px-3 py-1.5 text-[11px] text-muted-foreground">
      <p className="font-table tabular-nums">
        {rowCount} rows · page {pageIndex + 1} of {totalPages} · {totalCount} total
      </p>
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
