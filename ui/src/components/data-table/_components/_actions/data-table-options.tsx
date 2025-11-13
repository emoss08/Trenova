import { LazyLoader } from "@/components/error-boundary";
import { FilterStateSchema } from "@/lib/schemas/table-configuration-schema";
import { Resource } from "@/types/audit-entry";
import { Config, EnhancedColumnDef, ExtraAction } from "@/types/data-table";
import { LiveModeTableConfig } from "@/types/live-mode";
import React, { lazy } from "react";
import { DataTableFilters } from "../_filter/data-table-filters";
import { DataTableSort } from "../_sort/data-table-sort";
import { DataTableActionsSkeleton } from "../data-table-skeleton";
import { DataTableSearch } from "./data-table-search";

const DataTableActions = lazy(() => import("../data-table-actions"));

export function DataTableOptions({
  config,
  handleFilterChange,
  setSearchParams,
  filterState,
  columns,
  name,
  resource,
  exportModelName,
  extraActions,
  liveMode,
  liveModeEnabled,
  setLiveModeEnabled,
}: {
  config: Config;
  handleFilterChange: (state: FilterStateSchema) => void;
  setSearchParams: (params: Record<string, any>) => void;
  filterState: FilterStateSchema;
  columns: EnhancedColumnDef<any>[];
  name: string;
  resource: Resource;
  exportModelName: string;
  liveModeEnabled: boolean;
  setLiveModeEnabled: (enabled: boolean) => void;
  extraActions?: ExtraAction[];
  liveMode?: LiveModeTableConfig;
}) {
  const timeoutRef = React.useRef<ReturnType<typeof setTimeout>>(null);

  const debouncedHandleFilterChange = React.useCallback(
    (newFilterState: FilterStateSchema) => {
      clearTimeout(timeoutRef.current!);
      timeoutRef.current = setTimeout(() => {
        handleFilterChange(newFilterState);
      }, config.searchDebounce || 300);
    },
    [handleFilterChange, config.searchDebounce],
  );

  const handleCreateClick = React.useCallback(() => {
    setSearchParams({ modalType: "create", entityId: null });
  }, [setSearchParams]);

  return (
    <DataTableOptionsOuter>
      <DataTableOptionsInner>
        {(config.showFilterUI || config.showSortUI) && (
          <DataTableOptionsFilters>
            <DataTableSearch
              filterState={filterState}
              onFilterChange={debouncedHandleFilterChange}
              placeholder={`Search ${name.toLowerCase()}...`}
            />
            {config.showFilterUI && (
              <DataTableFilters
                columns={columns}
                filterState={filterState}
                onFilterChange={handleFilterChange}
                config={config}
              />
            )}
            {config.showSortUI && (
              <DataTableSort
                columns={columns}
                sortState={filterState.sort}
                onSortChange={(newSort) => {
                  const newFilterState = {
                    ...filterState,
                    sort: newSort,
                  };
                  handleFilterChange(newFilterState);
                }}
                config={config}
              />
            )}
          </DataTableOptionsFilters>
        )}
        <LazyLoader fallback={<DataTableActionsSkeleton />}>
          <DataTableActions
            name={name}
            resource={resource}
            exportModelName={exportModelName}
            extraActions={extraActions}
            handleCreateClick={handleCreateClick}
            liveModeConfig={liveMode}
            liveModeEnabled={liveModeEnabled}
            onLiveModeToggle={setLiveModeEnabled}
            filterState={filterState}
          />
        </LazyLoader>
      </DataTableOptionsInner>
    </DataTableOptionsOuter>
  );
}

function DataTableOptionsOuter({ children }: { children: React.ReactNode }) {
  return <div className="flex h-auto w-full flex-col">{children}</div>;
}

function DataTableOptionsInner({ children }: { children: React.ReactNode }) {
  return <div className="flex items-center justify-between">{children}</div>;
}

function DataTableOptionsFilters({ children }: { children: React.ReactNode }) {
  return <div className="flex flex-col gap-2 lg:flex-row">{children}</div>;
}
