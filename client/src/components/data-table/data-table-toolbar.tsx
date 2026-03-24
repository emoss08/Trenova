"use no memo";
import type { FilterItem, SortField } from "@/types/data-table";
import type { TableConfig } from "@/types/table-configuration";
import type { ColumnDef, Table } from "@tanstack/react-table";
import { PlusIcon } from "lucide-react";
import { lazy, Suspense, useState } from "react";
import { Button } from "../ui/button";
import { Skeleton } from "../ui/skeleton";
import { DataTableSaveConfigDialog } from "./data-table-save-config-dialog";

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

function ToolbarButtonSkeleton() {
  return <Skeleton className="h-8 w-20" />;
}

function SearchSkeleton() {
  return <Skeleton className="h-8 w-48" />;
}

type DataTableToolbarProps<TData> = {
  table: Table<TData>;
  columns: ColumnDef<TData>[];
  query: string;
  onSearchChange: (query: string) => void;
  filters: FilterItem[];
  onFiltersChange: (filters: FilterItem[]) => void;
  sort: SortField[];
  onSortChange: (sort: SortField[]) => void;
  onAddRecord?: () => void;
  resource?: string;
  currentConfig: TableConfig;
  onApplyConfig?: (config: TableConfig) => void;
};

export function DataTableToolbar<TData>({
  table,
  columns,
  query,
  onSearchChange,
  filters,
  onFiltersChange,
  sort,
  onSortChange,
  onAddRecord,
  resource,
  currentConfig,
  onApplyConfig,
}: DataTableToolbarProps<TData>) {
  const [saveDialogOpen, setSaveDialogOpen] = useState(false);

  return (
    <>
      <div className="flex items-center justify-between gap-2">
        <div className="flex flex-1 items-center gap-2">
          <Suspense fallback={<SearchSkeleton />}>
            <DataTableSearch value={query} onChange={onSearchChange} />
          </Suspense>
          <Suspense fallback={<ToolbarButtonSkeleton />}>
            <DataTableFilterBuilder
              columns={columns as ColumnDef<unknown>[]}
              filters={filters}
              onFiltersChange={onFiltersChange}
            />
          </Suspense>
          <Suspense fallback={<ToolbarButtonSkeleton />}>
            <DataTableSortBuilder
              columns={columns as ColumnDef<unknown>[]}
              sort={sort}
              onSortChange={onSortChange}
            />
          </Suspense>
        </div>

        <div className="flex items-center gap-2">
          <Suspense fallback={<ToolbarButtonSkeleton />}>
            <DataTableViewOptions table={table as Table<unknown>} />
          </Suspense>
          <Suspense fallback={<ToolbarButtonSkeleton />}>
            {resource && onApplyConfig && (
              <DataTableConfigManager
                resource={resource}
                onApplyConfig={onApplyConfig}
                onSaveConfig={() => setSaveDialogOpen(true)}
              />
            )}
          </Suspense>
          {onAddRecord && (
            <Button variant="default" size="sm" onClick={onAddRecord}>
              <PlusIcon className="size-3.5" />
              Add Record
            </Button>
          )}
        </div>
      </div>

      {resource && (
        <DataTableSaveConfigDialog
          open={saveDialogOpen}
          onOpenChange={setSaveDialogOpen}
          resource={resource}
          currentConfig={currentConfig}
        />
      )}
    </>
  );
}
