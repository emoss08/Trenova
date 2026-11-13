import { formatElapsedTime } from "@/lib/time-utils";
import { STAGE_META } from "@/lib/shipment-board/stage";
import { useRealtimeStore } from "@/stores/realtime-store";
import type { RouteMode } from "@/types/shipment-map";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { SegmentedControl } from "@trenova/shared/components/ui/segmented-control";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { useEffect, useState } from "react";
import { MAP_STAGES, type MapStage } from "./map-shipment-model";

/**
 * The map's one header row. The stage chips are both the legend and the
 * filter; the route mode, the live state and the map controls follow.
 *
 * `compact` is the row a tile has room for: the chips and the controls and
 * nothing else, on one line. The chips scroll sideways when they do not fit,
 * because a header that wraps takes a second row out of a map that has few to
 * spare.
 */
export function ShipmentMapHeader({
  counts,
  hiddenStages,
  onToggleStage,
  routeMode,
  onRouteModeChange,
  dataUpdatedAt,
  controls,
  compact = false,
}: {
  counts: Record<MapStage, number>;
  hiddenStages: readonly MapStage[];
  onToggleStage: (stage: MapStage) => void;
  routeMode: RouteMode;
  onRouteModeChange: (mode: RouteMode) => void;
  dataUpdatedAt: number;
  controls: React.ReactNode;
  compact?: boolean;
}) {
  const t = useT();
  const stages = MAP_STAGES.filter((stage) => counts[stage] > 0 || hiddenStages.includes(stage));

  const chips = stages.map((stage) => {
    const on = !hiddenStages.includes(stage);
    return (
      <button
        key={stage}
        type="button"
        aria-pressed={on}
        onClick={() => onToggleStage(stage)}
        className={cn(
          "ui-focus-ring inline-flex h-6 shrink-0 items-center gap-1.5 rounded-md border px-2 text-xs whitespace-nowrap",
          on
            ? "bg-card text-foreground hover:bg-surface-hover"
            : "border-dashed text-muted-foreground",
        )}
      >
        <span
          aria-hidden
          className={cn(
            "size-2 rounded-full",
            STAGE_META[stage].swatchClassName,
            !on && "opacity-40",
          )}
        />
        {t(STAGE_META[stage].label)}
        <span className="font-mono text-muted-foreground tabular-nums">{counts[stage]}</span>
      </button>
    );
  });

  if (compact) {
    return (
      <div className="flex min-h-10 shrink-0 items-center gap-2 border-b border-subtle bg-card px-2.5 py-1.5">
        <ScrollArea dragToScroll maskHeight={20} maskVariant="card" className="min-w-0 flex-1">
          <div role="group" aria-label={t("Show stages")} className="flex w-max items-center gap-1">
            {chips}
          </div>
        </ScrollArea>
        <div className="flex shrink-0 items-center gap-2">{controls}</div>
      </div>
    );
  }

  return (
    <div className="flex min-h-10 shrink-0 flex-wrap items-center gap-x-3 gap-y-1.5 border-b border-subtle bg-card px-2.5 py-1.5">
      <span className="text-xs font-semibold text-foreground">{t("Live map")}</span>
      <div role="group" aria-label={t("Show stages")} className="flex flex-wrap items-center gap-1">
        {chips}
      </div>
      <div className="ml-auto flex items-center gap-2">
        <SegmentedControl<RouteMode>
          aria-label={t("Routes shown")}
          value={routeMode}
          onValueChange={onRouteModeChange}
          items={[
            { value: "selected", label: t("Selected route") },
            { value: "lanes", label: t("Lanes") },
            { value: "off", label: t("No routes") },
          ]}
        />
        <LiveSyncIndicator dataUpdatedAt={dataUpdatedAt} />
        {controls}
      </div>
    </div>
  );
}

function LiveSyncIndicator({ dataUpdatedAt }: { dataUpdatedAt: number }) {
  const t = useT();
  const connectionState = useRealtimeStore((s) => s.connectionState);
  const lastEventAt = useRealtimeStore((s) => s.lastEventAt);
  const [now, setNow] = useState(() => Date.now());

  useEffect(() => {
    const interval = window.setInterval(() => setNow(Date.now()), 1000);
    return () => window.clearInterval(interval);
  }, []);

  const syncedAt = Math.max(lastEventAt ?? 0, dataUpdatedAt ?? 0);
  const live = connectionState === "connected";
  const elapsed = formatElapsedTime(syncedAt, now);

  return (
    <span className="text-muted-foreground hidden items-center gap-1.5 font-mono text-xs whitespace-nowrap tabular-nums @2xl/map:inline-flex">
      <span
        aria-hidden
        className={cn("size-1.5 rounded-full", live ? "bg-success" : "bg-muted-foreground")}
      />
      {live ? t("Live · synced {0}", elapsed) : t("Offline · synced {0}", elapsed)}
    </span>
  );
}
