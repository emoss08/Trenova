import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import { cn } from "@trenova/shared/lib/utils";
import { useMemo } from "react";
import type { Region, ShipmentAnalyticsData } from "../../analytics/mock-data";

const HIGH_INTENSITY_THRESHOLD = 0.55;
const REGIONS: Region[] = ["West", "Midwest", "South", "Northeast"];

type LaneHeatmapProps = {
  data: ShipmentAnalyticsData["laneHeatmap"];
};

export function LaneHeatmap({ data }: LaneHeatmapProps) {
  const { grid, total, max, top } = useMemo(() => buildGrid(data), [data]);

  return (
    <section className="cc-module-card flex min-h-65 flex-col">
      <header className="flex items-center justify-between border-b border-border px-3 py-2">
        <div className="flex min-w-0 items-center gap-2">
          <h3 className="cc-label text-foreground">Lane heatmap</h3>
          <span className="font-mono text-[10px] text-muted-foreground">
            origin → destination · {total} loads
          </span>
        </div>
        <span className="font-mono text-[10px] text-muted-foreground">{data.windowDays}d</span>
      </header>

      <div className="flex flex-1 flex-col gap-1 px-3 py-3">
        <div className="grid grid-cols-[60px_repeat(4,minmax(0,1fr))] gap-1">
          <span aria-hidden />
          {REGIONS.map((r) => (
            <span
              key={r}
              className="text-center font-mono text-[9.5px] tracking-wider text-muted-foreground uppercase"
            >
              {r}
            </span>
          ))}
        </div>
        {REGIONS.map((origin, rowIdx) => (
          <div key={origin} className="grid grid-cols-[60px_repeat(4,minmax(0,1fr))] gap-1">
            <span className="flex items-center font-mono text-[9.5px] tracking-wider text-muted-foreground uppercase">
              {origin}
            </span>
            {REGIONS.map((destination, colIdx) => (
              <HeatCell
                key={destination}
                value={grid[rowIdx][colIdx]}
                max={max}
                origin={origin}
                destination={destination}
                total={total}
              />
            ))}
          </div>
        ))}
      </div>

      <footer className="flex items-center justify-between border-t border-border px-3 py-2 text-[10px] text-muted-foreground">
        <span className="font-mono">
          {top ? `Top: ${top.origin} → ${top.destination} (${top.count})` : "No lane activity"}
        </span>
        <ScaleLegend max={max} />
      </footer>
    </section>
  );
}

type CellProps = {
  value: number | null;
  max: number;
  origin: Region;
  destination: Region;
  total: number;
};

function HeatCell({ value, max, origin, destination, total }: CellProps) {
  if (value === null) {
    return (
      <div
        aria-hidden
        className="border-border-2 flex h-[26px] items-center justify-center rounded-[3px] border font-mono text-[11px] text-muted-foreground tabular-nums"
      >
        ·
      </div>
    );
  }

  const t = max > 0 ? value / max : 0;
  const opacity = Math.round(t * 100);
  const isHighIntensity = t > HIGH_INTENSITY_THRESHOLD;
  const percent = total > 0 ? ((value / total) * 100).toFixed(1) : "0.0";
  const tooltipContent = value
    ? `${origin} → ${destination}: ${value} loads · ${percent}% of total`
    : `${origin} → ${destination}: 0 loads`;

  return (
    <Tooltip>
      <TooltipTrigger
        render={
          <div
            className={cn(
              "border-border-2 flex h-6.5 items-center justify-center rounded-[3px] border font-mono text-[11px] tabular-nums",
              value === 0 && "text-muted-foreground",
              value > 0 && !isHighIntensity && "text-foreground",
              isHighIntensity && "text-white",
            )}
            style={{
              background:
                value > 0
                  ? `color-mix(in oklch, var(--color-brand) ${opacity}%, transparent)`
                  : "transparent",
            }}
          >
            {value || "·"}
          </div>
        }
      />
      <TooltipContent>{tooltipContent}</TooltipContent>
    </Tooltip>
  );
}

function ScaleLegend({ max }: { max: number }) {
  return (
    <span className="flex items-center gap-1.5">
      <span className="font-mono text-[9px]">0</span>
      <span
        aria-hidden
        className="block h-1.5 w-20 rounded-full"
        style={{
          background:
            "linear-gradient(to right, color-mix(in oklch, var(--color-brand) 5%, transparent), var(--color-brand))",
        }}
      />
      <span className="font-mono text-[9px]">{max}</span>
    </span>
  );
}

function buildGrid(data: ShipmentAnalyticsData["laneHeatmap"]): {
  grid: (number | null)[][];
  total: number;
  max: number;
  top: ShipmentAnalyticsData["laneHeatmap"]["cells"][number] | null;
} {
  const byOriginDest = new Map<string, number>();
  let max = 0;
  let top: ShipmentAnalyticsData["laneHeatmap"]["cells"][number] | null = null;
  for (const lane of data.cells) {
    byOriginDest.set(`${lane.origin}|${lane.destination}`, lane.count);
    if (lane.count > max) max = lane.count;
    if (lane.count > 0 && (!top || lane.count > top.count)) top = lane;
  }

  const grid = REGIONS.map((origin) =>
    REGIONS.map<number | null>(
      (destination) =>
        byOriginDest.get(`${origin}|${destination}`) ?? (origin === destination ? 0 : null),
    ),
  );

  return { grid, total: data.total, max, top };
}
