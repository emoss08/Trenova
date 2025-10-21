import { Button } from "@/components/ui/button";
import { Icon } from "@/components/ui/icons";
import {
    Select,
    SelectContent,
    SelectGroup,
    SelectItem,
    SelectTrigger,
    SelectValue,
} from "@/components/ui/select";
import type {
    FilterStateSchema,
    SortFieldSchema,
} from "@/lib/schemas/table-configuration-schema";
import { EnhancedColumnDef } from "@/types/data-table";
import { faTrashCan } from "@fortawesome/pro-regular-svg-icons";
import { ArrowDown, ArrowUp } from "lucide-react";

interface DataTableSortRowProps {
  sort: FilterStateSchema["sort"][number];
  index: number;
  columns: EnhancedColumnDef<any>[];
  sortState: FilterStateSchema["sort"];
  onUpdate: (index: number, sort: FilterStateSchema["sort"][number]) => void;
  onRemove: (index: number) => void;
}

export function DataTableSortRow({
  sort,
  index,
  columns,
  sortState,
  onUpdate,
  onRemove,
}: DataTableSortRowProps) {
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

  const sortableColumns = columns.filter((col) => {
    if (!col.meta?.sortable) return false;
    const fieldId = col.meta?.apiField || col.id;
    return (
      fieldId === sort.field || !sortState.some((s) => s.field === fieldId)
    );
  });

  return (
    <DataTableSortRowInner>
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
    </DataTableSortRowInner>
  );
}

function DataTableSortRowInner({ children }: { children: React.ReactNode }) {
  return <div className="flex items-center gap-2">{children}</div>;
}
