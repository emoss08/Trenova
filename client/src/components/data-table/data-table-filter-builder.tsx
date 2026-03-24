"use no memo";
import { Button } from "@/components/ui/button";
import { Calendar } from "@/components/ui/calendar";
import { Input } from "@/components/ui/input";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Switch } from "@/components/ui/switch";
import {
  CONNECTOR_LABELS,
  generateFilterId,
  generateGroupId,
  getConnectorLabel,
  getDefaultOperatorForVariant,
  getOperatorLabel,
  getOperatorsForVariant,
  operatorRequiresValue,
} from "@/lib/data-table";
import { dateToUnixTimestamp, toDate, toUnixTimeStamp } from "@/lib/date";
import { cn, truncateText } from "@/lib/utils";
import type {
  FilterConnector,
  FilterGroupItem,
  FilterItem,
  FilterOperator,
  FilterVariant,
  SingleFilterItem,
} from "@/types/data-table";
import type { SelectOption } from "@/types/fields";
import type { ColumnDef } from "@tanstack/react-table";
import { CalendarIcon, FilterIcon, FolderPlusIcon, PlusIcon, TrashIcon } from "lucide-react";
import { useCallback, useMemo, useState } from "react";

type FilterableColumn = {
  id: string;
  apiField: string;
  label: string;
  filterType: FilterVariant;
  filterOptions?: SelectOption[];
  defaultOperator: FilterOperator;
};

type DataTableFilterBuilderProps<TData> = {
  columns: ColumnDef<TData>[];
  filters: FilterItem[];
  onFiltersChange: (filters: FilterItem[]) => void;
};

