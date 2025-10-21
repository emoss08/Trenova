import { Separator } from "@/components/ui/separator";
import {
  FilterStateSchema,
  LogicalOperator,
} from "@/lib/schemas/table-configuration-schema";
import { EnhancedColumnDef } from "@/types/data-table";
import { useCallback } from "react";
import {
  TableFilterConentHeader,
  TableFilterConentInner,
  TableFilterContentEmptyStat,
  TableFilterContentOuter,
} from "../_shared";
import { FilterRow } from "./data-table-filter-row";

export function AppliedFilterContent({
  filterState,
  columns,
  onFilterChange,
}: {
  filterState: FilterStateSchema;
  columns: EnhancedColumnDef<any>[];
  onFilterChange: (state: FilterStateSchema) => void;
}) {
  const handleRemoveFilter = (index: number) => {
    const newFilters = filterState.filters.filter((_, i) => i !== index);
    onFilterChange({
      ...filterState,
      filters: newFilters,
    });
  };

  const handleUpdateFilter = (
    index: number,
    updatedFilter: FilterStateSchema["filters"][number],
  ) => {
    const newFilters = filterState.filters.map((filter, i) =>
      i === index ? updatedFilter : filter,
    );
    onFilterChange({
      ...filterState,
      filters: newFilters,
    });
  };

  const handleLogicalOperatorChange = useCallback(
    (index: number, newOperator: LogicalOperator) => {
      const newOperators = [...(filterState.logicalOperators || [])];
      newOperators[index - 1] = newOperator;
      onFilterChange({
        ...filterState,
        logicalOperators: newOperators,
      });
    },
    [filterState, onFilterChange],
  );

  return filterState.filters.length > 0 ? (
    <TableFilterContentOuter>
      <TableFilterConentHeader
        title="Filters"
        description="Filters are applied in the order they are added."
      />
      <Separator />
      <TableFilterConentInner>
        {filterState.filters.map((filter, index) => (
          <FilterRow
            key={index}
            filter={filter}
            index={index}
            columns={columns}
            logicalOperator={
              index > 0
                ? filterState.logicalOperators?.[index - 1] || "and"
                : undefined
            }
            onUpdate={handleUpdateFilter}
            onRemove={handleRemoveFilter}
            onLogicalOperatorChange={handleLogicalOperatorChange}
          />
        ))}
      </TableFilterConentInner>
    </TableFilterContentOuter>
  ) : (
    <TableFilterContentEmptyStat
      title="Filters"
      description="Add filters to refine your results."
    />
  );
}
