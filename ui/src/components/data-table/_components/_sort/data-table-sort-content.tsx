import { Button } from "@/components/ui/button";
import { createSortField } from "@/lib/data-table-utils";
import type { FilterStateSchema } from "@/lib/schemas/table-configuration-schema";
import { Config, EnhancedColumnDef } from "@/types/data-table";
import { DataTableContentFooter, DataTableContentInner } from "../_shared";
import { AppliedSortContent } from "./applied-sort-content";

type DataTableSortContentProps = {
  columns: EnhancedColumnDef<any>[];
  sortState: FilterStateSchema["sort"];
  onSortChange: (sort: FilterStateSchema["sort"]) => void;
  config?: Config;
};

export function DataTableSortContent({
  columns,
  sortState,
  onSortChange,
  config,
}: DataTableSortContentProps) {
  const sortableColumns = columns.filter((col) => col.meta?.sortable);

  const handleAddSort = (column: EnhancedColumnDef<any> | undefined) => {
    if (!column) return;
    const fieldId = column.meta?.apiField || column.id;
    const newSort = createSortField(fieldId!, "asc");
    const newSortState = [...sortState, newSort];
    onSortChange(newSortState);
  };

  const firstAvailableColumn = sortableColumns.find(
    (col) =>
      !sortState.some((sort) => sort.field === (col.meta?.apiField || col.id)),
  );

  const handleClearAllSort = () => {
    onSortChange([]);
  };

  const availableColumns = config?.enableMultiSort
    ? sortableColumns.filter(
        (col) =>
          !sortState.some(
            (sort) => sort.field === (col.meta?.apiField || col.id),
          ),
      )
    : sortableColumns;
  return (
    <DataTableContentInner>
      <AppliedSortContent
        sortState={sortState}
        columns={columns}
        onSortChange={onSortChange}
      />
      <DataTableContentFooter>
        <Button
          size="lg"
          onClick={() => {
            handleAddSort(firstAvailableColumn);
          }}
          disabled={availableColumns.length === 0}
        >
          Add Sort
        </Button>
        {sortState.length > 0 && (
          <Button variant="outline" size="lg" onClick={handleClearAllSort}>
            Reset Sorting
          </Button>
        )}
        {availableColumns.length === 0 && (
          <p className="text-xs text-muted-foreground mt-1">
            All sortable columns are already in use
          </p>
        )}
      </DataTableContentFooter>
    </DataTableContentInner>
  );
}
