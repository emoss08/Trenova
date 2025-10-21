import { Input } from "@/components/ui/input";

import { Button } from "@/components/ui/button";
import { Calendar } from "@/components/ui/calendar";
import { Checkbox } from "@/components/ui/checkbox";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuTrigger,
  ShadcnDropdownMenuItem,
} from "@/components/ui/dropdown-menu";
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
import { cn } from "@/lib/utils";
import type { SelectOption } from "@/types/fields";
import { format } from "date-fns";
import { useState } from "react";

interface FilterValueInputProps {
  filterType: string;
  operator: string;
  value: any;
  options?: SelectOption[];
  onChange: (value: any) => void;
}

export function FilterValueInput({
  filterType,
  operator,
  value,
  options,
  onChange,
}: FilterValueInputProps) {
  const [selectedOption, setSelectedOption] = useState<SelectOption | null>(
    options?.find((option) => option.value === value) || null,
  );
  const booleanOptions = [
    { value: "true", label: "Yes", color: "#15803d" },
    { value: "false", label: "No", color: "#b91c1c" },
  ];

  if (operator === "isnull" || operator === "isnotnull") {
    return (
      <div className="flex items-center text-sm text-muted-foreground">
        No value needed
      </div>
    );
  }

  if (operator === "daterange") {
    return <DateRangeInput value={value} onChange={onChange} />;
  }

  if ((operator === "in" || operator === "notin") && options) {
    return (
      <MultiSelectInput options={options} value={value} onChange={onChange} />
    );
  }

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
            className="placeholder:text-muted"
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
      <Select
        value={value || ""}
        onValueChange={(value) => {
          setSelectedOption(
            booleanOptions?.find((option) => option.value === value) || null,
          );
          onChange(value);
        }}
      >
        <SelectTrigger className="w-[150px]">
          <SelectValue
            color={selectedOption?.color}
            placeholder="Select value..."
          />
        </SelectTrigger>
        <SelectContent>
          {booleanOptions.map((option) => (
            <SelectItem
              key={option.value}
              value={option.value}
              color={option.color}
              className="cursor-pointer"
              title={option.label}
            >
              {option.label}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
    );
  }

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
      <PopoverContent className="w-auto p-3" align="start">
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
        <Button
          variant="outline"
          className={cn(
            "group bg-primary/5 flex h-7 w-full items-center justify-between whitespace-nowrap rounded-md border border-muted-foreground/20",
            "px-1.5 py-2 text-xs ring-offset-background placeholder:text-muted-foreground outline-hidden",
            "data-[state=open]:border-foreground data-[state=open]:outline-hidden data-[state=open]:ring-4 data-[state=open]:ring-foreground/20",
            "focus-visible:border-foreground focus-visible:outline-hidden focus-visible:ring-4 focus-visible:ring-foreground/20",
            "transition-[border-color,box-shadow] duration-200 ease-in-out",
            "disabled:opacity-50 [&>span]:line-clamp-1 cursor-pointer disabled:cursor-not-allowed",
            "w-[200px] justify-start",
          )}
        >
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
              <Checkbox
                checked={selectedValues.includes(option.value)}
                onCheckedChange={() => handleToggle(option.value)}
              />
              {option.label}
            </div>
          </ShadcnDropdownMenuItem>
        ))}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
