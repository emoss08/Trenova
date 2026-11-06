import type { DispatchCandidate } from "@/lib/graphql/dispatch-console";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { costComponentLabel, costLineFact, endMarketReason, formatCents } from "./cost-vocabulary";

/**
 * What a driver taking a move costs, line by line, in money. The planner picks the plan
 * that costs least overall, so this is the same arithmetic it ran, not a summary of it.
 * Hard costs decide whether a move is worth covering; preferences only choose between
 * drivers, so they sit apart. Where the driver ends up is a credit, a negative line, when the
 * move leaves them somewhere worth more than where they are, and a cost when it pulls them
 * from somewhere busier; why it is worth that sits under it.
 */
export function CostBreakdown({
  candidate,
  coveragePriceCents,
  preferenceFloorCents = 0,
  className,
}: {
  candidate: DispatchCandidate;
  coveragePriceCents?: number;
  preferenceFloorCents?: number;
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
            {formatCents(candidate.costCents - preferenceFloorCents)}
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
              {preferenceFloorCents > 0 ? (
                <div className="flex items-baseline justify-between gap-2">
                  <span className="text-xs font-medium">
                    {t("Less what the best driver pays in preferences")}
                  </span>
                  <span className="text-success-foreground text-xs font-medium tabular-nums">
                    {formatCents(-preferenceFloorCents)}
                  </span>
                </div>
              ) : null}
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
  const t = useT();
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
              {line.component === "endMarket" && candidate.endMarket ? (
                <div className="text-muted-foreground mt-0.5 flex flex-col text-2xs">
                  {endMarketReason(candidate.endMarket, t).map((sentence) => (
                    <p key={sentence}>{sentence}</p>
                  ))}
                </div>
              ) : null}
            </li>
          );
        })}
    </ul>
  );
}