export default function DataTableFilterBuilder<TData>({
  columns,
  filters,
  onFiltersChange,
}: DataTableFilterBuilderProps<TData>) {
  const [open, setOpen] = useState(false);

  const filterableColumns = useMemo<FilterableColumn[]>(() => {
    return columns
      .filter((col) => {
        const meta = col.meta;
        return meta?.filterable === true && meta?.apiField;
      })
      .map((col) => {
        const meta = col.meta!;
        return {
          id: String("accessorKey" in col ? col.accessorKey : col.id),
          apiField: meta.apiField!,
          label: meta.label || String("accessorKey" in col ? col.accessorKey : col.id),
          filterType: (meta.filterType || "text") as FilterVariant,
          filterOptions: meta.filterOptions,
          defaultOperator:
            meta.defaultFilterOperator ||
            getDefaultOperatorForVariant((meta.filterType || "text") as FilterVariant),
        };
      });
  }, [columns]);

  const createNewFilter = useCallback(
    (connector: FilterConnector = "and"): SingleFilterItem | null => {
      if (filterableColumns.length === 0) return null;
      const column = filterableColumns[0];
      return {
        type: "filter",
        id: generateFilterId(),
        field: column.id,
        apiField: column.apiField,
        label: column.label,
        operator: column.defaultOperator,
        value: null,
        filterType: column.filterType,
        filterOptions: column.filterOptions,
        connector,
      };
    },
    [filterableColumns],
  );

  const handleAddFilter = useCallback(() => {
    const newFilter = createNewFilter(filters.length === 0 ? "and" : "and");
    if (!newFilter) return;
    onFiltersChange([...filters, newFilter]);
  }, [createNewFilter, filters, onFiltersChange]);

  const handleAddFilterGroup = useCallback(() => {
    const firstFilter = createNewFilter("or");
    const secondFilter = createNewFilter("or");
    if (!firstFilter || !secondFilter) return;

    const newGroup: FilterGroupItem = {
      type: "group",
      id: generateGroupId(),
      connector: "and",
      items: [firstFilter, secondFilter],
    };
    onFiltersChange([...filters, newGroup]);
  }, [createNewFilter, filters, onFiltersChange]);

  const handleUpdateFilter = useCallback(
    (filterId: string, updates: Partial<SingleFilterItem>) => {
      const updatedFilters = filters.map((item) => {
        if (item.type === "filter" && item.id === filterId) {
          return { ...item, ...updates };
        }
        if (item.type === "group") {
          return {
            ...item,
            items: item.items.map((f) => (f.id === filterId ? { ...f, ...updates } : f)),
          };
        }
        return item;
      });
      onFiltersChange(updatedFilters);
    },
    [filters, onFiltersChange],
  );

  const handleFilterFieldChange = useCallback(
    (filterId: string, columnId: string) => {
      const column = filterableColumns.find((c) => c.id === columnId);
      if (!column) return;
      handleUpdateFilter(filterId, {
        field: column.id,
        apiField: column.apiField,
        label: column.label,
        filterType: column.filterType,
        filterOptions: column.filterOptions,
        operator: column.defaultOperator,
        value: null,
      });
    },
    [filterableColumns, handleUpdateFilter],
  );

  const handleFilterOperatorChange = useCallback(
    (filterId: string, operator: FilterOperator) => {
      handleUpdateFilter(filterId, {
        operator,
        value: operatorRequiresValue(operator) ? undefined : null,
      });
    },
    [handleUpdateFilter],
  );

  const handleFilterValueChange = useCallback(
    (filterId: string, value: unknown) => {
      handleUpdateFilter(filterId, { value });
    },
    [handleUpdateFilter],
  );

  const handleConnectorChange = useCallback(
    (itemId: string, connector: FilterConnector) => {
      const updatedFilters = filters.map((item) => {
        if (item.id === itemId) {
          return { ...item, connector };
        }
        if (item.type === "group") {
          return {
            ...item,
            items: item.items.map((f) => (f.id === itemId ? { ...f, connector } : f)),
          };
        }
        return item;
      });
      onFiltersChange(updatedFilters);
    },
    [filters, onFiltersChange],
  );

  const handleRemoveFilter = useCallback(
    (filterId: string) => {
      const updatedFilters = filters
        .map((item) => {
          if (item.type === "filter" && item.id === filterId) {
            return null;
          }
          if (item.type === "group") {
            const remainingItems = item.items.filter((f) => f.id !== filterId);
            if (remainingItems.length === 0) return null;
            if (remainingItems.length === 1) {
              return {
                ...remainingItems[0],
                connector: item.connector,
              };
            }
            return { ...item, items: remainingItems };
          }
          return item;
        })
        .filter(Boolean) as FilterItem[];
      onFiltersChange(updatedFilters);
    },
    [filters, onFiltersChange],
  );

  const handleRemoveGroup = useCallback(
    (groupId: string) => {
      onFiltersChange(filters.filter((item) => item.id !== groupId));
    },
    [filters, onFiltersChange],
  );

  const handleAddFilterToGroup = useCallback(
    (groupId: string) => {
      const newFilter = createNewFilter("or");
      if (!newFilter) return;

      const updatedFilters = filters.map((item) => {
        if (item.type === "group" && item.id === groupId) {
          return { ...item, items: [...item.items, newFilter] };
        }
        return item;
      });
      onFiltersChange(updatedFilters);
    },
    [createNewFilter, filters, onFiltersChange],
  );

  const handleResetFilters = useCallback(() => {
    onFiltersChange([]);
  }, [onFiltersChange]);

  const totalFilters = useMemo(() => {
    return filters.reduce((count, item) => {
      if (item.type === "filter") return count + 1;
      return count + item.items.length;
    }, 0);
  }, [filters]);

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger
        render={
          <Button variant="outline" className="h-8">
            <FilterIcon className="size-3.5" />
            Filter
            {totalFilters > 0 && (
              <span className="ml-1.5 flex size-5 items-center justify-center rounded-md bg-muted font-mono text-xs">
                {totalFilters}
              </span>
            )}
          </Button>
        }
      />
      <PopoverContent
        className={cn("dark w-auto overflow-hidden p-0", filters.length === 0 && "min-w-[500px]")}
        align="start"
      >
        {filters.length === 0 ? (
          <div className="flex flex-col items-start gap-3 p-3">
            <div className="flex flex-col items-start">
              <h3 className="text-xl font-semibold">No filters applied</h3>
              <p className="text-sm text-muted-foreground">
                Add filters to narrow down your results.
              </p>
            </div>
            <Button onClick={handleAddFilter} disabled={filterableColumns.length === 0}>
              <PlusIcon className="size-3.5" />
              Add Filter
            </Button>
          </div>
        ) : (
          <>
            <div className="flex flex-col gap-2 p-3">
              {filters.map((item, index) =>
                item.type === "filter" ? (
                  <FilterRow
                    key={item.id}
                    filter={item}
                    index={index}
                    columns={filterableColumns}
                    onFieldChange={handleFilterFieldChange}
                    onOperatorChange={handleFilterOperatorChange}
                    onValueChange={handleFilterValueChange}
                    onConnectorChange={handleConnectorChange}
                    onRemove={handleRemoveFilter}
                  />
                ) : (
                  <FilterGroupRow
                    key={item.id}
                    group={item}
                    index={index}
                    columns={filterableColumns}
                    onFieldChange={handleFilterFieldChange}
                    onOperatorChange={handleFilterOperatorChange}
                    onValueChange={handleFilterValueChange}
                    onConnectorChange={handleConnectorChange}
                    onRemoveFilter={handleRemoveFilter}
                    onRemoveGroup={handleRemoveGroup}
                    onAddFilter={handleAddFilterToGroup}
                  />
                ),
              )}
            </div>
            <div className="flex items-center gap-2 border-t bg-sidebar p-2 dark:bg-background">
              <Button
                variant="outline"
                onClick={handleAddFilter}
                disabled={filterableColumns.length === 0}
              >
                <PlusIcon className="size-3.5" />
                Add Filter
              </Button>
              <Button
                variant="outline"
                onClick={handleAddFilterGroup}
                disabled={filterableColumns.length === 0}
              >
                <FolderPlusIcon className="size-3.5" />
                Add Filter Group
              </Button>
              {filters.length > 0 && (
                <Button variant="ghost" onClick={handleResetFilters}>
                  Reset Filters
                </Button>
              )}
            </div>
          </>
        )}
      </PopoverContent>
    </Popover>
  );
}

