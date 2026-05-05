import { Spinner } from "@/components/ui/spinner";
import { formatToUserTimezone } from "@/lib/date";
import {
  getDestinationLocation,
  getOriginLocation,
  getOriginStop,
  getTotalMiles,
} from "@/lib/shipment-utils";
import { cn, formatCurrency } from "@/lib/utils";
import type { Shipment } from "@/types/shipment";
import { GripVerticalIcon } from "lucide-react";
import { useCommandCenterUrl } from "../url-state";
import { ModuleCard } from "./module-card";
import { useUnassignedShipments } from "./use-right-stack-data";

function parseDecimal(value: string | number | null | undefined): number {
  if (value === null || value === undefined) return 0;
  if (typeof value === "number") return value;
  const parsed = Number(value);
  return Number.isFinite(parsed) ? parsed : 0;
}

function pickupDisplay(shipment: Shipment): string {
  const stop = getOriginStop(shipment);
  if (!stop?.scheduledWindowStart) return "—";
  return formatToUserTimezone(stop.scheduledWindowStart, {
    showTimeZone: false,
    showSeconds: false,
  });
}

function priorityFor(s: Shipment): {
  label: "HOT" | "MED" | "LOW";
  tone: "danger" | "warning" | "muted";
} {
  if (s.status === "New") return { label: "HOT", tone: "danger" };
  if (s.status === "PartiallyAssigned") return { label: "MED", tone: "warning" };
  return { label: "LOW", tone: "muted" };
}

function equipCode(s: Shipment): string {
  return s.moves?.[0]?.assignment?.trailer?.equipmentType?.code ?? "—";
}

const PILL_TONE: Record<"danger" | "warning" | "muted", string> = {
  danger: "bg-destructive/12 text-destructive",
  warning: "bg-warning/15 text-warning",
  muted: "bg-muted text-muted-foreground",
};

export function UnassignedQueue() {
  const { data, isLoading } = useUnassignedShipments();
  const [, setUrl] = useCommandCenterUrl();
  const shipments = (data?.results ?? []) as Shipment[];

  const pendingRevenue = shipments.reduce(
    (total, s) => total + parseDecimal(s.totalChargeAmount as unknown as string),
    0,
  );

  return (
    <ModuleCard
      id="unassigned"
      title="Unassigned"
      count={data?.count}
      countTone="warning"
      rightSlot={
        <span className="font-table hidden text-[9.5px] tabular-nums text-muted-foreground sm:inline">
          {formatCurrency(pendingRevenue)} waiting
        </span>
      }
    >
      <div className="flex flex-col gap-1.5 p-2">
        {isLoading && (
          <div className="flex items-center gap-2 text-[10.5px] text-muted-foreground">
            <Spinner className="size-3" /> Loading…
          </div>
        )}
        {!isLoading && shipments.length === 0 && (
          <p className="py-3 text-center text-[10.5px] text-muted-foreground">
            All loads are assigned ✓
          </p>
        )}
        {shipments.map((s) => {
          const origin = getOriginLocation(s)?.code ?? "—";
          const dest = getDestinationLocation(s)?.code ?? "—";
          const revenue = parseDecimal(s.totalChargeAmount as unknown as string);
          const miles = getTotalMiles(s);
          const priority = priorityFor(s);
          return (
            <button
              key={s.id}
              type="button"
              onClick={() => s.id && setUrl({ expanded: s.id })}
              className="flex flex-col gap-1 rounded border border-border bg-muted/30 px-2 py-1.5 text-left transition-colors hover:border-foreground/20 hover:bg-muted/60"
            >
              <div className="flex items-center justify-between gap-1.5">
                <div className="flex items-center gap-1 min-w-0">
                  <GripVerticalIcon className="size-2.5 shrink-0 text-muted-foreground" />
                  <span className="font-table truncate text-[10.5px] font-semibold tabular-nums">
                    {origin} → {dest}
                  </span>
                </div>
                <span
                  className={cn(
                    "shrink-0 rounded px-1 py-px text-[8.5px] font-bold tracking-wide uppercase",
                    PILL_TONE[priority.tone],
                  )}
                >
                  {priority.label}
                </span>
              </div>
              <div className="flex items-center justify-between gap-2 text-[10px] text-muted-foreground">
                <span className="truncate">{s.customer?.name ?? "—"}</span>
                <span className="font-table shrink-0 tabular-nums">{equipCode(s)}</span>
              </div>
              <div className="flex items-baseline justify-between gap-2 font-table tabular-nums">
                <span className="text-[9.5px] text-muted-foreground truncate">
                  pickup {pickupDisplay(s)}
                </span>
                <span className="text-[10.5px] font-semibold">
                  {formatCurrency(revenue)}{" "}
                  <span className="text-[9.5px] font-normal text-muted-foreground">
                    · {miles}mi
                  </span>
                </span>
              </div>
            </button>
          );
        })}
      </div>
    </ModuleCard>
  );
}
