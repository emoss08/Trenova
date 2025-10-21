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
import {
  defaultFilterOperators,
  getAvailableOperators,
} from "@/lib/data-table-utils";
import type {
  FilterStateSchema,
  LogicalOperator,
} from "@/lib/schemas/table-configuration-schema";
import { EnhancedColumnDef } from "@/types/data-table";
import { faTrashCan } from "@fortawesome/pro-regular-svg-icons";
import { FilterValueInput } from "./data-table-filter-value-input";

interface FilterRowProps {
  filter: FilterStateSchema["filters"][number];
  index: number;
  columns: EnhancedColumnDef<any>[];
  logicalOperator?: LogicalOperator;
  onUpdate: (
    index: number,
    filter: FilterStateSchema["filters"][number],
  ) => void;
  onRemove: (index: number) => void;
  onLogicalOperatorChange?: (index: number, operator: LogicalOperator) => void;
}

export function FilterRow({
  filter,
  index,
  columns,
  logicalOperator,
  onUpdate,
  onRemove,
  onLogicalOperatorChange,
}: FilterRowProps) {
  const column = columns.find(
    (col) => (col.meta?.apiField || col.id) === filter.field,
  );

  if (!column) return null;

  const filterType = column.meta?.filterType || "text";
  const availableOperators = getAvailableOperators(filterType);

  const handleOperatorChange = (operator: string) => {
    onUpdate(index, {
      ...filter,
      operator: operator as FilterStateSchema["filters"][number]["operator"],
    });
  };

  const handleValueChange = (value: any) => {
    onUpdate(index, { ...filter, value });
  };

  const handleFieldChange = (newField: string) => {
    const newColumn = columns.find(
      (col) => (col.meta?.apiField || col.id) === newField,
    );
    if (!newColumn) return;

    const newFilterType = newColumn.meta?.filterType || "text";
    const newDefaultOperator =
      newColumn.meta?.defaultFilterOperator ||
      defaultFilterOperators[newFilterType];

    onUpdate(index, {
      field: newField,
      operator: newDefaultOperator,
      value: "",
    });
  };

  const filterableColumns = columns.filter((col) => col.meta?.filterable);

  return (
    <FilterRowInner>
      {logicalOperator && onLogicalOperatorChange ? (
        <Select
          value={logicalOperator}
          onValueChange={(value) =>
            onLogicalOperatorChange(index, value as LogicalOperator)
          }
        >
          <SelectTrigger className="w-full min-w-[60px] max-w-[72px]">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectGroup className="flex flex-col gap-0.5">
              <SelectItem value="and" className="cursor-pointer">
                and
              </SelectItem>
              <SelectItem disabled value="or" className="cursor-pointer">
                or
              </SelectItem>
            </SelectGroup>
          </SelectContent>
        </Select>
      ) : (
        <div className="min-w-[72px] text-center">
          <p className="text-sm text-muted-foreground">Where</p>
        </div>
      )}
      <Select value={filter.field} onValueChange={handleFieldChange}>
        <SelectTrigger className="w-auto min-w-[120px]">
          <SelectValue placeholder="Select field..." />
        </SelectTrigger>
        <SelectContent>
          <SelectGroup className="flex flex-col gap-0.5">
            {filterableColumns.map((col) => {
              const fieldId = col.meta?.apiField || col.id;
              const fieldLabel =
                typeof col.header === "string" ? col.header : fieldId;
              return (
                <SelectItem
                  key={fieldId}
                  value={fieldId as string}
                  className="cursor-pointer"
                >
                  {col.meta?.label || fieldLabel}
                </SelectItem>
              );
            })}
          </SelectGroup>
        </SelectContent>
      </Select>
      <Select value={filter.operator} onValueChange={handleOperatorChange}>
        <SelectTrigger className="w-auto min-w-[120px]">
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          <SelectGroup>
            {availableOperators.map((op) => (
              <SelectItem
                key={op.value}
                value={op.value}
                className="cursor-pointer"
              >
                {op.label}
              </SelectItem>
            ))}
          </SelectGroup>
        </SelectContent>
      </Select>
      <FilterValueInput
        filterType={filterType}
        operator={filter.operator}
        value={filter.value}
        options={column.meta?.filterOptions}
        onChange={handleValueChange}
      />
      <Button
        variant="outline"
        size="icon"
        onClick={() => onRemove(index)}
        className="shrink-0"
      >
        <Icon icon={faTrashCan} className="size-4" />
        <span className="sr-only">Remove filter</span>
      </Button>
    </FilterRowInner>
  );
}

function FilterRowInner({ children }: { children: React.ReactNode }) {
  return <div className="flex items-center gap-2">{children}</div>;
}
