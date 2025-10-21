import { Separator } from "@/components/ui/separator";
import { FilterStateSchema } from "@/lib/schemas/table-configuration-schema";
import { EnhancedColumnDef } from "@/types/data-table";
import {
  TableFilterConentHeader,
  TableFilterConentInner,
  TableFilterContentEmptyStat,
  TableFilterContentOuter,
} from "../_shared";
import { DataTableSortRow } from "./data-table-sort-row";

export function AppliedSortContent({
  sortState,
  columns,
  onSortChange,
}: {
  sortState: FilterStateSchema["sort"];
  columns: EnhancedColumnDef<any>[];
  onSortChange: (sort: FilterStateSchema["sort"]) => void;
}) {
  const handleUpdateSort = (
    index: number,
    updatedSort: FilterStateSchema["sort"][number],
  ) => {
    const newSort = sortState.map((sort, i) =>
      i === index ? updatedSort : sort,
    );
    onSortChange(newSort);
  };

  const handleRemoveSort = (index: number) => {
    const newSort = sortState.filter((_, i) => i !== index);
    onSortChange(newSort);
  };

  return sortState.length > 0 ? (
    <TableFilterContentOuter>
      <TableFilterConentHeader
        title="Sort"
        description="Sorting is applied in the order they are added."
      />
      <Separator />
      <TableFilterConentInner>
        {sortState.map((sort, index) => (
          <DataTableSortRow
            key={`${sort.field}-${sort.direction}-${index}`}
            sort={sort}
            index={index}
            columns={columns}
            sortState={sortState}
            onUpdate={handleUpdateSort}
            onRemove={handleRemoveSort}
          />
        ))}
      </TableFilterConentInner>
    </TableFilterContentOuter>
  ) : (
    <TableFilterContentEmptyStat
      title="Sorting"
      description="Add sorting to organize your rows."
    />
  );
}