type FilterGroupRowProps = {
  group: FilterGroupItem;
  index: number;
  columns: FilterableColumn[];
  onFieldChange: (filterId: string, columnId: string) => void;
  onOperatorChange: (filterId: string, operator: FilterOperator) => void;
  onValueChange: (filterId: string, value: unknown) => void;
  onConnectorChange: (itemId: string, connector: FilterConnector) => void;
  onRemoveFilter: (filterId: string) => void;
  onRemoveGroup: (groupId: string) => void;
  onAddFilter: (groupId: string) => void;
};

function FilterGroupRow({
  group,
  index,
  columns,
  onFieldChange,
  onOperatorChange,
  onValueChange,
  onConnectorChange,
  onRemoveFilter,
  onRemoveGroup,
  onAddFilter,
}: FilterGroupRowProps) {
  return (
    <div className="flex w-full flex-col gap-1">
      <div className="flex items-start gap-2">
        {index === 0 ? (
          <span className="flex h-7 w-12 shrink-0 items-center text-sm text-muted-foreground">
            Where
          </span>
        ) : (
          <Select
            value={group.connector}
            onValueChange={(val) => onConnectorChange(group.id, val as FilterConnector)}
          >
            <SelectTrigger className="w-18">
              <SelectValue placeholder="Select Connector">
                {getConnectorLabel(group.connector)}
              </SelectValue>
            </SelectTrigger>
            <SelectContent>
              <SelectGroup>
                {Object.entries(CONNECTOR_LABELS).map(([value, label]) => (
                  <SelectItem key={value} value={value}>
                    {label}
                  </SelectItem>
                ))}
              </SelectGroup>
            </SelectContent>
          </Select>
        )}

        <div className="flex-1 rounded-md border border-dashed bg-muted/30 p-2">
          <div className="flex flex-col gap-1.5">
            {group.items.map((filter, filterIndex) => (
              <FilterRow
                key={filter.id}
                filter={filter}
                index={filterIndex}
                columns={columns}
                onFieldChange={onFieldChange}
                onOperatorChange={onOperatorChange}
                onValueChange={onValueChange}
                onConnectorChange={onConnectorChange}
                onRemove={onRemoveFilter}
                isNested
              />
            ))}
          </div>
          <Button
            variant="ghost"
            className="mt-2 h-6 text-xs"
            onClick={() => onAddFilter(group.id)}
          >
            <PlusIcon className="mr-1 size-3" />
            Add Filter to group
          </Button>
        </div>

        <Button
          variant="ghost"
          size="icon"
          className="size-7 text-muted-foreground hover:text-destructive"
          onClick={() => onRemoveGroup(group.id)}
        >
          <TrashIcon className="size-4" />
        </Button>
      </div>
    </div>
  );
}

