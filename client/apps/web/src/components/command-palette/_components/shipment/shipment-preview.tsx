import { ShipmentStatusBadge } from "@/components/status-badge";
import { Badge } from "@/components/ui/badge";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Skeleton } from "@/components/ui/skeleton";
import { formatSplitDateTime } from "@/lib/date";
import { queries } from "@/lib/queries";
import { getDestinationStop, getOriginStop } from "@/lib/shipment-utils";
import { formatCurrency } from "@/lib/utils";
import { useAuthStore } from "@/stores/auth-store";
import type { MoveStatus, Shipment, ShipmentMove, Stop } from "@/types/shipment";
import { useQuery } from "@tanstack/react-query";
import { ArrowRightIcon } from "lucide-react";
import { ShipmentRouteMap } from "./shipment-preview-map";

const moveStatusConfig: Record<
  MoveStatus,
  { label: string; variant: "secondary" | "info" | "orange" | "active" | "inactive" }
> = {
  New: { label: "New", variant: "secondary" },
  Assigned: { label: "Assigned", variant: "info" },
  InTransit: { label: "In Transit", variant: "orange" },
  Completed: { label: "Completed", variant: "active" },
  Canceled: { label: "Canceled", variant: "inactive" },
};

const stopDotColor: Record<MoveStatus, string> = {
  New: "bg-purple-500",
  Assigned: "bg-blue-500",
  InTransit: "bg-blue-500",
  Completed: "bg-green-500",
  Canceled: "bg-red-500",
};

export function ShipmentSearchPreview({ shipmentId }: { shipmentId?: string }) {
  const enabled = Boolean(shipmentId);

  const { data, isLoading, isError } = useQuery({
    ...queries.shipment.get(shipmentId, { expandShipmentDetails: "true" }),
    enabled,
    staleTime: 30_000,
  });

  if (!shipmentId) {
    return (
      <div className="flex h-full w-full items-center justify-center text-2xs text-muted-foreground">
        Hover a shipment to preview details
      </div>
    );
  }

  if (isLoading) {
    return <PreviewSkeleton />;
  }

  if (isError || !data) {
    return (
      <div className="flex h-full items-center justify-center text-2xs text-muted-foreground">
        Unable to load shipment.
      </div>
    );
  }

  return <ShipmentPreviewContent shipment={data} />;
}

function formatCityState(stop: Stop | null | undefined): string | null {
  if (!stop?.location) return null;
  const city = stop.location.city;
  const stateAbbr = stop.location.state?.abbreviation;
  if (city && stateAbbr) return `${city}, ${stateAbbr}`;
  return city || stateAbbr || null;
}

function ShipmentPreviewContent({ shipment }: { shipment: Shipment }) {
  const origin = getOriginStop(shipment);
  const destination = getDestinationStop(shipment);
  const originLabel = formatCityState(origin);
  const destLabel = formatCityState(destination);

  const totalMileage = shipment.moves.reduce((sum, m) => sum + (m.distance ?? 0), 0);

  const details: { label: string; value: string }[] = [];
  if (shipment.pieces != null) {
    details.push({ label: "Pieces", value: shipment.pieces.toLocaleString() });
  }
  if (shipment.weight != null) {
    details.push({ label: "Weight", value: `${shipment.weight.toLocaleString()} lbs` });
  }
  if (totalMileage > 0) {
    details.push({ label: "Mileage", value: `${totalMileage.toLocaleString()} mi` });
  }
  if (shipment.totalChargeAmount != null && Number(shipment.totalChargeAmount) > 0) {
    details.push({ label: "Total", value: formatCurrency(Number(shipment.totalChargeAmount)) });
  }

  return (
    <ScrollArea className="h-full">
      <div className="flex flex-col gap-3 px-3 py-2 text-sm">
        <div className="flex flex-col gap-0.5">
          <div className="flex items-center justify-between gap-2">
            <span className="text-base font-semibold">{shipment.proNumber || shipment.id}</span>
            <ShipmentStatusBadge status={shipment.status} />
          </div>
          <p className="max-w-full truncate text-xs text-muted-foreground">
            {shipment.customer?.name}
            {shipment.customer?.code && ` (${shipment.customer.code})`}
            {shipment.bol && ` · BOL: ${shipment.bol}`}
          </p>
        </div>
        <ShipmentRouteMap moves={shipment.moves} />
        {(originLabel || destLabel) && (
          <div className="flex flex-col gap-0.5 border-t pt-2">
            <span className="text-2xs font-medium text-muted-foreground">Route</span>
            <div className="flex items-center gap-1.5 text-xs font-medium">
              <span>{originLabel ?? "—"}</span>
              <ArrowRightIcon className="size-3 shrink-0 text-muted-foreground" />
              <span>{destLabel ?? "—"}</span>
            </div>
          </div>
        )}
        {details.length > 0 && (
          <div className="flex flex-col gap-1.5 border-t pt-2">
            <span className="text-2xs font-medium text-muted-foreground">Details</span>
            <div className="grid grid-cols-2 gap-x-4 gap-y-1">
              {details.map((d) => (
                <div key={d.label} className="flex flex-col">
                  <span className="text-2xs text-muted-foreground">{d.label}</span>
                  <span className="text-xs font-medium">{d.value}</span>
                </div>
              ))}
            </div>
          </div>
        )}
        {shipment.moves.length > 0 && (
          <div className="flex flex-col gap-2 border-t pt-2">
            {shipment.moves
              .slice()
              .sort((a, b) => a.sequence - b.sequence)
              .map((move) => (
                <MoveCard key={move.id} move={move} />
              ))}
          </div>
        )}
      </div>
    </ScrollArea>
  );
}

