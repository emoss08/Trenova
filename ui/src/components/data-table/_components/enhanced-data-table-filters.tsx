import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Calendar } from "@/components/ui/calendar";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuTrigger,
  ShadcnDropdownMenuItem,
} from "@/components/ui/dropdown-menu";
import { Icon } from "@/components/ui/icons";
import { Input } from "@/components/ui/input";
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
import { Separator } from "@/components/ui/separator";
import {
  createFieldFilter,
  defaultFilterOperators,
  getAvailableOperators,
} from "@/lib/enhanced-data-table-utils";
import type {
  FilterStateSchema,
  LogicalOperator,
} from "@/lib/schemas/table-configuration-schema";
import type {
  EnhancedColumnDef,
  EnhancedDataTableConfig,
} from "@/types/enhanced-data-table";
import type { SelectOption } from "@/types/fields";
import { faBarsFilter, faTrashCan } from "@fortawesome/pro-regular-svg-icons";
import { format } from "date-fns";
import { useState } from "react";

interface EnhancedDataTableFiltersProps {
  columns: EnhancedColumnDef<any>[];
  filterState: FilterStateSchema;
  onFilterChange: (state: FilterStateSchema) => void;
  config?: EnhancedDataTableConfig;
}

export function EnhancedDataTableFilters({
  columns,
  filterState,
  onFilterChange,
  config,
}: EnhancedDataTableFiltersProps) {
  // Get filterable columns
  const filterableColumns = columns.filter((col) => col.meta?.filterable);

  // Handle adding a new filter
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

  // Handle removing a filter
  const handleRemoveFilter = (index: number) => {
    const newFilters = filterState.filters.filter((_, i) => i !== index);
    onFilterChange({
      ...filterState,
      filters: newFilters,
    });
  };

  // Handle updating a filter
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

  const firstAvailableColumn = filterableColumns.find(
    (col) =>
      !filterState.filters.some(
        (filter) => filter.field === (col.meta?.apiField || col.id),
      ),
  );

  // Clear all filters
  const handleClearAllFilters = () => {
    onFilterChange({
      ...filterState,
      filters: [],
    });
  };

  // Get available columns for adding filters (exclude already filtered ones)
  const availableColumns = filterableColumns.filter(
    (col) =>
      !filterState.filters.some(
        (filter) => filter.field === (col.meta?.apiField || col.id),
      ),
  );

  if (!config?.showFilterUI) {
    return null;
  }

  return (
    <div className="space-y-2">
      <div className="flex items-center gap-2">
        <Popover>
          <PopoverTrigger asChild>
            <Button variant="outline" className="flex items-center gap-2">
              <Icon icon={faBarsFilter} className="size-4" />
              <span className="text-sm">Filter</span>
              {filterState.filters.length > 0 && (
                <Badge
                  withDot={false}
                  className="h-[18.24px] rounded-[3.2px] px-[5.12px] text-xs"
                  variant="secondary"
                >
                  {filterState.filters.length}
                </Badge>
              )}
            </Button>
          </PopoverTrigger>
          <PopoverContent className="w-auto sm:min-w-[380px] p-0" align="start">
            <div className="space-y-3">
              {filterState.filters.length > 0 ? (
                <div className="flex flex-col gap-0.5">
                  <div className="flex items-center justify-between py-1 px-2">
                    <p className="text-sm font-medium">Filters</p>
                    <p className="text-xs text-muted-foreground">
                      Filters are applied in the order they are added.
                    </p>
                  </div>
                  <Separator />
                  <div className="flex max-h-[300px] flex-col gap-2 overflow-y-auto px-4 py-2">
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
                        onLogicalOperatorChange={(newOperator) => {
                          const newOperators = [
                            ...(filterState.logicalOperators || []),
                          ];
                          newOperators[index - 1] = newOperator;
                          onFilterChange({
                            ...filterState,
                            logicalOperators: newOperators,
                          });
                        }}
                      />
                    ))}
                  </div>
                </div>
              ) : (
                <div className="px-4 pt-4">
                  <p className="font-medium">No filters applied</p>
                  <p className="text-sm text-muted-foreground">
                    Add filters to refine your results.
                  </p>
                </div>
              )}

              {/* Add Filter Button */}
              <div className="flex w-full items-center gap-2 px-4 pb-2">
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
                  <Button
                    variant="outline"
                    size="lg"
                    onClick={handleClearAllFilters}
                  >
                    Reset Filters
                  </Button>
                )}
                {availableColumns.length === 0 && (
                  <p className="text-xs text-muted-foreground mt-1">
                    All filterable columns are already in use
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
  onLogicalOperatorChange?: (operator: LogicalOperator) => void;
}

function FilterRow({
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

  // Get filterable columns for the field selector
  const filterableColumns = columns.filter((col) => col.meta?.filterable);

  return (
    <div className="flex items-center gap-2">
      {logicalOperator && onLogicalOperatorChange ? (
        <Select value={logicalOperator} onValueChange={onLogicalOperatorChange}>
          <SelectTrigger className="w-full min-w-[60px]">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectGroup className="flex flex-col gap-0.5">
              <SelectItem value="and" className="cursor-pointer">
                and
              </SelectItem>
              <SelectItem value="or" className="cursor-pointer">
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
      {/* Field */}
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
                  {fieldLabel}
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
    </div>
  );
}

interface FilterValueInputProps {
  filterType: string;
  operator: string;
  value: any;
  options?: SelectOption[];
  onChange: (value: any) => void;
}

function FilterValueInput({
  filterType,
  operator,
  value,
  options,
  onChange,
}: FilterValueInputProps) {
  const [selectedOption, setSelectedOption] = useState<SelectOption | null>(
    options?.find((option) => option.value === value) || null,
  );

  // Handle null/undefined operators (isnull, isnotnull)
  if (operator === "isnull" || operator === "isnotnull") {
    return (
      <div className="flex items-center text-sm text-muted-foreground">
        No value needed
      </div>
    );
  }

  // Date range picker
  if (operator === "daterange") {
    return <DateRangeInput value={value} onChange={onChange} />;
  }

  // Multi-select for 'in' and 'notin' operators
  if ((operator === "in" || operator === "notin") && options) {
    return (
      <MultiSelectInput options={options} value={value} onChange={onChange} />
    );
  }

  // Select input for select columns
  if (filterType === "select" && options) {
    return (
      <Select
        value={value || ""}
        onValueChange={(value) => {
          setSelectedOption(
            options.find((option) => option.value === value) || null,
          );
          onChange(value);
        }}
      >
        <SelectTrigger className="w-[150px]">
          <SelectValue
            placeholder="Select value..."
            color={selectedOption?.color}
          />
        </SelectTrigger>
        <SelectContent>
          <SelectGroup className="flex flex-col gap-0.5">
            {options.map((option) => (
              <SelectItem
                key={String(option.value)}
                value={String(option.value)}
                description={option.description}
                icon={option.icon}
                color={option.color}
              >
                {option.label}
              </SelectItem>
            ))}
          </SelectGroup>
        </SelectContent>
      </Select>
    );
  }

  // Number input
  if (filterType === "number") {
    return (
      <Input
        type="number"
        value={value || ""}
        onChange={(e) => onChange(e.target.value ? Number(e.target.value) : "")}
        className="w-[150px]"
        placeholder="Enter number..."
      />
    );
  }

  if (filterType === "boolean") {
    return (
      <Select value={value || ""} onValueChange={onChange}>
        <SelectTrigger className="w-[150px]">
          <SelectValue placeholder="Select value..." />
        </SelectTrigger>
        <SelectContent>
          <SelectItem
            color="#15803d"
            value="true"
            className="cursor-pointer"
            title="True"
          >
            Yes
          </SelectItem>
          <SelectItem
            color="#b91c1c"
            value="false"
            className="cursor-pointer"
            title="False"
          >
            No
          </SelectItem>
        </SelectContent>
      </Select>
    );
  }

  // Default text input
  return (
    <Input
      type="text"
      value={value || ""}
      onChange={(e) => onChange(e.target.value)}
      className="w-[150px]"
      placeholder="Enter value..."
    />
  );
}

function DateRangeInput({
  value,
  onChange,
}: {
  value: any;
  onChange: (value: any) => void;
}) {
  const [isOpen, setIsOpen] = useState(false);
  const [dateRange, setDateRange] = useState<{ start?: Date; end?: Date }>(
    value || {},
  );

  const handleDateRangeChange = (range: { start?: Date; end?: Date }) => {
    setDateRange(range);
    onChange(range);
  };

  const formatDateRange = () => {
    if (dateRange.start && dateRange.end) {
      return `${format(dateRange.start, "MMM dd")} - ${format(dateRange.end, "MMM dd")}`;
    }
    if (dateRange.start) {
      return `From ${format(dateRange.start, "MMM dd")}`;
    }
    if (dateRange.end) {
      return `Until ${format(dateRange.end, "MMM dd")}`;
    }
    return "Select dates...";
  };

  return (
    <Popover open={isOpen} onOpenChange={setIsOpen}>
      <PopoverTrigger asChild>
        <Button variant="outline" className="w-[200px] justify-start">
          {formatDateRange()}
        </Button>
      </PopoverTrigger>
      <PopoverContent className="w-auto p-0" align="start">
        <div className="p-3">
          <div className="space-y-2">
            <div>
              <label className="text-sm font-medium">Start Date</label>
              <Calendar
                mode="single"
                selected={dateRange.start}
                onSelect={(date) =>
                  handleDateRangeChange({ ...dateRange, start: date })
                }
              />
            </div>
            <div>
              <label className="text-sm font-medium">End Date</label>
              <Calendar
                mode="single"
                selected={dateRange.end}
                onSelect={(date) =>
                  handleDateRangeChange({ ...dateRange, end: date })
                }
              />
            </div>
          </div>
          <div className="flex justify-end gap-2 mt-3">
            <Button
              variant="outline"
              size="sm"
              onClick={() => {
                setDateRange({});
                onChange({});
              }}
            >
              Clear
            </Button>
            <Button size="sm" onClick={() => setIsOpen(false)}>
              Done
            </Button>
          </div>
        </div>
      </PopoverContent>
    </Popover>
  );
}

function MultiSelectInput({
  options,
  value,
  onChange,
}: {
  options: { label: string; value: string }[];
  value: string[];
  onChange: (value: string[]) => void;
}) {
  const selectedValues = Array.isArray(value) ? value : [];

  const handleToggle = (optionValue: string) => {
    const newValues = selectedValues.includes(optionValue)
      ? selectedValues.filter((v) => v !== optionValue)
      : [...selectedValues, optionValue];
    onChange(newValues);
  };

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button variant="outline" className="w-[200px] justify-start">
          {selectedValues.length === 0
            ? "Select values..."
            : `${selectedValues.length} selected`}
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent className="w-[200px]">
        {options.map((option) => (
          <ShadcnDropdownMenuItem
            key={option.value}
            onClick={() => handleToggle(option.value)}
            className="cursor-pointer"
          >
            <div className="flex items-center gap-2" title={option.label}>
              <input
                type="checkbox"
                checked={selectedValues.includes(option.value)}
                readOnly
              />
              {option.label}
            </div>
          </ShadcnDropdownMenuItem>
        ))}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
