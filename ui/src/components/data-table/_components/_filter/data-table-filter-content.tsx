import { Button } from "@/components/ui/button";
import {
  createFieldFilter,
  defaultFilterOperators,
} from "@/lib/data-table-utils";
import { FilterStateSchema } from "@/lib/schemas/table-configuration-schema";
import { EnhancedColumnDef } from "@/types/data-table";
import { DataTableContentFooter, DataTableContentInner } from "../_shared";
import { AppliedFilterContent } from "./applied-filter-content";

type DataTableFilterContentProps = {
  filterState: FilterStateSchema;
  columns: EnhancedColumnDef<any>[];
  onFilterChange: (state: FilterStateSchema) => void;
};

export function DataTableFilterContent({
  filterState,
  columns,
  onFilterChange,
}: DataTableFilterContentProps) {
  const filterableColumns = columns.filter((col) => col.meta?.filterable);

  const handleAddFilter = (column: EnhancedColumnDef<any> | undefined) => {
    if (!column) return;

    const columnId = column.meta?.apiField || column.id;

    const filterType = column.meta?.filterType || "text";
    const defaultOperator =
      column.meta?.defaultFilterOperator || defaultFilterOperators[filterType];

    const newFilter = createFieldFilter(
      columnId as string,
      defaultOperator,
      "",
    );

    onFilterChange({
      ...filterState,
      filters: [...filterState.filters, newFilter],
    });
  };

  const firstAvailableColumn = filterableColumns.find(
    (col) =>
      !filterState.filters.some(
        (filter) => filter.field === (col.meta?.apiField || col.id),
      ),
  );

  const handleClearAllFilters = () => {
    onFilterChange({
      ...filterState,
      filters: [],
    });
  };

  const availableColumns = filterableColumns.filter(
    (col) =>
      !filterState.filters.some(
        (filter) => filter.field === (col.meta?.apiField || col.id),
      ),
  );
  return (
    <DataTableContentInner>
      <AppliedFilterContent
        filterState={filterState}
        columns={columns}
        onFilterChange={onFilterChange}
      />
      <DataTableContentFooter>
        <Button
          size="lg"
          onClick={() => {
            handleAddFilter(firstAvailableColumn);
          }}
          disabled={availableColumns.length === 0}
        >
          Add Filter
        </Button>
        {filterState.filters.length > 0 && (
          <Button variant="outline" size="lg" onClick={handleClearAllFilters}>
            Reset Filters
          </Button>
        )}
        {availableColumns.length === 0 && (
          <p className="text-xs text-muted-foreground mt-1">
            All filterable columns are already in use
          </p>
        )}
      </DataTableContentFooter>
    </DataTableContentInner>
  );
}
