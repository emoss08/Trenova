import type { VehiclePosition } from "@/lib/graphql/telematics";
import { queries } from "@/lib/queries";
import { formatTimeAgo } from "@/lib/time-utils";
import { Separator } from "@trenova/shared/components/ui/separator";
import { cn } from "@trenova/shared/lib/utils";
import { useQuery } from "@tanstack/react-query";
import { AdvancedMarker } from "@vis.gl/react-google-maps";
import { MapPinIcon, TruckIcon, UserIcon, XIcon } from "lucide-react";
import { useEffect, useState } from "react";

const POSITION_MAX_AGE_SECONDS = 3600;
const STALE_THRESHOLD_MS = 10 * 60 * 1000;
const MOVING_SPEED_MPH = 3;
const CLOCK_TICK_MS = 30_000;

function positionColor(position: VehiclePosition): string {
  const engineOff = position.engineState?.toLowerCase() === "off";
  return engineOff ? "var(--muted-foreground)" : "var(--brand)";
}

function isStale(position: VehiclePosition, now: number): boolean {
  return now - position.recordedAt * 1000 > STALE_THRESHOLD_MS;
}

function isMoving(position: VehiclePosition): boolean {
  return position.speedMph > MOVING_SPEED_MPH;
}

function engineStateLabel(engineState: string | null): string {
  if (!engineState) return "—";
  switch (engineState.toLowerCase()) {
    case "on":
      return "On";
    case "off":
      return "Off";
    case "idle":
      return "Idle";
    default:
      return engineState;
  }
}

export function VehiclePositionOverlay({ enabled = true }: { enabled?: boolean }) {
  const statusQuery = useQuery({
    ...queries.telematics.status(),
    staleTime: 5 * 60 * 1000,
    retry: false,
    refetchOnWindowFocus: false,
    enabled,
  });
  const telematicsEnabled = statusQuery.data?.enabled ?? false;
  const fetchEnabled = enabled && telematicsEnabled;

  const positionsQuery = useQuery({
    ...queries.telematics.vehiclePositions(POSITION_MAX_AGE_SECONDS),
    refetchInterval: 30_000,
    staleTime: 15_000,
    retry: false,
    refetchOnWindowFocus: false,
    enabled: fetchEnabled,
  });

  const [hoveredId, setHoveredId] = useState<string | null>(null);
  const [pinnedId, setPinnedId] = useState<string | null>(null);
  const [now, setNow] = useState(() => Date.now());

  useEffect(() => {
    if (!fetchEnabled) return;
    const interval = window.setInterval(() => setNow(Date.now()), CLOCK_TICK_MS);
    return () => window.clearInterval(interval);
  }, [fetchEnabled]);

  if (!fetchEnabled) return null;

  const positions = positionsQuery.data ?? [];
  if (positions.length === 0) return null;

  const openId = pinnedId ?? hoveredId;
  const openPosition = openId
    ? (positions.find((p) => p.tractorId === openId) ?? null)
    : null;

  return (
    <>
      {positions.map((position) => (
        <VehicleMarker
          key={position.tractorId}
          position={position}
          stale={isStale(position, now)}
          active={openId === position.tractorId}
          onMouseEnter={() => setHoveredId(position.tractorId)}
          onMouseLeave={() =>
            setHoveredId((current) => (current === position.tractorId ? null : current))
          }
          onClick={() =>
            setPinnedId((current) => (current === position.tractorId ? null : position.tractorId))
          }
        />
      ))}
      {openPosition && (
        <VehicleDetailCard
          position={openPosition}
          stale={isStale(openPosition, now)}
          pinned={pinnedId === openPosition.tractorId}
          now={now}
          onClose={() => {
            setPinnedId(null);
            setHoveredId(null);
          }}
        />
      )}
    </>
  );
}

