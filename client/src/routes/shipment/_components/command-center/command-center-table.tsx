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
import { fetchData } from "@/hooks/data-table/use-data-table-query";
import { useDebounce } from "@/hooks/use-debounce";
import {
  convertFilterItemsToFieldFilters,
  convertFilterItemsToFilterGroups,
  initializeFilterItemsFromFieldFilters,
  initializeFilterItemsFromFilterGroups,
} from "@/lib/data-table";
import { queries } from "@/lib/queries";
import { cn } from "@/lib/utils";
import type { FieldFilter, FilterItem, SortField } from "@/types/data-table";
import type { Shipment } from "@/types/shipment";
import type { TableConfig } from "@/types/table-configuration";
import { useQuery } from "@tanstack/react-query";
import {
  flexRender,
  getCoreRowModel,
  useReactTable,
  type ColumnDef,
  type Row,
  type SortingState,
  type Table as TanstackTable,
  type VisibilityState,
} from "@tanstack/react-table";
import { ChevronLeftIcon, ChevronRightIcon, LayoutGridIcon, TableIcon } from "lucide-react";
import { lazy, Suspense, useEffect, useMemo, useRef, useState } from "react";
import { ExpandedRow } from "./expanded-row";
import type { ShipmentDocumentUploadContext } from "./expanded-row/document-stack";
import { FilterChipRow } from "./filter-chip-row";
import { SavedViewsBar } from "./saved-views-bar";
import { useCommandCenterStore } from "./store";
import { TimelinePlaceholder } from "./timeline-placeholder";
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
const DataTableSortBuilder = lazy(() => import("@/components/data-table/data-table-sort-builder"));
const DataTableViewOptions = lazy(() => import("@/components/data-table/data-table-view-options"));
const DataTableConfigManager = lazy(
  () => import("@/components/data-table/data-table-config-manager"),
);
const DataTableSaveConfigDialog = lazy(() =>
  import("@/components/data-table/data-table-save-config-dialog").then((m) => ({
    default: m.DataTableSaveConfigDialog,
  })),
);

function ToolbarButtonSkeleton() {
  return <Skeleton className="h-7 w-20" />;
}

function SearchSkeleton() {
  return <Skeleton className="h-7 w-48" />;
}

const SHIPMENTS_LINK = "/shipments/";
const QUERY_KEY = "shipment-list";
const RESOURCE_NAME = "Shipment";

type CommandCenterTableProps = {
  columns: ColumnDef<Shipment>[];
  mandatoryFieldFilters: FieldFilter[];
  onUploadDocument: (shipment: Shipment, context?: ShipmentDocumentUploadContext) => void;
  onSummaryChange?: (summary: CommandCenterTableSummary) => void;
};

export type CommandCenterTableSummary = {
  totalCount: number;
  dataUpdatedAt: number;
};

export function CommandCenterTable({
  columns,
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
  const [sort, setSort] = useState<SortField[]>([]);
  const [columnVisibility, setColumnVisibility] = useState<VisibilityState>({});
  const [columnOrder, setColumnOrder] = useState<string[]>([]);
  const [saveDialogOpen, setSaveDialogOpen] = useState(false);

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
  }, [mergedFieldFilters, userFilterGroups, sort, query, setUrl]);

  const dataQuery = useQuery({
    queryKey: [
      QUERY_KEY,
      "command-center",
      { pageIndex, pageSize },
      mergedFieldFilters,
      userFilterGroups,
      sort,
      query,
    ],
    queryFn: () =>
      fetchData<Shipment & Record<string, unknown>>(SHIPMENTS_LINK, pageIndex, pageSize, {
        query,
        fieldFilters: mergedFieldFilters,
        filterGroups: userFilterGroups,
        sort,
        extraSearchParams: { expandShipmentDetails: true },
      }),
    placeholderData: (prev) => prev,
  });

  const totalCount = dataQuery.data?.count ?? 0;
  const totalPages = Math.max(1, Math.ceil(totalCount / pageSize));
  const rows = (dataQuery.data?.results ?? []) as Shipment[];
  const backgroundQueriesEnabled = dataQuery.isSuccess;

  useEffect(() => {
    if (!dataQuery.data) return;
    onSummaryChange?.({
      totalCount,
      dataUpdatedAt: dataQuery.dataUpdatedAt,
    });
  }, [dataQuery.data, dataQuery.dataUpdatedAt, onSummaryChange, totalCount]);

  const sortingState = useMemo<SortingState>(
    () => sort.map((s) => ({ id: s.field, desc: s.direction === "desc" })),
    [sort],
  );

  const table = useReactTable({
    data: rows,
    columns,
    getCoreRowModel: getCoreRowModel(),
    state: { sorting: sortingState, columnVisibility, columnOrder },
    onColumnVisibilityChange: setColumnVisibility,
    onColumnOrderChange: setColumnOrder,
    manualPagination: true,
    manualSorting: true,
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
      setSort(config.sort ?? []);
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
      filterItems.length === 0 &&
      sort.length === 0
    ) {
      appliedDefaultRef.current = true;
      handleApplyConfig(defaultConfig.tableConfig);
    }
  }, [defaultConfig, filterItems.length, sort.length, handleApplyConfig]);

  const currentConfig = useMemo<TableConfig>(() => {
    const visibility: Record<string, boolean> = {};
    for (const col of table.getAllColumns()) visibility[col.id] = col.getIsVisible();
    return {
      fieldFilters: userFieldFilters,
      filterGroups: userFilterGroups,
      joinOperator: "and",
      sort,
      pageSize,
      columnVisibility: visibility,
      columnOrder: table.getState().columnOrder,
    };
  }, [userFieldFilters, userFilterGroups, sort, pageSize, table]);

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

  if (viewMode === "timeline") {
    return (
      <section className="flex flex-col rounded-md border border-border bg-card">
        <SavedViewsBar rightSlot={rightSlot} countsEnabled={backgroundQueriesEnabled} />
        <div className="p-4">
          <TimelinePlaceholder />
        </div>
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

  return (
    <section className="flex flex-col overflow-hidden rounded-md border border-border bg-card">
      <SavedViewsBar rightSlot={rightSlot} countsEnabled={backgroundQueriesEnabled} />

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
        <Suspense fallback={<ToolbarButtonSkeleton />}>
          <DataTableSortBuilder
            columns={columns as ColumnDef<unknown>[]}
            sort={sort}
            onSortChange={setSort}
          />
        </Suspense>
        <div className="mx-1 h-4 w-px bg-border" />
        <FilterChipRow />
        <p className="ml-auto shrink-0 font-table text-[10.5px] text-muted-foreground tabular-nums">
          {rows.length} of {totalCount} results
        </p>
        <Suspense fallback={<ToolbarButtonSkeleton />}>
          <DataTableViewOptions table={table as unknown as TanstackTable<unknown>} />
        </Suspense>
      </div>

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
        <LayoutGridIcon className="size-3" />
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
  onUploadDocument,
}: {
  row: Row<Shipment>;
  isExpanded: boolean;
  isHighlighted: boolean;
  onClick: () => void;
  onMouseEnter: () => void;
  onMouseLeave: () => void;
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
        <tr className="border-b border-border bg-muted">
          <td colSpan={row.getVisibleCells().length} className="p-0">
            <div className="cc-fade-in">
              <ExpandedRow shipment={row.original} onUploadDocument={onUploadDocument} />
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
            <SelectTrigger className="h-6 w-[58px] py-0 text-[11px]">
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
