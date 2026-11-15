import { useT } from "@trenova/shared/i18n/use-t";
import type { RowData } from "@tanstack/react-table";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@trenova/shared/components/ui/dropdown-menu";
import { useDataTable } from "@/contexts/data-table-context";
import type {
  AddRecordAction,
  DataTableGraphQLSource,
  DataTableQueryOptions,
  FieldFilter,
  FilterItem,
  SortField,
  ColumnDef,
  DataTableToolbarSlots,
  Table,
} from "@trenova/shared/types/data-table";
import type { ComposedTableQuery } from "@/types/table-query";
import type { TableArrangement, TableConfig, TableViewSource } from "@/types/table-configuration";
import { useDataTableView, useDataTableViewActions } from "./data-table-view-context";
import type { DataTableViewState } from "./data-table-view-store";
import { ChevronDownIcon, Download01Icon, PlusIcon } from "@trenova/shared/components/icons";
import { Fragment, lazy, Suspense, useState } from "react";
import { DataTableViewers } from "@/components/presence/data-table-viewers";
import { Button } from "@trenova/shared/components/ui/button";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import { DataTableSaveConfigDialog } from "./data-table-save-config-dialog";
import { cn, toSentenceFragment } from "@trenova/shared/lib/utils";
import { DEFAULT_CREATE_ACTION_ID, type FilterableField } from "@/lib/data-table";
import { TABLE_TOOLBAR_CLASS } from "./data-table-chrome";
import { PlanFeature } from "@/lib/plan-feature";
import { usePlanFeature } from "@trenova/shared/hooks/use-plan-feature";

const DataTableSearch = lazy(() => import("@/components/data-table/data-table-search"));

const DataTableFilterBuilder = lazy(
  () => import("@/components/data-table/data-table-filter-builder"),
);

const DataTableSortBuilder = lazy(() => import("@/components/data-table/data-table-sort-builder"));

const DataTableFormatBuilder = lazy(
  () => import("@/components/data-table/data-table-format-builder"),
);

const DataTableDisplayMenu = lazy(() => import("@/components/data-table/data-table-display-menu"));

const DataTableConfigManager = lazy(
  () => import("@/components/data-table/data-table-config-manager"),
);

const DataTableExportDialog = lazy(
  () => import("@/components/data-table/data-table-export-dialog"),
);

const DataTableAsk = lazy(() =>
  import("@/components/data-table/data-table-ask").then((module) => ({
    default: module.DataTableAsk,
  })),
);

function ToolbarButtonSkeleton() {
  return <Skeleton className="h-7 w-20" />;
}

function SearchSkeleton() {
  return <Skeleton className="h-7 w-48" />;
}

type DataTableExportContext<TData extends Record<string, any>> = {
  /** The table's permission resource; with one, the table can export as a report. */
  permissionResource?: string;
  graphql: DataTableGraphQLSource<TData>;
  queryOptions: Omit<DataTableQueryOptions, "cursor">;
  currentPageRows: TData[];
  totalCount: number | null;
};

type DataTableToolbarProps<TData extends Record<string, any>> = {
  table: Table<TData>;
  columns: ColumnDef<TData>[];
  filterFields: readonly FilterableField[];
  query: string;
  onSearchChange: (query: string) => void;
  filters: FilterItem[];
  onFiltersChange: (filters: FilterItem[]) => void;
  sort: SortField[];
  onSortChange: (sort: SortField[]) => void;
  /** The filters as the API states them, which is what the composer narrows. */
  fieldFilters?: FieldFilter[];
  onAskApplied?: (composed: ComposedTableQuery) => void;
  addRecordActions?: AddRecordAction[];
  resource?: string;
  /**
   * The permission resource presence joins under. DataTable passes its name as
   * resource, which also titles the "New" button, and "AI Audit Event" is no
   * resource the server knows.
   */
  presenceResource?: string;
  /** What the table shows now: the filters, sort, page size and columns a view keeps. */
  arrangement: TableArrangement;
  onApplyConfig?: (config: TableConfig, source?: TableViewSource) => void;
  hasSavedLayout?: boolean;
  onResetLayout?: () => void;
  onFitColumns?: () => void;
  canMarkChanges?: boolean;
  canShowTotals?: boolean;
  canVirtualize?: boolean;
  exportContext?: DataTableExportContext<TData>;
  slots?: DataTableToolbarSlots;
};

