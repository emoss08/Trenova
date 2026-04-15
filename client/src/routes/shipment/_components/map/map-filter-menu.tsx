import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuCheckboxItem,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import type { ShipmentStatus } from "@/types/shipment";
import { AlertTriangleIcon, FilterIcon } from "lucide-react";
import type { MapFilters } from "./map-filter-bar";

const STATUS_FILTERS: { value: ShipmentStatus; label: string; color: string }[] = [
  { value: "InTransit", label: "In Transit", color: "#0891b2" },
  { value: "Assigned", label: "Assigned", color: "#8b5cf6" },
  { value: "New", label: "New", color: "#6b7280" },
  { value: "Canceled", label: "Canceled", color: "#dc2626" },
];

export function MapFilterMenu({
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
  const hasActiveFilters = filters.delayedOnly || filters.statuses.size > 0;

  const toggleDelayed = () =>
    onFiltersChange({ ...filters, delayedOnly: !filters.delayedOnly });

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
    <DropdownMenu>
      <Tooltip>
        <TooltipTrigger
          render={
            <DropdownMenuTrigger
              render={
                <Button variant="ghost" size="icon-sm" className="relative" />
              }
            />
          }
        >
          <FilterIcon className="size-3.5" />
          {hasActiveFilters && (
            <span className="absolute top-0.5 right-0.5 size-1.5 rounded-full bg-brand" />
          )}
        </TooltipTrigger>
        <TooltipContent side="bottom">Filters</TooltipContent>
      </Tooltip>
      <DropdownMenuContent align="end" sideOffset={6} className="w-48">
        <DropdownMenuGroup>
          <DropdownMenuLabel>Filters</DropdownMenuLabel>
          <DropdownMenuCheckboxItem
            checked={filters.delayedOnly}
            onCheckedChange={toggleDelayed}
          >
            <AlertTriangleIcon className="size-3.5 text-red-500" />
            <span>Delayed Only</span>
          </DropdownMenuCheckboxItem>
          <DropdownMenuSeparator />
          <DropdownMenuLabel>Status</DropdownMenuLabel>
          {STATUS_FILTERS.map((s) => (
            <DropdownMenuCheckboxItem
              key={s.value}
              checked={filters.statuses.has(s.value)}
              onCheckedChange={() => toggleStatus(s.value)}
            >
              <span
                className="inline-block size-2 shrink-0 rounded-full"
                style={{ backgroundColor: s.color }}
              />
              <span>{s.label}</span>
            </DropdownMenuCheckboxItem>
          ))}
        </DropdownMenuGroup>
        <DropdownMenuSeparator />
        <div className="flex items-center justify-between px-1.5 py-1">
          <span className="text-2xs text-muted-foreground">Showing</span>
          <Badge variant="outline" className="text-2xs">
            {filteredCount}/{totalCount}
          </Badge>
        </div>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
