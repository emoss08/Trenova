import { cn } from "@/lib/utils";
import type { COMMODITY_PALETTE } from "./constants";

type PaletteEntry = (typeof COMMODITY_PALETTE)[number];

interface CommoditySegment {
  name: string;
  weight: number;
  lengthFeet: number;
  instructions?: string;
  palette: PaletteEntry;
}

export function LinearFeetBar({
  totalLinearFeet,
  trailerLengthFeet,
  utilization,
  commodities,
}: {
  totalLinearFeet: number;
  trailerLengthFeet: number;
  utilization: number;
  commodities?: CommoditySegment[];
}) {
  const isOver = totalLinearFeet > trailerLengthFeet;

  return (
    <div className="rounded-lg border border-border p-3">
      <div className="mb-2 flex items-center justify-between">
        <span className="text-2xs font-medium uppercase tracking-wider text-muted-foreground">
          Linear Feet
        </span>
        <span className={cn("text-xs font-semibold tabular-nums", isOver && "text-destructive")}>
          {totalLinearFeet.toFixed(1)} / {trailerLengthFeet} ft
        </span>
      </div>

      {/* Stacked segmented bar */}
      <div className="relative h-4 w-full overflow-hidden rounded-full bg-muted">
        {commodities && commodities.length > 0 ? (
          <div className="flex h-full">
            {commodities.map((c) => {
              const pct = trailerLengthFeet > 0 ? (c.lengthFeet / trailerLengthFeet) * 100 : 0;
              if (pct <= 0) return null;
              return (
                <div
                  key={c.name}
                  className={cn("h-full first:rounded-l-full last:rounded-r-full", c.palette.barBg)}
                  style={{ width: `${Math.min(pct, 100)}%` }}
                  title={`${c.name}: ${c.lengthFeet.toFixed(1)}ft`}
                />
              );
            })}
          </div>
        ) : (
          <div
            className={cn(
              "h-full rounded-full transition-all duration-500",
              isOver ? "bg-red-500" : utilization > 95 ? "bg-amber-400" : "bg-emerald-500",
            )}
            style={{ width: `${Math.min(utilization, 100)}%` }}
          />
        )}
        <span className="absolute inset-0 flex items-center justify-center text-[9px] font-bold text-white drop-shadow-[0_0_3px_rgba(0,0,0,0.5)]">
          {utilization.toFixed(0)}%
        </span>
      </div>

      {/* Commodity legend */}
      {commodities && commodities.length > 0 && (
        <div className="mt-2 flex flex-wrap items-center gap-x-3 gap-y-1">
          {commodities.slice(0, 4).map((c) => (
            <div key={c.name} className="flex items-center gap-1 text-2xs">
              <div className={cn("size-2 rounded-sm border", c.palette.dotBg, c.palette.dotBorder)} />
              <span className="text-foreground">{c.name}</span>
              <span className="text-muted-foreground">{c.lengthFeet.toFixed(1)}ft</span>
            </div>
          ))}
          {commodities.length > 4 && (
            <span className="text-2xs text-muted-foreground">
              +{commodities.length - 4} more
            </span>
          )}
        </div>
      )}
    </div>
  );
}
