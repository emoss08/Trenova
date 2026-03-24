import type { ShipmentStatus } from "@/types/shipment";
import { Badge } from "@/components/ui/badge";
import { ControlPosition, MapControl } from "@vis.gl/react-google-maps";
import { AlertTriangle } from "lucide-react";

const STATUS_FILTERS: { value: ShipmentStatus; label: string; color: string }[] = [
  { value: "InTransit", label: "In Transit", color: "#0891b2" },
  { value: "Assigned", label: "Assigned", color: "#8b5cf6" },
  { value: "New", label: "New", color: "#6b7280" },
  { value: "Canceled", label: "Canceled", color: "#dc2626" },
];

export type MapFilters = {
  delayedOnly: boolean;
  statuses: Set<ShipmentStatus>;
};

function FilterChip({
  active,
  color,
  onClick,
  children,
}: {
  active: boolean;
  color: string;
  onClick: () => void;
  children: React.ReactNode;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      className="flex items-center gap-1 rounded-md border px-2 py-1 text-2xs font-medium transition-colors"
      style={{
        backgroundColor: active ? `${color}18` : "transparent",
        borderColor: active ? `${color}40` : "var(--color-border)",
        color: active ? color : "var(--color-muted-foreground)",
      }}
    >
      {children}
    </button>
  );
}

export function MapFilterBar({
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
    <MapControl position={ControlPosition.TOP_LEFT}>
      <div className="m-2.5 flex items-center gap-1 rounded-lg border bg-background/95 px-2 py-1.5 shadow-sm backdrop-blur-sm">
        <FilterChip
          active={filters.delayedOnly}
          color="#dc2626"
          onClick={toggleDelayed}
        >
          <AlertTriangle className="size-3" />
          Delayed
        </FilterChip>

        <div className="mx-0.5 h-4 w-px shrink-0 bg-border" />

        {STATUS_FILTERS.map((s) => (
          <FilterChip
            key={s.value}
            active={filters.statuses.has(s.value)}
            color={s.color}
            onClick={() => toggleStatus(s.value)}
          >
            {s.label}
          </FilterChip>
        ))}

        <div className="mx-0.5 h-4 w-px shrink-0 bg-border" />

        <Badge variant="outline" className="text-2xs">
          {filteredCount}/{totalCount}
        </Badge>
      </div>
    </MapControl>
  );
}