export function DataTableToolbar<TData extends Record<string, any>>({
  table,
  columns,
  filterFields,
  query,
  onSearchChange,
  filters,
  onFiltersChange,
  sort,
  onSortChange,
  fieldFilters = [],
  onAskApplied,
  addRecordActions = [],
  resource,
  presenceResource,
  arrangement,
  onApplyConfig,
  hasSavedLayout,
  onResetLayout,
  onFitColumns,
  canMarkChanges,
  canShowTotals,
  canVirtualize,
  exportContext,
  slots,
}: DataTableToolbarProps<TData>) {
  const t = useT();
  // Narrowing a table by description is Nova's, which only a plan with Trenova AI has: without
  // it the input is never mounted, its code never loaded and its catalogue never read.
  const asksNova = usePlanFeature(PlanFeature.AIDesk).available;

  const { canExport } = useDataTable<TData, unknown>();
  const [saveDialogOpen, setSaveDialogOpen] = useState(false);
  const [exportDialogOpen, setExportDialogOpen] = useState(false);
  const [formatDialogOpen, setFormatDialogOpen] = useState(false);
  const hasAddRecordActions = addRecordActions.length > 0;
  const hasSingleAddRecordAction = addRecordActions.length === 1;
  const showExport = !!exportContext && !!resource && canExport;

  return (
    <>
      <div className={TABLE_TOOLBAR_CLASS}>
        <div className="flex min-w-0 flex-1 items-center gap-2">
          <Suspense fallback={<SearchSkeleton />}>
            <DataTableSearch
              value={query}
              onChange={onSearchChange}
              suggestions={slots?.searchSuggestions}
              shortcut={slots?.searchShortcut}
            />
          </Suspense>
          <Suspense fallback={<ToolbarButtonSkeleton />}>
            <DataTableFilterBuilder
              fields={filterFields}
              filters={filters}
              onFiltersChange={onFiltersChange}
              labelClassName={slots?.responsive?.label}
            />
          </Suspense>
          <Suspense fallback={<ToolbarButtonSkeleton />}>
            <DataTableSortBuilder
              columns={columns as unknown as ColumnDef<RowData>[]}
              sort={sort}
              onSortChange={onSortChange}
              labelClassName={slots?.responsive?.label}
            />
          </Suspense>
          {/* Whether this table can be narrowed by description is a server
              fact, so the input asks the server itself and renders nothing
              until it knows — a skeleton here would promise a control that
              most tables never get. */}
          {asksNova && resource && onAskApplied && (
            <Suspense fallback={null}>
              <DataTableAsk
                resource={resource}
                query={query}
                fieldFilters={fieldFilters}
                sort={sort}
                onApply={onAskApplied}
              />
            </Suspense>
          )}
        </div>

        <div className="flex shrink-0 items-center gap-2">
          {presenceResource ? <DataTableViewers resource={presenceResource} /> : null}
          {slots?.trailing}
          {showExport && (
            <Tooltip>
              <TooltipTrigger
                render={
                  <Button
                    variant="outline"
                    size="sm"
                    aria-label={t("Export, print or schedule")}
                    onClick={() => setExportDialogOpen(true)}
                  >
                    <Download01Icon className="size-4" />
                  </Button>
                }
              />
              <TooltipContent>{t("Export, print or schedule")}</TooltipContent>
            </Tooltip>
          )}
          <div className={cn("contents", slots?.responsive?.secondary)}>
            <Suspense fallback={<ToolbarButtonSkeleton />}>
              <DataTableDisplayMenu
                table={table as Table<RowData>}
                onEditFormatRules={() => setFormatDialogOpen(true)}
                hasSavedLayout={hasSavedLayout}
                onResetLayout={onResetLayout}
                onFitColumns={onFitColumns}
                canMarkChanges={canMarkChanges}
                canShowTotals={canShowTotals}
                canVirtualize={canVirtualize}
              />
            </Suspense>
          </div>
          <div className={cn("contents", slots?.responsive?.secondary)}>
            <Suspense fallback={<ToolbarButtonSkeleton />}>
              {resource && onApplyConfig && (
                <DataTableConfigManager
                  resource={resource}
                  onApplyConfig={onApplyConfig}
                  onSaveConfig={() => setSaveDialogOpen(true)}
                  arrangement={arrangement}
                />
              )}
            </Suspense>
          </div>
          {slots?.end}
          {hasSingleAddRecordAction ? (
            <Button variant="default" size="sm" onClick={addRecordActions[0]?.onClick}>
              <PlusIcon className="size-3.5" />
              {resource ? t("New {0}", toSentenceFragment(resource)) : t("New record")}
            </Button>
          ) : null}
          {hasAddRecordActions && !hasSingleAddRecordAction ? (
            <DropdownMenu>
              <DropdownMenuTrigger render={<Button variant="default" size="sm" />}>
                <PlusIcon className="size-3.5" />
                {resource ? t("New {0}", toSentenceFragment(resource)) : t("New record")}
                <ChevronDownIcon className="size-3.5" />
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end" className="w-80">
                {addRecordActions.map((action, index) => (
                  <Fragment key={action.id}>
                    {addRecordActions[index - 1]?.id === DEFAULT_CREATE_ACTION_ID && (
                      <DropdownMenuSeparator />
                    )}
                    <DropdownMenuItem
                      title={t(action.label)}
                      description={action.description ? t(action.description) : undefined}
                      onClick={action.onClick}
                      className="items-start py-1.5"
                      descriptionClassProps="text-pretty"
                      startContent={
                        action.icon ? (
                          <action.icon className="text-muted-foreground mt-0.5 size-4" />
                        ) : undefined
                      }
                    />
                  </Fragment>
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
          arrangement={arrangement}
        />
      )}
      {formatDialogOpen && (
        <Suspense fallback={null}>
          <FormatRulesDialog
            columns={columns as unknown as ColumnDef<RowData>[]}
            onClose={() => setFormatDialogOpen(false)}
          />
        </Suspense>
      )}
      {showExport && exportDialogOpen && (
        <Suspense fallback={null}>
          <DataTableExportDialog
            open={exportDialogOpen}
            onOpenChange={setExportDialogOpen}
            resource={resource}
            permissionResource={exportContext.permissionResource}
            table={table as Table<Record<string, any>>}
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

const selectFormatRules = (state: DataTableViewState) => state.formatRules;

/** The colour rules being edited, read and written straight to the table's view. */
function FormatRulesDialog({
  columns,
  onClose,
}: {
  columns: ColumnDef<RowData>[];
  onClose: () => void;
}) {
  const rules = useDataTableView(selectFormatRules);
  const { setFormatRules } = useDataTableViewActions();

  return (
    <DataTableFormatBuilder
      open
      onOpenChange={(open) => {
        if (!open) onClose();
      }}
      columns={columns}
      rules={rules}
      onRulesChange={setFormatRules}
    />
  );
}