function MoveCard({ move }: { move: ShipmentMove }) {
  const user = useAuthStore((s) => s.user);
  const config = moveStatusConfig[move.status];
  const sortedStops = [...move.stops].sort((a, b) => a.sequence - b.sequence);

  return (
    <div className="rounded-lg border bg-card p-2.5">
      <div className="mb-2 flex items-center justify-between">
        <div className="flex items-center gap-1.5">
          <span className="text-xs font-semibold">Move {move.sequence + 1}</span>
          {move.distance != null && move.distance > 0 && (
            <span className="text-2xs text-muted-foreground">
              · {move.distance.toLocaleString()} mi
            </span>
          )}
        </div>
        <Badge variant={config.variant} className="text-2xs">
          {config.label}
        </Badge>
      </div>
      <div className="relative flex flex-col">
        {sortedStops.map((stop, idx) => {
          const dotColor = stopDotColor[move.status];
          const isLast = idx === sortedStops.length - 1;
          const windowStart = stop.scheduledWindowStart
            ? formatSplitDateTime(stop.scheduledWindowStart, user?.timeFormat, user?.timezone)
            : null;
          const windowEnd =
            stop.scheduledWindowEnd != null && stop.scheduledWindowEnd > 0
              ? formatSplitDateTime(stop.scheduledWindowEnd, user?.timeFormat, user?.timezone)
              : null;

          return (
            <div key={stop.id} className="relative flex gap-2.5 pb-4 last:pb-0">
              {!isLast && <div className="absolute top-3 -bottom-2 left-[4px] w-px bg-border" />}
              <div className="relative z-1 flex flex-col items-center">
                <div className={`mt-0.5 size-2.5 shrink-0 rounded-full ${dotColor}`} />
              </div>
              <div className="flex min-w-0 flex-col gap-0.5">
                <div className="flex items-center gap-1.5">
                  <span className="text-xs font-medium">{stop.type}</span>
                  {stop.location?.name && (
                    <span className="truncate text-2xs text-muted-foreground">
                      – {stop.location.name}
                    </span>
                  )}
                </div>
                {windowStart && (
                  <span className="text-2xs text-muted-foreground">
                    {windowStart.date} · {windowStart.time}
                    {windowEnd &&
                      ` - ${windowEnd.date === windowStart.date ? windowEnd.time : `${windowEnd.date} · ${windowEnd.time}`}`}
                  </span>
                )}
              </div>
            </div>
          );
        })}
      </div>
      {move.assignment && (move.assignment.tractor || move.assignment.primaryWorker) && (
        <div className="mt-2 grid grid-cols-2 gap-2 border-t pt-2">
          {move.assignment.tractor && (
            <div className="flex flex-col">
              <span className="text-2xs text-muted-foreground">Tractor</span>
              <span className="text-xs font-medium">{move.assignment.tractor.code}</span>
            </div>
          )}
          {move.assignment.primaryWorker && (
            <div className="flex flex-col">
              <span className="text-2xs text-muted-foreground">Worker</span>
              <span className="truncate text-xs font-medium">
                {move.assignment.primaryWorker.firstName}{" "}
                {move.assignment.primaryWorker.lastName?.charAt(0)}.
              </span>
            </div>
          )}
        </div>
      )}
    </div>
  );
}

function PreviewSkeleton() {
  return (
    <div className="flex flex-col gap-3 px-3 py-2">
      <div className="flex flex-col gap-1">
        <div className="flex items-center justify-between">
          <Skeleton className="h-5 w-32" />
          <Skeleton className="h-5 w-16 rounded-full" />
        </div>
        <Skeleton className="h-3.5 w-48" />
      </div>
      <Skeleton className="h-32 w-full rounded-md" />
      <div className="flex flex-col gap-1 border-t pt-2">
        <Skeleton className="h-3 w-12" />
        <Skeleton className="h-4 w-40" />
      </div>
      <div className="flex flex-col gap-1.5 border-t pt-2">
        <Skeleton className="h-3 w-12" />
        <div className="grid grid-cols-2 gap-x-4 gap-y-2">
          <div className="flex flex-col gap-1">
            <Skeleton className="h-3 w-10" />
            <Skeleton className="h-4 w-14" />
          </div>
          <div className="flex flex-col gap-1">
            <Skeleton className="h-3 w-10" />
            <Skeleton className="h-4 w-14" />
          </div>
        </div>
      </div>
      <div className="flex flex-col gap-2 border-t pt-2">
        <div className="rounded-lg border bg-card p-2.5">
          <div className="mb-2 flex items-center justify-between">
            <Skeleton className="h-4 w-16" />
            <Skeleton className="h-4 w-14 rounded-full" />
          </div>
          <div className="flex flex-col gap-3">
            <div className="flex gap-2.5">
              <Skeleton className="mt-0.5 size-2.5 shrink-0 rounded-full" />
              <div className="flex flex-col gap-1">
                <Skeleton className="h-3.5 w-36" />
                <Skeleton className="h-3 w-28" />
              </div>
            </div>
            <div className="flex gap-2.5">
              <Skeleton className="mt-0.5 size-2.5 shrink-0 rounded-full" />
              <div className="flex flex-col gap-1">
                <Skeleton className="h-3.5 w-36" />
                <Skeleton className="h-3 w-28" />
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
