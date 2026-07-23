"use no memo";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@trenova/shared/components/ui/dropdown-menu";
import { useDataTable } from "@/contexts/data-table-context";
import type {
  AddRecordAction,
  DataTableGraphQLConfig,
  DataTableQueryOptions,
  FilterItem,
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
import type { ColumnDef, Table } from "@tanstack/react-table";
import { ChevronDownIcon, DownloadIcon, PlusIcon } from "lucide-react";
import { lazy, Suspense, useState } from "react";
import { Button } from "@trenova/shared/components/ui/button";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
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

const DataTableFormatBuilder = lazy(
  () => import("@/components/data-table/data-table-format-builder"),
);

const DataTableDisplayMenu = lazy(
  () => import("@/components/data-table/data-table-display-menu"),
);

const DataTableConfigManager = lazy(
  () => import("@/components/data-table/data-table-config-manager"),
);

const DataTableExportDialog = lazy(
  () => import("@/components/data-table/data-table-export-dialog"),
);

function ToolbarButtonSkeleton() {
  return <Skeleton className="h-7 w-20" />;
}

function SearchSkeleton() {
  return <Skeleton className="h-7 w-48" />;
}

type DataTableExportContext<TData extends Record<string, any>> = {
  graphql: DataTableGraphQLConfig<TData>;
  queryOptions: Omit<DataTableQueryOptions, "cursor">;
  currentPageRows: TData[];
  totalCount: number | null;
};

type DataTableToolbarProps<TData extends Record<string, any>> = {
  table: Table<TData>;
  columns: ColumnDef<TData>[];
  query: string;
  onSearchChange: (query: string) => void;
  filters: FilterItem[];
  onFiltersChange: (filters: FilterItem[]) => void;
  sort: SortField[];
  onSortChange: (sort: SortField[]) => void;
  addRecordActions?: AddRecordAction[];
  resource?: string;
  currentConfig: TableConfig;
  onApplyConfig?: (config: TableConfig, source?: TableViewSource) => void;
  activeView?: ActiveTableView | null;
  isViewDirty?: boolean;
  onViewPersisted?: (config: TableConfiguration) => void;
  onViewDeleted?: (id: string) => void;
  formatRules?: TableFormatRule[];
  onFormatRulesChange?: (rules: TableFormatRule[]) => void;
  density?: TableDensity;
  onDensityChange?: (density: TableDensity) => void;
  exportContext?: DataTableExportContext<TData>;
};

export function DataTableToolbar<TData extends Record<string, any>>({
  table,
  columns,
  query,
  onSearchChange,
  filters,
  onFiltersChange,
  sort,
  onSortChange,
  addRecordActions = [],
  resource,
  currentConfig,
  onApplyConfig,
  activeView,
  isViewDirty = false,
  onViewPersisted,
  onViewDeleted,
  formatRules = [],
  onFormatRulesChange,
  density = "comfortable",
  onDensityChange,
  exportContext,
}: DataTableToolbarProps<TData>) {
  const { canExport } = useDataTable<TData, unknown>();
  const [saveDialogOpen, setSaveDialogOpen] = useState(false);
  const [exportDialogOpen, setExportDialogOpen] = useState(false);
  const [formatDialogOpen, setFormatDialogOpen] = useState(false);
  const hasAddRecordActions = addRecordActions.length > 0;
  const hasSingleAddRecordAction = addRecordActions.length === 1;
  const showExport = !!exportContext && !!resource && canExport;

  return (
    <>
      <div className="flex items-center justify-between gap-2">
        <div className="flex flex-1 items-center gap-2">
          <Suspense fallback={<SearchSkeleton />}>
            <DataTableSearch value={query} onChange={onSearchChange} />
          </Suspense>
          <Suspense fallback={<ToolbarButtonSkeleton />}>
            <DataTableFilterBuilder
              columns={columns as unknown as ColumnDef<unknown>[]}
              filters={filters}
              onFiltersChange={onFiltersChange}
            />
          </Suspense>
          <Suspense fallback={<ToolbarButtonSkeleton />}>
            <DataTableSortBuilder
              columns={columns as unknown as ColumnDef<unknown>[]}
              sort={sort}
              onSortChange={onSortChange}
            />
          </Suspense>
        </div>

        <div className="flex items-center gap-2">
          <Suspense fallback={<ToolbarButtonSkeleton />}>
            <DataTableDisplayMenu
              table={table as Table<unknown>}
              density={density}
              onDensityChange={onDensityChange}
              formatRuleCount={formatRules.length}
              onEditFormatRules={
                onFormatRulesChange ? () => setFormatDialogOpen(true) : undefined
              }
            />
          </Suspense>
          {showExport && (
            <Tooltip>
              <TooltipTrigger
                render={
                  <Button
                    variant="outline"
                    size="sm"
                    aria-label="Export to CSV"
                    onClick={() => setExportDialogOpen(true)}
                  >
                    <DownloadIcon className="size-4" />
                  </Button>
                }
              />
              <TooltipContent>Export to CSV</TooltipContent>
            </Tooltip>
          )}
          <Suspense fallback={<ToolbarButtonSkeleton />}>
            {resource && onApplyConfig && (
              <DataTableConfigManager
                resource={resource}
                onApplyConfig={onApplyConfig}
                onSaveConfig={() => setSaveDialogOpen(true)}
                currentConfig={currentConfig}
                activeViewId={activeView?.id ?? null}
                activeViewName={activeView?.name ?? null}
                isViewDirty={isViewDirty}
                onViewPersisted={onViewPersisted}
                onViewDeleted={onViewDeleted}
              />
            )}
          </Suspense>
          {hasSingleAddRecordAction ? (
            <Button
              variant="default"
              size="sm"
              onClick={addRecordActions[0]?.onClick}
            >
              <PlusIcon className="size-3.5" />
              Add Record
            </Button>
          ) : null}
          {hasAddRecordActions && !hasSingleAddRecordAction ? (
            <DropdownMenu>
              <DropdownMenuTrigger render={<Button variant="default" size="sm" />}>
                <PlusIcon className="size-3.5" />
                Add Record
                <ChevronDownIcon className="size-3.5" />
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end" className="min-w-72">
                {addRecordActions.map((action) => (
                  <DropdownMenuItem
                    key={action.id}
                    title={action.label}
                    description={action.description}
                    onClick={action.onClick}
                    startContent={action.icon ? <action.icon className="size-4" /> : undefined}
                  />
                ))}
              </DropdownMenuContent>
            </DropdownMenu>
          ) : null}
        </div>
      </div>

      {resource && (
        <DataTableSaveConfigDialog
          open={saveDialogOpen}
          onOpenChange={setSaveDialogOpen}
          resource={resource}
          currentConfig={currentConfig}
          onSaved={onViewPersisted}
        />
      )}
      {onFormatRulesChange && formatDialogOpen && (
        <Suspense fallback={null}>
          <DataTableFormatBuilder
            open={formatDialogOpen}
            onOpenChange={setFormatDialogOpen}
            columns={columns as unknown as ColumnDef<unknown>[]}
            rules={formatRules}
            onRulesChange={onFormatRulesChange}
          />
        </Suspense>
      )}
      {showExport && exportDialogOpen && (
        <Suspense fallback={null}>
          <DataTableExportDialog
            open={exportDialogOpen}
            onOpenChange={setExportDialogOpen}
            resource={resource}
            table={table}
            graphql={exportContext.graphql}
            queryOptions={exportContext.queryOptions}
            currentPageRows={exportContext.currentPageRows}
            totalCount={exportContext.totalCount}
          />
        </Suspense>
      )}
    </>
  );
}
