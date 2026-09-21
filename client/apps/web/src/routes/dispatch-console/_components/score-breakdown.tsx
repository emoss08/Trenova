import { useT } from "@trenova/shared/i18n/use-t";
import type { DispatchScoreFactor } from "@/lib/graphql/dispatch-console";
import { cn } from "@trenova/shared/lib/utils";
import { ChevronDownIcon } from "lucide-react";
import { useState } from "react";
import { scoreTone } from "./dispatch-vocabulary";

/**
 * Every recommendation shows its work. Dispatchers reject rankings they cannot audit, so
 * the factor breakdown is part of the answer rather than a debugging affordance. Factors
 * that earned nothing stay out of the way until asked for — a zero is still an answer,
 * just not the headline.
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
  const t = useT();

  const [showFlat, setShowFlat] = useState(false);

  const maxContribution = factors.reduce((max, factor) => Math.max(max, factor.contribution), 0);
  const signal = factors.filter((factor) => factor.contribution > 0);
  const flat = factors.filter((factor) => factor.contribution <= 0);

  return (
    <div className={cn("flex flex-col gap-2.5", className)}>
      <div className="flex flex-col gap-1.5">
        <div className="flex items-baseline justify-between">
          <span className="text-muted-foreground text-xs font-medium">
            {t("Match score")}
          </span>
          <span className="flex items-baseline gap-1">
            <span
              className={cn("text-lg leading-none font-semibold tabular-nums", scoreTone(score))}
            >
              {score}
            </span>
            <span className="text-muted-foreground text-2xs">/ 100</span>
          </span>
        </div>
        <div className="bg-muted h-1 w-full overflow-hidden rounded-full">
          <div className="bg-brand h-full rounded-full" style={{ width: `${score}%` }} />
        </div>
      </div>

      {factors.length === 0 ? (
        <p className="text-muted-foreground text-xs">
          {t("Not enough data to score this pairing.")}
        </p>
      ) : (
        <>
          <ul className="flex flex-col gap-2">
            {signal.map((factor) => (
              <FactorRow
                key={factor.key}
                factor={factor}
                maxContribution={maxContribution}
              />
            ))}
          </ul>

          {flat.length > 0 ? (
            <div className="flex flex-col gap-2">
              <button
                type="button"
                className="text-muted-foreground hover:text-foreground flex items-center gap-1 self-start text-2xs transition-colors"
                onClick={() => setShowFlat((previous) => !previous)}
                aria-expanded={showFlat}
              >
                <ChevronDownIcon
                  className={cn("size-3 transition-transform", showFlat && "rotate-180")}
                  aria-hidden
                />
                {t(
                  "{0, plural, one {# factor} other {# factors}} contributed nothing",
                  flat.length,
                )}
              </button>
              {showFlat ? (
                <ul className="border-border flex flex-col gap-1.5 border-l pl-3">
                  {flat.map((factor) => (
                    <li key={factor.key} className="flex flex-col gap-px">
                      <div className="flex items-baseline justify-between gap-2">
                        <span className="text-muted-foreground text-xs">{t(factor.label)}</span>
                        <span className="text-muted-foreground/60 text-2xs tabular-nums">
                          +0.0
                        </span>
                      </div>
                      <span className="text-muted-foreground/70 text-2xs leading-snug">
                        {factor.detail}
                      </span>
                    </li>
                  ))}
                </ul>
              ) : null}
            </div>
          ) : null}
        </>
      )}
    </div>
  );
}

function FactorRow({
  factor,
  maxContribution,
}: {
  factor: DispatchScoreFactor;
  maxContribution: number;
}) {
  const t = useT();

  const share = maxContribution > 0 ? (factor.contribution / maxContribution) * 100 : 0;

  return (
    <li className="flex flex-col gap-0.5">
      <div className="flex items-baseline justify-between gap-2">
        <span className="text-xs font-medium">{t(factor.label)}</span>
        <span className="text-muted-foreground text-xs font-medium tabular-nums">
          +{factor.contribution.toFixed(1)}
        </span>
      </div>
      <div className="bg-muted h-1 w-full overflow-hidden rounded-full">
        <div className="bg-brand/70 h-full rounded-full" style={{ width: `${share}%` }} />
      </div>
      <span className="text-muted-foreground text-2xs leading-snug">{factor.detail}</span>
    </li>
  );
}
