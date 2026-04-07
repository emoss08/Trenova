import { cn } from "@/lib/utils";
import type { AxleWeight } from "@/types/loading-optimization";
import type { RevenueContext } from "./use-loading-optimization";

function axleLabel(axle: string) {
  switch (axle) {
    case "drive":
      return "Drive Axle";
    case "trailer":
      return "Trailer Axle";
    default:
      return axle;
  }
}

export function AxleWeightDisplay({
  axleWeights,
  totalWeight,
  maxWeight,
  revenue,
}: {
  axleWeights: AxleWeight[];
  totalWeight: number;
  maxWeight: number;
  revenue?: RevenueContext | null;
}) {
  const isOverweight = totalWeight > maxWeight;
  const cargoAxles = axleWeights.filter((a) => a.axle !== "steer");

  return (
    <div className="rounded-lg border border-border p-3">
      <div className="mb-2.5 flex items-center justify-between">
        <span className="text-2xs font-medium uppercase tracking-wider text-muted-foreground">
          Axle Weights
        </span>
        <span className={cn("text-xs font-semibold tabular-nums", isOverweight && "text-destructive")}>
          {totalWeight.toLocaleString()} / {maxWeight.toLocaleString()} lbs
        </span>
      </div>
      <div className="grid grid-cols-2 gap-2">
        {cargoAxles.map((axle) => (
          <div
            key={axle.axle}
            className={cn(
              "rounded-md border px-2.5 py-2",
              axle.compliant ? "border-border" : "border-destructive/40 bg-destructive/5",
            )}
          >
            <div className="mb-1.5 flex items-center justify-between">
              <span className="text-2xs font-medium text-muted-foreground">{axleLabel(axle.axle)}</span>
              {!axle.compliant && (
                <span className="rounded-full bg-destructive/15 px-1.5 py-px text-[9px] font-semibold text-destructive">
                  OVER
                </span>
              )}
            </div>
            <div className="mb-1.5 h-1.5 w-full overflow-hidden rounded-full bg-muted">
              <div
                className={cn(
                  "h-full rounded-full transition-all",
                  axle.compliant ? "bg-emerald-500" : "bg-destructive",
                )}
                style={{ width: `${Math.min(axle.percentage, 100)}%` }}
              />
            </div>
            <div className="flex items-baseline justify-between">
              <span className={cn("text-xs font-semibold tabular-nums", !axle.compliant && "text-destructive")}>
                {axle.weight.toLocaleString()}
              </span>
              <span className="text-2xs text-muted-foreground">/ {axle.limit.toLocaleString()} lbs</span>
            </div>
          </div>
        ))}
      </div>

      {/* Revenue metrics inline */}
      {revenue && (
        <div className="mt-2.5 flex items-center gap-4 border-t border-border pt-2.5">
          <span className="text-2xs font-medium text-muted-foreground">Revenue</span>
          <span className="text-xs font-semibold tabular-nums text-foreground">
            ${revenue.revenuePerFoot.toFixed(0)}<span className="text-2xs font-normal text-muted-foreground">/ft</span>
          </span>
          {revenue.revenuePerMile > 0 && (
            <span className="text-xs font-semibold tabular-nums text-foreground">
              ${revenue.revenuePerMile.toFixed(2)}<span className="text-2xs font-normal text-muted-foreground">/mi</span>
            </span>
          )}
          {revenue.emptySpaceFeet > 0 && (
            <span className="ml-auto text-xs tabular-nums text-muted-foreground">
              {revenue.emptySpaceFeet.toFixed(0)}ft unused
            </span>
          )}
        </div>
      )}
    </div>
  );
}
