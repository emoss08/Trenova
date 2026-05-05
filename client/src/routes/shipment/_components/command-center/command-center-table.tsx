"use no memo";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { Spinner } from "@/components/ui/spinner";
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
import type {
  FieldFilter,
  FilterItem,
  SortField,
} from "@/types/data-table";
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
import {
  ChevronLeftIcon,
  ChevronRightIcon,
  LayoutGridIcon,
  TableIcon,
} from "lucide-react";
import { lazy, Suspense, useEffect, useMemo, useRef, useState } from "react";
import { ExpandedRow } from "./expanded-row";
import { FilterChipRow } from "./filter-chip-row";
import { SavedViewsBar } from "./saved-views-bar";
import { useCommandCenterStore } from "./store";
import { TimelinePlaceholder } from "./timeline-placeholder";
import { useCommandCenterUrl, type CommandCenterViewMode } from "./url-state";

const DataTableSearch = lazy(
  () => import("@/components/data-table/data-table-search"),
);
const DataTableFilterBuilder = lazy(
  () => import("@/components/data-table/data-table-filter-builder"),
);
const DataTableSortBuilder = lazy(
  () => import("@/components/data-table/data-table-sort-builder"),
);
const DataTableViewOptions = lazy(
  () => import("@/components/data-table/data-table-view-options"),
);
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
const PAGE_SIZE = 25;
const QUERY_KEY = "shipment-list";
const RESOURCE_NAME = "Shipment";

type CommandCenterTableProps = {
  columns: ColumnDef<Shipment>[];
  mandatoryFieldFilters: FieldFilter[];
};

