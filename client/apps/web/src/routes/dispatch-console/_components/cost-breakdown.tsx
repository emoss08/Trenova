import type { DispatchCandidate } from "@/lib/graphql/dispatch-console";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { costComponentLabel, costLineFact, formatCents } from "./cost-vocabulary";

/**
 * What a driver taking a move costs, line by line, in money. The planner picks the plan
 * that costs least overall, so this is the same arithmetic it ran, not a summary of it.
 * Hard costs decide whether a move is worth covering; preferences only choose between
 * drivers, so they sit apart. A credit (a move that leaves the driver where drivers are
 * worth more) reads as a negative line.
 */
export function CostBreakdown({
  candidate,
  coveragePriceCents,
  className,
}: {
  candidate: DispatchCandidate;
  coveragePriceCents?: number;
  className?: string;
}) {
  const t = useT();
  const hard = candidate.costLines.filter((line) => !line.soft);
  const soft = candidate.costLines.filter((line) => line.soft);
  const largest = candidate.costLines.reduce((max, line) => Math.max(max, Math.abs(line.cents)), 0);

  return (
    <div className={cn("flex flex-col gap-2.5", className)}>
      <div className="flex items-baseline justify-between gap-2">
        <span className="text-muted-foreground text-xs font-medium">{t("Cost to cover")}</span>
        <span className="flex items-baseline gap-1.5">
          <span className="text-lg leading-none font-semibold tabular-nums">
            {formatCents(candidate.costCents)}
          </span>
          {coveragePriceCents != null ? (
            <span className="text-muted-foreground text-2xs">
              {t("of {0} the move is worth", formatCents(coveragePriceCents))}
            </span>
          ) : null}
        </span>
      </div>

      {candidate.costLines.length === 0 ? (
        <p className="text-muted-foreground text-xs">{t("This pairing costs nothing extra.")}</p>
      ) : (
        <>
          <CostLines lines={hard} candidate={candidate} largest={largest} />
          {soft.length > 0 ? (
            <div className="flex flex-col gap-1.5">
              <span className="text-muted-foreground text-2xs font-medium">{t("Preferences")}</span>
              <CostLines lines={soft} candidate={candidate} largest={largest} />
            </div>
          ) : null}
        </>
      )}
    </div>
  );
}

function CostLines({
  lines,
  candidate,
  largest,
}: {
  lines: DispatchCandidate["costLines"];
  candidate: DispatchCandidate;
  largest: number;
}) {
  if (lines.length === 0) return null;

  return (
    <ul className="flex flex-col gap-2">
      {[...lines]
        .sort((a, b) => Math.abs(b.cents) - Math.abs(a.cents))
        .map((line) => {
          const fact = costLineFact(line.component, candidate);
          const share = largest > 0 ? (Math.abs(line.cents) / largest) * 100 : 0;
          const credit = line.cents < 0;
          return (
            <li key={line.component} className="flex flex-col gap-0.5">
              <div className="flex items-baseline justify-between gap-2">
                <span className="text-xs font-medium">
                  {costComponentLabel(line.component)}
                  {fact ? (
                    <span className="text-muted-foreground ml-1.5 font-normal">{fact}</span>
                  ) : null}
                </span>
                <span
                  className={cn(
                    "text-xs font-medium tabular-nums",
                    credit ? "text-success-foreground" : "text-muted-foreground",
                  )}
                >
                  {formatCents(line.cents)}
                </span>
              </div>
              <div className="bg-muted h-1 w-full overflow-hidden rounded-full">
                <div
                  className={cn("h-full rounded-full", credit ? "bg-success/70" : "bg-brand/70")}
                  style={{ width: `${share}%` }}
                />
              </div>
            </li>
          );
        })}
    </ul>
  );
}
