import type { DispatchScoreFactor } from "@/lib/graphql/dispatch-console";
import { cn } from "@trenova/shared/lib/utils";
import { ChevronDownIcon } from "lucide-react";
import { motion, useReducedMotion } from "motion/react";
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
  const [showFlat, setShowFlat] = useState(false);
  const reducedMotion = useReducedMotion();

  const maxContribution = factors.reduce((max, factor) => Math.max(max, factor.contribution), 0);
  const signal = factors.filter((factor) => factor.contribution > 0);
  const flat = factors.filter((factor) => factor.contribution <= 0);

  return (
    <div className={cn("flex flex-col gap-2.5", className)}>
      <div className="flex flex-col gap-1.5">
        <div className="flex items-baseline justify-between">
          <span className="text-[10px] font-medium tracking-wider text-muted-foreground uppercase">
            Match score
          </span>
          <span className="flex items-baseline gap-1">
            <span
              className={cn("text-lg leading-none font-semibold tabular-nums", scoreTone(score))}
            >
              {score}
            </span>
            <span className="text-[10px] text-muted-foreground">/ 100</span>
          </span>
        </div>
        <div className="h-1 w-full overflow-hidden rounded-full bg-muted">
          <motion.div
            className="h-full rounded-full bg-brand"
            initial={reducedMotion ? false : { width: 0 }}
            animate={{ width: `${score}%` }}
            transition={{ duration: 0.5, ease: [0.22, 1, 0.36, 1] }}
          />
        </div>
      </div>

      {factors.length === 0 ? (
        <p className="text-[11px] text-muted-foreground">Not enough data to score this pairing.</p>
      ) : (
        <>
          <ul className="flex flex-col gap-2">
            {signal.map((factor, index) => (
              <FactorRow
                key={factor.key}
                factor={factor}
                maxContribution={maxContribution}
                index={index}
                reducedMotion={Boolean(reducedMotion)}
              />
            ))}
          </ul>

          {flat.length > 0 ? (
            <div className="flex flex-col gap-2">
              <button
                type="button"
                className="flex items-center gap-1 self-start text-[10.5px] text-muted-foreground transition-colors hover:text-foreground"
                onClick={() => setShowFlat((previous) => !previous)}
                aria-expanded={showFlat}
              >
                <ChevronDownIcon
                  className={cn("size-3 transition-transform", showFlat && "rotate-180")}
                  aria-hidden
                />
                {flat.length} factor{flat.length === 1 ? "" : "s"} contributed nothing
              </button>
              {showFlat ? (
                <ul className="flex flex-col gap-1.5 border-l border-border pl-3">
                  {flat.map((factor) => (
                    <li key={factor.key} className="flex flex-col gap-px">
                      <div className="flex items-baseline justify-between gap-2">
                        <span className="text-[11px] text-muted-foreground">{factor.label}</span>
                        <span className="text-[10px] tabular-nums text-muted-foreground/60">
                          +0.0
                        </span>
                      </div>
                      <span className="text-[10px] leading-snug text-muted-foreground/70">
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
  index,
  reducedMotion,
}: {
  factor: DispatchScoreFactor;
  maxContribution: number;
  index: number;
  reducedMotion: boolean;
}) {
  const share = maxContribution > 0 ? (factor.contribution / maxContribution) * 100 : 0;

  return (
    <li className="flex flex-col gap-0.5">
      <div className="flex items-baseline justify-between gap-2">
        <span className="text-[11px] font-medium">{factor.label}</span>
        <span className="text-[11px] font-medium tabular-nums text-muted-foreground">
          +{factor.contribution.toFixed(1)}
        </span>
      </div>
      <div className="h-1 w-full overflow-hidden rounded-full bg-muted">
        <motion.div
          className="h-full rounded-full bg-brand/70"
          initial={reducedMotion ? false : { width: 0 }}
          animate={{ width: `${share}%` }}
          transition={{ duration: 0.4, delay: 0.05 * index, ease: [0.22, 1, 0.36, 1] }}
        />
      </div>
      <span className="text-[10px] leading-snug text-muted-foreground">{factor.detail}</span>
    </li>
  );
}
