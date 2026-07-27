import type { DispatchScoreFactor } from "@/lib/graphql/dispatch-console";
import { cn } from "@trenova/shared/lib/utils";
import { scoreTone } from "./dispatch-vocabulary";

/**
 * Every recommendation shows its work. Dispatchers reject rankings they cannot audit, so
 * the factor breakdown is part of the answer rather than a debugging affordance.
 */
export function ScoreBreakdown({
  score,
  factors,
  className,
}: {
  score: number;
  factors: readonly DispatchScoreFactor[];
  className?: string;
}) {
  const maxContribution = factors.reduce((max, factor) => Math.max(max, factor.contribution), 0);

  return (
    <div className={cn("flex flex-col gap-2", className)}>
      <div className="flex items-baseline gap-1.5">
        <span className={cn("text-lg font-semibold tabular-nums", scoreTone(score))}>{score}</span>
        <span className="text-[11px] text-muted-foreground">of 100</span>
      </div>

      {factors.length === 0 ? (
        <p className="text-[11px] text-muted-foreground">
          Not enough data to score this pairing.
        </p>
      ) : (
        <ul className="flex flex-col gap-1.5">
          {factors.map((factor) => (
            <li key={factor.key} className="flex flex-col gap-0.5">
              <div className="flex items-baseline justify-between gap-2">
                <span className="text-[11px] font-medium">{factor.label}</span>
                <span className="text-[11px] tabular-nums text-muted-foreground">
                  +{factor.contribution.toFixed(1)}
                </span>
              </div>
              <div className="h-1 w-full overflow-hidden rounded-full bg-muted">
                <div
                  className="h-full rounded-full bg-brand"
                  style={{
                    width: `${
                      maxContribution > 0 ? (factor.contribution / maxContribution) * 100 : 0
                    }%`,
                  }}
                />
              </div>
              <span className="text-[10px] leading-tight text-muted-foreground">
                {factor.detail}
              </span>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