type FilterRowProps = {
  filter: SingleFilterItem;
  index: number;
  columns: FilterableColumn[];
  onFieldChange: (filterId: string, columnId: string) => void;
  onOperatorChange: (filterId: string, operator: FilterOperator) => void;
  onValueChange: (filterId: string, value: unknown) => void;
  onConnectorChange: (filterId: string, connector: FilterConnector) => void;
  onRemove: (filterId: string) => void;
  isNested?: boolean;
};

function FilterRow({
  filter,
  index,
  columns,
  onFieldChange,
  onOperatorChange,
  onValueChange,
  onConnectorChange,
  onRemove,
  isNested = false,
}: FilterRowProps) {
  const operators = getOperatorsForVariant(filter.filterType);
  const needsValue = operatorRequiresValue(filter.operator);

  return (
    <div className="flex items-center gap-2">
      {index === 0 ? (
        <span className="w-12 shrink-0 text-sm text-muted-foreground">
          {isNested ? "" : "Where"}
        </span>
      ) : (
        <Select
          value={filter.connector}
          onValueChange={(val) => onConnectorChange(filter.id, val as FilterConnector)}
        >
          <SelectTrigger className="w-18">
            <SelectValue>{getConnectorLabel(filter.connector)}</SelectValue>
          </SelectTrigger>
          <SelectContent>
            <SelectGroup>
              {Object.entries(CONNECTOR_LABELS).map(([value, label]) => (
                <SelectItem key={value} value={value}>
                  {label}
                </SelectItem>
              ))}
            </SelectGroup>
          </SelectContent>
        </Select>
      )}

      <Select value={filter.field} onValueChange={(val) => onFieldChange(filter.id, val ?? "")}>
        <SelectTrigger className="w-28">
          <SelectValue>{filter.label}</SelectValue>
        </SelectTrigger>
        <SelectContent>
          <SelectGroup>
            {columns.map((col) => (
              <SelectItem key={col.id} value={col.id}>
                {col.label}
              </SelectItem>
            ))}
          </SelectGroup>
        </SelectContent>
      </Select>

      <Select
        value={filter.operator}
        onValueChange={(val) => onOperatorChange(filter.id, val as FilterOperator)}
      >
        <SelectTrigger className="w-36">
          <SelectValue>{getOperatorLabel(filter.operator)}</SelectValue>
        </SelectTrigger>
        <SelectContent>
          <SelectGroup>
            {operators.map((op) => (
              <SelectItem key={op} value={op}>
                {getOperatorLabel(op)}
              </SelectItem>
            ))}
          </SelectGroup>
        </SelectContent>
      </Select>

      {needsValue ? (
        <div className="w-36">
          <FilterValueInput filter={filter} onChange={(val) => onValueChange(filter.id, val)} />
        </div>
      ) : (
        <div className="w-36" />
      )}

      <Button
        variant="ghost"
        size="icon"
        className="size-7 text-muted-foreground hover:text-destructive"
        onClick={() => onRemove(filter.id)}
      >
        <TrashIcon className="size-4" />
      </Button>
    </div>
  );
}

type FilterValueInputProps = {
  filter: SingleFilterItem;
  onChange: (value: unknown) => void;
};