function VehicleMarker({
  position,
  stale,
  active,
  onMouseEnter,
  onMouseLeave,
  onClick,
}: {
  position: VehiclePosition;
  stale: boolean;
  active: boolean;
  onMouseEnter: () => void;
  onMouseLeave: () => void;
  onClick: () => void;
}) {
  const color = positionColor(position);
  const moving = isMoving(position);
  const title = `${position.tractorCode}${position.primaryWorkerName ? ` · ${position.primaryWorkerName}` : ""}`;

  return (
    <AdvancedMarker
      position={{ lat: position.latitude, lng: position.longitude }}
      zIndex={active ? 90 : 40}
      title={title}
      onClick={onClick}
    >
      <div
        className={cn(
          "relative flex items-center justify-center transition-opacity",
          moving && !stale && "cc-pulse-pin",
        )}
        onMouseEnter={onMouseEnter}
        onMouseLeave={onMouseLeave}
        style={{
          width: 22,
          height: 22,
          color,
          opacity: stale ? 0.45 : 1,
        }}
      >
        <span
          aria-hidden
          className="flex size-full items-center justify-center rounded-full"
          style={{
            background: color,
            border: active ? "2px solid var(--card)" : "1.5px solid var(--card)",
            boxShadow: "0 2px 4px rgba(0,0,0,0.3)",
          }}
        >
          <TruckIcon className="size-3" style={{ color: "var(--brand-foreground)" }} />
        </span>
        <span
          aria-hidden
          className="absolute inset-0 transition-transform duration-300"
          style={{ transform: `rotate(${position.headingDegrees}deg)` }}
        >
          <span
            className="absolute -top-[7px] left-1/2 -translate-x-1/2"
            style={{
              width: 0,
              height: 0,
              borderLeft: "3.5px solid transparent",
              borderRight: "3.5px solid transparent",
              borderBottom: `5px solid ${color}`,
            }}
          />
        </span>
      </div>
    </AdvancedMarker>
  );
}

function VehicleDetailCard({
  position,
  stale,
  pinned,
  now,
  onClose,
}: {
  position: VehiclePosition;
  stale: boolean;
  pinned: boolean;
  now: number;
  onClose: () => void;
}) {
  return (
    <AdvancedMarker
      position={{ lat: position.latitude, lng: position.longitude }}
      zIndex={150}
      onClick={(e) => e.stopPropagation()}
    >
      <div className={cn("relative mb-8", !pinned && "pointer-events-none")}>
        <div className="cc-fade-in relative z-10 flex w-64 flex-col gap-2.5 rounded-lg border bg-popover p-3 text-xs text-popover-foreground shadow-lg ring-1 ring-foreground/10">
          <div className="flex items-start justify-between gap-2">
            <div className="flex min-w-0 flex-col gap-0.5">
              <span className="flex items-center gap-1.5 truncate text-sm font-semibold text-foreground">
                <TruckIcon className="size-3.5 shrink-0 text-muted-foreground" />
                {position.tractorCode}
                {stale && (
                  <span className="rounded bg-warning/15 px-1 py-px text-[8.5px] font-bold tracking-wide text-warning uppercase">
                    Stale
                  </span>
                )}
              </span>
              {position.primaryWorkerName && (
                <span className="flex items-center gap-1 text-2xs text-muted-foreground">
                  <UserIcon className="size-3" />
                  {position.primaryWorkerName}
                </span>
              )}
            </div>
            {pinned && (
              <button
                type="button"
                onClick={onClose}
                className="rounded text-muted-foreground hover:text-foreground"
                aria-label="Close vehicle info"
              >
                <XIcon className="size-3.5" />
              </button>
            )}
          </div>

          <Separator />
          <div
            className={cn(
              "grid gap-2 text-2xs tabular-nums",
              position.fuelPercent != null ? "grid-cols-3" : "grid-cols-2",
            )}
          >
            <div className="flex flex-col">
              <span className="text-muted-foreground">Speed</span>
              <span className="text-foreground">{Math.round(position.speedMph)} mph</span>
            </div>
            <div className="flex flex-col">
              <span className="text-muted-foreground">Engine</span>
              <span className="text-foreground">{engineStateLabel(position.engineState)}</span>
            </div>
            {position.fuelPercent != null && (
              <div className="flex flex-col">
                <span className="text-muted-foreground">Fuel</span>
                <span className="text-foreground">{Math.round(position.fuelPercent)}%</span>
              </div>
            )}
          </div>

          {position.formattedLocation && (
            <>
              <Separator />
              <div className="flex items-start gap-2">
                <MapPinIcon className="mt-0.5 size-3.5 shrink-0 text-muted-foreground" />
                <span className="min-w-0 text-2xs leading-relaxed text-foreground">
                  {position.formattedLocation}
                </span>
              </div>
            </>
          )}

          <Separator />
          <span className="font-table text-2xs text-muted-foreground tabular-nums">
            Updated {formatTimeAgo(position.recordedAt * 1000, now)}
            {stale && " · position may be out of date"}
          </span>
        </div>
      </div>
    </AdvancedMarker>
  );
}