export function CommandCenterTable({
  columns,
  mandatoryFieldFilters,
}: CommandCenterTableProps) {
  const [{ mode: viewMode, expanded: expandedId, page, q: query }, setUrl] =
    useCommandCenterUrl();
  const pageIndex = Math.max(0, page - 1);
  const setQuery = (next: string) =>
    void setUrl({ q: next.length === 0 ? null : next, page: 1 });
  const setPageIndex = (next: number) => void setUrl({ page: next + 1 });
  const setViewMode = (next: CommandCenterViewMode) => void setUrl({ mode: next });
  const toggleExpandedId = (id: string) =>
    void setUrl({ expanded: expandedId === id ? null : id });

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

  // Reset to page 1 whenever the filter/sort/view inputs change so we never
  // land on an out-of-range page after refining the result set.
  useEffect(() => {
    void setUrl({ page: 1 });
  }, [mergedFieldFilters, userFilterGroups, sort, query, setUrl]);

  const totalShipmentCount = useTotalShipmentCount();

  const dataQuery = useQuery({
    queryKey: [
      QUERY_KEY,
      "command-center",
      { pageIndex, pageSize: PAGE_SIZE },
      mergedFieldFilters,
      userFilterGroups,
      sort,
      query,
    ],
    queryFn: () =>
      fetchData<Shipment & Record<string, unknown>>(
        SHIPMENTS_LINK,
        pageIndex,
        PAGE_SIZE,
        {
          query,
          fieldFilters: mergedFieldFilters,
          filterGroups: userFilterGroups,
          sort,
          extraSearchParams: { expandShipmentDetails: true },
        },
      ),
    placeholderData: (prev) => prev,
  });

  const totalCount = dataQuery.data?.count ?? 0;
  const totalPages = Math.max(1, Math.ceil(totalCount / PAGE_SIZE));
  const rows = (dataQuery.data?.results ?? []) as Shipment[];

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

  // ── default-config bootstrap (one-shot) ────────────────────────────────
  const { data: defaultConfig } = useQuery({
    ...queries.tableConfiguration.default(RESOURCE_NAME),
    retry: false,
    staleTime: Infinity,
  });
  const appliedDefaultRef = useRef(false);

  const handleApplyConfig = useMemo(
    () => (config: TableConfig) => {
      const fieldFilters = config.fieldFilters ?? [];
      const filterGroups = (config.filterGroups ?? []).filter(
        (g) => g.filters?.length > 0,
      );
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
      pageSize: PAGE_SIZE,
      columnVisibility: visibility,
      columnOrder: table.getState().columnOrder,
    };
  }, [userFieldFilters, userFilterGroups, sort, table]);

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
      <section className="cc-card">
        <SavedViewsBar rightSlot={rightSlot} />
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
    <section className="cc-card overflow-hidden">
      <SavedViewsBar rightSlot={rightSlot} />

      <div className="flex items-center gap-2 border-b border-border px-3 py-1.5">
        <Suspense fallback={<SearchSkeleton />}>
          <DataTableSearch
            value={query}
            onChange={setQuery}
            placeholder="Search shipments..."
          />
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
          {totalCount} of {totalShipmentCount ?? "—"} results
        </p>
        <Suspense fallback={<ToolbarButtonSkeleton />}>
          <DataTableViewOptions table={table as unknown as TanstackTable<unknown>} />
        </Suspense>
      </div>

      <div className="cc-table-scroll relative">
        <table className="cc-table">
          <colgroup>
            {table.getVisibleFlatColumns().map((col) => (
              <col key={col.id} style={{ width: `${col.getSize()}px` }} />
            ))}
          </colgroup>
          <thead>
            {table.getHeaderGroups().map((hg) => (
              <tr key={hg.id}>
                {hg.headers.map((header) => (
                  <th
                    key={header.id}
                    className="cc-th"
                    style={{ width: `${header.getSize()}px` }}
                  >
                    {header.isPlaceholder
                      ? null
                      : flexRender(header.column.columnDef.header, header.getContext())}
                  </th>
                ))}
              </tr>
            ))}
          </thead>
          <tbody>
            {table.getRowModel().rows.length === 0 && !dataQuery.isLoading ? (
              <tr>
                <td
                  colSpan={table.getVisibleFlatColumns().length}
                  className="px-3 py-12 text-center text-sm text-muted-foreground"
                >
                  No shipments match the current view.
                </td>
              </tr>
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
                  />
                );
              })
            )}
          </tbody>
        </table>
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
}: {
  row: Row<Shipment>;
  isExpanded: boolean;
  isHighlighted: boolean;
  onClick: () => void;
  onMouseEnter: () => void;
  onMouseLeave: () => void;
}) {
  return (
    <>
      <tr
        className={cn(
          "cc-row group/row",
          isExpanded && "cc-row-expanded",
          isHighlighted && "cc-row-highlighted",
        )}
        onClick={onClick}
        onMouseEnter={onMouseEnter}
        onMouseLeave={onMouseLeave}
      >
        {row.getVisibleCells().map((cell) => (
          <td key={cell.id} className="cc-td">
            {flexRender(cell.column.columnDef.cell, cell.getContext())}
          </td>
        ))}
      </tr>
      {isExpanded && (
        <tr className="cc-row-expansion">
          <td colSpan={row.getVisibleCells().length} className="p-0">
            <div className="cc-fade-in">
              <ExpandedRow shipment={row.original} />
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
  onPrev,
  onNext,
}: {
  totalCount: number;
  pageIndex: number;
  totalPages: number;
  rowCount: number;
  onPrev: () => void;
  onNext: () => void;
}) {
  return (
    <div className="flex items-center justify-between border-t border-border px-3 py-1.5 text-[11px] text-muted-foreground">
      <p className="font-table tabular-nums">
        {rowCount} rows · page {pageIndex + 1} of {totalPages} · {totalCount} total
      </p>
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
  );
}

function useTotalShipmentCount(): number | undefined {
  const query = useQuery({
    queryKey: [QUERY_KEY, "command-center", "total"],
    queryFn: () =>
      fetchData<Shipment & Record<string, unknown>>(SHIPMENTS_LINK, 0, 1, {
        fieldFilters: [],
      }),
    staleTime: 60_000,
  });
  return query.data?.count;
}
