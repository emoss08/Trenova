import type { TotalCompensationSummary } from "@/lib/graphql/benefits";
import { useRichT } from "@trenova/shared/i18n/rich";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatMinor, formatMinorCompact } from "@trenova/shared/lib/money";
import type { ReactNode } from "react";
import { useDrawn } from "@/hooks/use-drawn";

export type CompensationShape = {
  total: number;
  /** Shares of the bar, 0 to 1. */
  paidShare: number;
  benefitShare: number;
  /** Where the bracket over the driver's contribution starts and how wide it is, 0 to 1. */
  contributionStart: number;
  contributionShare: number;
  /** Axis marks in minor units, from zero up to the largest round step under the total. */
  ticks: number[];
};

const STEPS = [1, 2, 2.5, 5];

/** A round step that gives four or five marks across the total. */
function tickStep(total: number): number {
  if (total <= 0) return 0;
  const rough = total / 5;
  const magnitude = 10 ** Math.floor(Math.log10(rough));
  const step = STEPS.map((factor) => factor * magnitude).find((value) => value >= rough);
  return step ?? 10 * magnitude;
}

/**
 * Total compensation as one bar: what was paid and what the employer put into benefits. The
 * driver's own contribution came out of the pay, so it is bracketed over the end of the paid part
 * rather than added to the total.
 */
export function compensationShape(
  summary: Pick<
    TotalCompensationSummary,
    "grossPayMinor" | "employerBenefitMinor" | "employeeBenefitMinor"
  >,
): CompensationShape {
  const paid = Math.max(0, summary.grossPayMinor);
  const benefits = Math.max(0, summary.employerBenefitMinor);
  const total = paid + benefits;
  if (total === 0) {
    return {
      total: 0,
      paidShare: 0,
      benefitShare: 0,
      contributionStart: 0,
      contributionShare: 0,
      ticks: [],
    };
  }
  const contribution = Math.min(Math.max(0, summary.employeeBenefitMinor), paid);
  const step = tickStep(total);
  const ticks: number[] = [];
  for (let value = 0; value <= total && step > 0; value += step) ticks.push(value);
  return {
    total,
    paidShare: paid / total,
    benefitShare: benefits / total,
    contributionStart: (paid - contribution) / total,
    contributionShare: contribution / total,
    ticks,
  };
}

function pct(share: number): string {
  return `${(share * 100).toFixed(3)}%`;
}

export function CompensationBar({ summary }: { summary: TotalCompensationSummary }) {
  const t = useT();
  const rt = useRichT();
  const strong = {
    b: (chunk: ReactNode) => <b className="font-medium text-foreground tabular-nums">{chunk}</b>,
  };
  const shape = compensationShape(summary);
  const drawn = useDrawn();
  const contribution = Math.min(summary.employeeBenefitMinor, summary.grossPayMinor);

  return (
    <section
      aria-label={t("Total compensation this year")}
      className="rounded-lg border bg-card px-4 pt-3.5 pb-4"
    >
      <div className="flex flex-wrap items-end justify-between gap-x-4 gap-y-2">
        <div>
          <p className="text-xs font-medium text-foreground-subtle">
            {t("Total compensation in {0}", summary.planYear)}
          </p>
          <p className="text-2xl font-semibold tabular-nums">
            {formatMinor(summary.totalCompensationMinor)}
          </p>
        </div>
      </div>
      {shape.total === 0 ? (
        <p className="mt-3 text-xs text-muted-foreground">
          {t("Nothing paid or contributed this year yet.")}
        </p>
      ) : (
        <>
          <div
            role="img"
            data-testid="compensation-bar"
            aria-label={t(
              "Total compensation {0}: paid {1} and employer benefits {2}. The driver's contribution of {3} comes out of the pay.",
              formatMinor(summary.totalCompensationMinor),
              formatMinor(summary.grossPayMinor),
              formatMinor(summary.employerBenefitMinor),
              formatMinor(contribution),
            )}
            className="relative mt-4 pt-7.5"
          >
            {shape.contributionShare > 0 ? (
              <div
                aria-hidden
                className="absolute top-0 h-6.5 transition-opacity delay-320 duration-260 ease-swift motion-reduce:transition-none"
                style={{
                  left: pct(shape.contributionStart),
                  width: pct(shape.contributionShare),
                  opacity: drawn ? 1 : 0,
                }}
              >
                <span className="absolute inset-x-0 bottom-1 h-1.5 rounded-t-xs border border-b-0 border-foreground" />
                <span className="absolute right-0 bottom-3.5 text-xs whitespace-nowrap tabular-nums">
                  {t("Their contribution {0}", formatMinor(contribution))}
                </span>
              </div>
            ) : null}
            <div className="flex h-4 overflow-hidden rounded-xs bg-surface-active">
              <span
                className="flex h-full w-full origin-left transition-transform duration-420 ease-settle motion-reduce:transition-none"
                style={{ transform: `scaleX(${drawn ? 1 : 0})` }}
              >
                <span className="h-full bg-ink" style={{ width: pct(shape.paidShare) }} />
                <span
                  className="h-full border-l border-card bg-chart-3"
                  style={{ width: pct(shape.benefitShare) }}
                />
              </span>
            </div>
            <div aria-hidden className="relative mt-1.5 h-5.5 border-t border-border">
              {shape.ticks.map((tick, index) => {
                const last = index === shape.ticks.length - 1 && tick / shape.total > 0.9;
                return (
                  <span
                    key={tick}
                    className="absolute top-0 pt-1.5 text-2xs whitespace-nowrap text-foreground-subtle tabular-nums"
                    style={{
                      left: pct(tick / shape.total),
                      transform: last ? "translateX(-100%)" : undefined,
                    }}
                  >
                    {formatMinorCompact(tick)}
                  </span>
                );
              })}
            </div>
          </div>
          <div className="mt-1.5 flex flex-wrap gap-x-5 gap-y-1.5 text-xs text-muted-foreground">
            <span className="inline-flex items-center gap-1.5">
              <span className="size-2.5 rounded-xs bg-ink" />
              {rt("Paid <b>{0}</b>", strong, formatMinor(summary.grossPayMinor))}
            </span>
            <span className="inline-flex items-center gap-1.5">
              <span className="size-2.5 rounded-xs bg-chart-3" />
              {rt(
                "Employer benefits <b>{0}</b>",
                strong,
                formatMinor(summary.employerBenefitMinor),
              )}
            </span>
            {contribution > 0 ? (
              <span className="inline-flex items-center gap-1.5">
                <span className="h-1.5 w-2.5 rounded-t-xs border border-b-0 border-foreground" />
                {rt(
                  "Their contribution <b>{0}</b>, taken from the pay",
                  strong,
                  formatMinor(contribution),
                )}
              </span>
            ) : null}
          </div>
        </>
      )}
    </section>
  );
}
