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
          <span className="text-muted-foreground text-[10px] font-medium tracking-wider uppercase">
            Match score
          </span>
          <span className="flex items-baseline gap-1">
            <span
              className={cn("text-lg leading-none font-semibold tabular-nums", scoreTone(score))}
            >
              {score}
            </span>
            <span className="text-muted-foreground text-[10px]">/ 100</span>
          </span>
        </div>
        <div className="bg-muted h-1 w-full overflow-hidden rounded-full">
          <motion.div
            className="bg-brand h-full rounded-full"
            initial={reducedMotion ? false : { width: 0 }}
            animate={{ width: `${score}%` }}
            transition={{ duration: 0.5, ease: [0.22, 1, 0.36, 1] }}
          />
        </div>
      </div>

      {factors.length === 0 ? (
        <p className="text-muted-foreground text-[11px]">Not enough data to score this pairing.</p>
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
                className="text-muted-foreground hover:text-foreground flex items-center gap-1 self-start text-[10.5px] transition-colors"
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
                <ul className="border-border flex flex-col gap-1.5 border-l pl-3">
                  {flat.map((factor) => (
                    <li key={factor.key} className="flex flex-col gap-px">
                      <div className="flex items-baseline justify-between gap-2">
                        <span className="text-muted-foreground text-[11px]">{factor.label}</span>
                        <span className="text-muted-foreground/60 text-[10px] tabular-nums">
                          +0.0
                        </span>
                      </div>
                      <span className="text-muted-foreground/70 text-[10px] leading-snug">
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
        <span className="text-muted-foreground text-[11px] font-medium tabular-nums">
          +{factor.contribution.toFixed(1)}
        </span>
      </div>
      <div className="bg-muted h-1 w-full overflow-hidden rounded-full">
        <motion.div
          className="bg-brand/70 h-full rounded-full"
          initial={reducedMotion ? false : { width: 0 }}
          animate={{ width: `${share}%` }}
          transition={{ duration: 0.4, delay: 0.05 * index, ease: [0.22, 1, 0.36, 1] }}
        />
      </div>
      <span className="text-muted-foreground text-[10px] leading-snug">{factor.detail}</span>
    </li>
  );
}
