"use no memo";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Icon } from "@/components/ui/icons";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { createSortField } from "@/lib/enhanced-data-table-utils";
import type {
  FilterStateSchema,
  SortFieldSchema,
} from "@/lib/schemas/table-configuration-schema";
import type {
  EnhancedColumnDef,
  EnhancedDataTableConfig,
} from "@/types/enhanced-data-table";
import {
  faArrowUpArrowDown,
  faTrashCan,
} from "@fortawesome/pro-regular-svg-icons";
import { ArrowDown, ArrowUp } from "lucide-react";

interface EnhancedDataTableSortProps {
  columns: EnhancedColumnDef<any>[];
  sortState: FilterStateSchema["sort"];
  onSortChange: (sort: FilterStateSchema["sort"]) => void;
  config?: EnhancedDataTableConfig;
}

export function EnhancedDataTableSort({
  columns,
  sortState,
  onSortChange,
  config,
}: EnhancedDataTableSortProps) {
  // Get sortable columns
  const sortableColumns = columns.filter((col) => col.meta?.sortable);

  // Handle adding a new sort field
  const handleAddSort = (column: EnhancedColumnDef<any> | undefined) => {
    if (!column) return;
    const fieldId = column.meta?.apiField || column.id;
    const newSort = createSortField(fieldId!, "asc");
    const newSortState = [...sortState, newSort];
    onSortChange(newSortState);
  };

  // Handle removing a sort field
  const handleRemoveSort = (index: number) => {
    const newSort = sortState.filter((_, i) => i !== index);
    onSortChange(newSort);
  };

  const firstAvailableColumn = sortableColumns.find(
    (col) =>
      !sortState.some((sort) => sort.field === (col.meta?.apiField || col.id)),
  );

  // Handle updating a sort field
  const handleUpdateSort = (
    index: number,
    updatedSort: FilterStateSchema["sort"][number],
  ) => {
    const newSort = sortState.map((sort, i) =>
      i === index ? updatedSort : sort,
    );
    onSortChange(newSort);
  };

  // Clear all sorting
  const handleClearAllSort = () => {
    onSortChange([]);
  };

  // Get available columns for adding sort (exclude already sorted ones if not multi-sort)
  const availableColumns = config?.enableMultiSort
    ? sortableColumns.filter(
        (col) =>
          !sortState.some(
            (sort) => sort.field === (col.meta?.apiField || col.id),
          ),
      )
    : sortableColumns;

  if (!config?.showSortUI) {
    return null;
  }

  return (
    <div className="space-y-2">
      {/* Sort controls */}
      <div className="flex items-center gap-2">
        <Popover>
          <PopoverTrigger asChild>
            <Button variant="outline" className="flex items-center gap-2">
              <Icon icon={faArrowUpArrowDown} className="size-4" />
              <span className="text-sm">Sort</span>
              {sortState.length > 0 && (
                <Badge
                  withDot={false}
                  className="h-[18.24px] rounded-[3.2px] px-[5.12px] text-xs"
                  variant="secondary"
                >
                  {sortState.length}
                </Badge>
              )}
            </Button>
          </PopoverTrigger>
          <PopoverContent className="w-auto sm:min-w-[380px]" align="start">
            <div className="space-y-3">
              {sortState.length > 0 ? (
                <div>
                  <p className="text-sm font-medium mb-2">Sort</p>
                  <div className="flex max-h-[300px] flex-col gap-2 overflow-y-auto p-1">
                    {sortState.map((sort, index) => (
                      <SortRow
                        key={`${sort.field}-${sort.direction}-${index}`}
                        sort={sort}
                        index={index}
                        columns={columns}
                        sortState={sortState}
                        onUpdate={handleUpdateSort}
                        onRemove={handleRemoveSort}
                      />
                    ))}
                  </div>
                </div>
              ) : (
                <div>
                  <p className="text-sm font-medium">No sorting applied</p>
                  <p className="text-sm text-muted-foreground">
                    Add sorting to organize your rows.
                  </p>
                </div>
              )}

              <div className="flex w-full items-center gap-2">
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
                  <Button
                    variant="outline"
                    size="lg"
                    onClick={handleClearAllSort}
                  >
                    Reset Sorting
                  </Button>
                )}
                {availableColumns.length === 0 && (
                  <p className="text-xs text-muted-foreground mt-1">
                    All sortable columns are already in use
                  </p>
                )}
              </div>
            </div>
          </PopoverContent>
        </Popover>
      </div>
    </div>
  );
}

interface SortRowProps {
  sort: FilterStateSchema["sort"][number];
  index: number;
  columns: EnhancedColumnDef<any>[];
  sortState: FilterStateSchema["sort"];
  onUpdate: (index: number, sort: FilterStateSchema["sort"][number]) => void;
  onRemove: (index: number) => void;
}

function SortRow({
  sort,
  index,
  columns,
  sortState,
  onUpdate,
  onRemove,
}: SortRowProps) {
  const column = columns.find(
    (col) => (col.meta?.apiField || col.id) === sort.field,
  );

  if (!column) {
    return null;
  }

  const handleDirectionChange = (direction: SortFieldSchema["direction"]) => {
    onUpdate(index, {
      ...sort,
      direction: direction,
    });
  };

  const handleFieldChange = (field: string) => {
    onUpdate(index, {
      ...sort,
      field: field,
    });
  };

  // Get sortable columns but exclude already used fields (except current field)
  const sortableColumns = columns.filter((col) => {
    if (!col.meta?.sortable) return false;
    const fieldId = col.meta?.apiField || col.id;
    // Include current field or fields not already used
    return (
      fieldId === sort.field || !sortState.some((s) => s.field === fieldId)
    );
  });

  return (
    <div className="flex items-center gap-2">
      <Select value={sort.field} onValueChange={handleFieldChange}>
        <SelectTrigger className="min-w-[100px]">
          <SelectValue placeholder="Select field..." />
        </SelectTrigger>
        <SelectContent>
          <SelectGroup className="flex flex-col gap-0.5">
            {sortableColumns.map((col) => {
              const fieldId = col.meta?.apiField || col.id;
              const fieldLabel =
                typeof col.header === "string" ? col.header : fieldId;
              return (
                <SelectItem
                  key={fieldId}
                  value={fieldId as string}
                  className="cursor-pointer"
                >
                  {fieldLabel}
                </SelectItem>
              );
            })}
          </SelectGroup>
        </SelectContent>
      </Select>
      <Select value={sort.direction} onValueChange={handleDirectionChange}>
        <SelectTrigger className="w-auto min-w-[100px]">
          <SelectValue placeholder="Select direction..." />
        </SelectTrigger>
        <SelectContent>
          <SelectItem value="asc">
            <div className="flex items-center gap-2">
              <ArrowUp className="size-4" />
              Asc
            </div>
          </SelectItem>
          <SelectItem value="desc">
            <div className="flex items-center gap-2">
              <ArrowDown className="size-4" />
              Desc
            </div>
          </SelectItem>
        </SelectContent>
      </Select>

      <Button
        variant="outline"
        size="icon"
        onClick={() => onRemove(index)}
        className="shrink-0"
      >
        <Icon icon={faTrashCan} className="size-4" />
        <span className="sr-only">Remove sort</span>
      </Button>
    </div>
  );
}
