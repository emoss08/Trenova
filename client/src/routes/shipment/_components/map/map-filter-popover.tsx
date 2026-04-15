import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import type { ShipmentStatus } from "@/types/shipment";
import { FilterIcon } from "lucide-react";
import type { MapFilters } from "./map-filter-bar";

const STATUS_FILTERS: { value: ShipmentStatus; label: string; color: string }[] = [
  { value: "InTransit", label: "In Transit", color: "#0891b2" },
  { value: "Assigned", label: "Assigned", color: "#8b5cf6" },
  { value: "New", label: "New", color: "#6b7280" },
  { value: "Delayed", label: "Delayed", color: "#dc2626" },
  { value: "Canceled", label: "Canceled", color: "#9ca3af" },
];

export function MapFilterPopover({
  filters,
  onFiltersChange,
  totalCount,
  filteredCount,
}: {
  filters: MapFilters;
  onFiltersChange: (filters: MapFilters) => void;
  totalCount: number;
  filteredCount: number;
}) {
  const hasActiveFilters = filters.statuses.size > 0;

  const toggleStatus = (status: ShipmentStatus) => {
    const next = new Set(filters.statuses);
    if (next.has(status)) {
      next.delete(status);
    } else {
      next.add(status);
    }
    onFiltersChange({ ...filters, statuses: next });
  };

  return (
    <Popover>
      <Tooltip>
        <TooltipTrigger
          render={
            <PopoverTrigger
              render={
                <Button
                  variant="outline"
                  size="icon"
                  className="relative size-8 bg-background shadow-sm"
                />
              }
            />
          }
        >
          <FilterIcon className="size-4" />
          {hasActiveFilters && (
            <span className="absolute top-1 right-1 size-2 rounded-full bg-brand" />
          )}
        </TooltipTrigger>
        <TooltipContent side="left">Filters</TooltipContent>
      </Tooltip>
      <PopoverContent side="left" sideOffset={8} className="w-44 p-2.5">
        <span className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">
          Status
        </span>
        <div className="mt-1 flex flex-col">
          {STATUS_FILTERS.map((s) => (
            <label
              key={s.value}
              className="flex cursor-pointer items-center gap-2 rounded-md px-1.5 py-1 text-sm hover:bg-accent"
            >
              <Checkbox
                checked={filters.statuses.has(s.value)}
                onCheckedChange={() => toggleStatus(s.value)}
              />
              <span
                className="inline-block size-2 shrink-0 rounded-full"
                style={{ backgroundColor: s.color }}
              />
              {s.label}
            </label>
          ))}
        </div>
        <div className="mt-2 flex items-center justify-between border-t pt-2">
          <span className="text-2xs text-muted-foreground">Showing</span>
          <Badge variant="outline" className="text-2xs">
            {filteredCount}/{totalCount}
          </Badge>
        </div>
      </PopoverContent>
    </Popover>
  );
}
