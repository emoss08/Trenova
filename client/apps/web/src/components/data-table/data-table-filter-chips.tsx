"use no memo";
import { Button } from "@/components/ui/button";
import { getOperatorLabel, operatorRequiresValue, stringifyUnknown } from "@/lib/data-table";
import type { FilterGroupItem, FilterItem, SingleFilterItem } from "@/types/data-table";
import { SearchIcon, XIcon } from "lucide-react";

type DataTableFilterChipsProps = {
  filters: FilterItem[];
  onFiltersChange: (filters: FilterItem[]) => void;
  query: string;
  onClearQuery: () => void;
};

function formatFilterValue(filter: SingleFilterItem): string {
  if (!operatorRequiresValue(filter.operator)) return "";
  const { value } = filter;
  if (Array.isArray(value)) {
    const labels = value.map((v) => {
      const stringValue = stringifyUnknown(v);
      return (
        filter.filterOptions?.find((o) => stringifyUnknown(o.value) === stringValue)?.label ??
        stringValue
      );
    });
    return labels.length > 2
      ? `${labels.slice(0, 2).join(", ")} +${labels.length - 2}`
      : labels.join(", ");
  }
  if (typeof value === "boolean") return value ? "Yes" : "No";
  const stringValue = stringifyUnknown(value);
  const optionLabel = filter.filterOptions?.find(
    (o) => stringifyUnknown(o.value) === stringValue,
  )?.label;
  return optionLabel ?? stringValue;
}

function FilterChip({ label, onRemove }: { label: React.ReactNode; onRemove: () => void }) {
  return (
    <span className="flex h-6 items-center gap-1 rounded-md border border-border bg-muted/50 pr-1 pl-2 text-xs">
      {label}
      <Button
        type="button"
        variant="ghost"
        size="icon-xs"
        className="size-4 rounded-sm text-muted-foreground hover:text-foreground"
        onClick={onRemove}
        aria-label="Remove filter"
      >
        <XIcon className="size-3" />
      </Button>
    </span>
  );
}

export default function DataTableFilterChips({
  filters,
  onFiltersChange,
  query,
  onClearQuery,
}: DataTableFilterChipsProps) {
  const hasQuery = query !== "";
  if (filters.length === 0 && !hasQuery) return null;

  const removeFilter = (id: string) => {
    onFiltersChange(filters.filter((f) => f.id !== id));
  };

  const clearAll = () => {
    onFiltersChange([]);
    if (hasQuery) onClearQuery();
  };

  return (
    <div className="flex flex-wrap items-center gap-1.5">
      {hasQuery && (
        <FilterChip
          label={
            <span className="flex items-center gap-1">
              <SearchIcon className="size-3 text-muted-foreground" />
              <span className="max-w-40 truncate font-medium">{query}</span>
            </span>
          }
          onRemove={onClearQuery}
        />
      )}
      {filters.map((filter) => {
        if (filter.type === "group") {
          const group = filter as FilterGroupItem;
          return (
            <FilterChip
              key={group.id}
              label={
                <span>
                  <span className="text-muted-foreground">Group·</span>{" "}
                  <span className="font-medium">{group.items.length} conditions</span>
                </span>
              }
              onRemove={() => removeFilter(group.id)}
            />
          );
        }

        const single = filter as SingleFilterItem;
        const value = formatFilterValue(single);
        return (
          <FilterChip
            key={single.id}
            label={
              <span className="max-w-64 truncate">
                <span className="font-medium">{single.label}</span>{" "}
                <span className="text-muted-foreground">{getOperatorLabel(single.operator)}</span>
                {value && <span className="font-medium"> {value}</span>}
              </span>
            }
            onRemove={() => removeFilter(single.id)}
          />
        );
      })}
      {(filters.length > 1 || (filters.length > 0 && hasQuery)) && (
        <Button
          type="button"
          variant="ghost"
          size="xs"
          className="h-6 px-2 text-xs text-muted-foreground hover:text-foreground"
          onClick={clearAll}
        >
          Clear all
        </Button>
      )}
    </div>
  );
}