function FilterValueInput({ filter, onChange }: FilterValueInputProps) {
  const { filterType, operator, value, filterOptions } = filter;

  const selectedValue = filterOptions?.find((option) => option.value === value)?.label;

  if (filterType === "select" && filterOptions) {
    if (operator === "in" || operator === "notin") {
      const selectedValues = Array.isArray(value) ? (value as string[]) : [];
      const selectedLabels = selectedValues
        .map((v) => filterOptions.find((o) => o.value === v)?.label)
        .filter(Boolean);

      return (
        <Select multiple value={selectedValues} onValueChange={onChange}>
          <SelectTrigger className="w-full">
            <SelectValue>
              {selectedLabels.length > 0
                ? selectedLabels.length === 1
                  ? selectedLabels[0]
                  : `${selectedLabels.length} Selected`
                : "Select Values"}
            </SelectValue>
          </SelectTrigger>
          <SelectContent>
            <SelectGroup>
              {filterOptions.map((option) => (
                <SelectItem key={String(option.value)} value={option.value as string}>
                  {option.label}
                </SelectItem>
              ))}
            </SelectGroup>
          </SelectContent>
        </Select>
      );
    }

    return (
      <Select value={value as string} onValueChange={onChange}>
        <SelectTrigger className="w-full">
          <SelectValue>{selectedValue}</SelectValue>
        </SelectTrigger>
        <SelectContent>
          <SelectGroup>
            {filterOptions.map((option) => (
              <SelectItem key={String(option.value)} value={option.value as string}>
                {option.label}
              </SelectItem>
            ))}
          </SelectGroup>
        </SelectContent>
      </Select>
    );
  }

  if (filterType === "boolean") {
    return (
      <div className="flex h-7 items-center gap-2 rounded-md border bg-background px-2.5">
        <Switch checked={value === true} onCheckedChange={(checked) => onChange(checked)} />
        <span className="text-sm">{value === true ? "Yes" : "No"}</span>
      </div>
    );
  }

  if (filterType === "date") {
    if (operator === "lastndays" || operator === "nextndays") {
      return (
        <Input
          type="number"
          min={1}
          value={(value as number) || ""}
          onChange={(e) => onChange(Number(e.target.value) || 1)}
          placeholder="Days"
        />
      );
    }

    if (operator === "daterange") {
      const dateValue = (value as { from?: number; to?: number }) || {
        from: undefined,
        to: undefined,
      };

      return (
        <Popover>
          <PopoverTrigger
            render={
              <Button
                variant="outline"
                className={cn(
                  "h-8 w-full justify-start truncate rounded-md bg-muted text-left text-sm font-normal",
                  (!dateValue.from || !dateValue.to) && "text-muted-foreground",
                )}
              >
                <CalendarIcon className="size-4" />
                {dateValue.from && dateValue.to
                  ? truncateText(
                      `${toDate(dateValue.from)?.toLocaleDateString()} - ${toDate(dateValue.to)?.toLocaleDateString()}`,
                      13,
                    )
                  : "Pick date range"}
              </Button>
            }
          />
          <PopoverContent className="dark w-auto p-0" align="start">
            <Calendar
              mode="range"
              className="w-[250px]"
              selected={{
                from: dateValue.from ? toDate(dateValue.from) : undefined,
                to: dateValue.to ? toDate(dateValue.to) : undefined,
              }}
              onSelect={(date) => {
                onChange(
                  date && (date.from || date.to)
                    ? {
                        from: toUnixTimeStamp(date.from),
                        to: toUnixTimeStamp(date.to),
                      }
                    : { from: undefined, to: undefined },
                );
              }}
            />
          </PopoverContent>
        </Popover>
      );
    }

    const dateValue = typeof value === "number" ? toDate(value) : undefined;
    return (
      <Popover>
        <PopoverTrigger
          render={
            <Button
              variant="outline"
              className={cn(
                "h-8 w-full justify-start rounded-md bg-muted text-left text-sm font-normal",
                !dateValue && "text-muted-foreground",
              )}
            >
              <CalendarIcon className="mr-2 size-4" />
              {dateValue ? dateValue.toLocaleDateString() : "Pick date"}
            </Button>
          }
        />
        <PopoverContent className="w-auto p-0" align="start">
          <Calendar
            mode="single"
            selected={dateValue}
            onSelect={(date) => onChange(date ? dateToUnixTimestamp(date) : null)}
          />
        </PopoverContent>
      </Popover>
    );
  }

  if (filterType === "number") {
    return (
      <Input
        type="number"
        value={(value as number) || ""}
        onChange={(e) => onChange(Number(e.target.value))}
        placeholder="Value"
      />
    );
  }

  return (
    <Input
      type="text"
      value={(value as string) || ""}
      onChange={(e) => onChange(e.target.value)}
      placeholder="Value"
    />
  );
}
