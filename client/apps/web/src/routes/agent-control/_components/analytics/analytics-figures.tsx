import type { AIAnalyticsTotals } from "@/lib/graphql/ai-analytics";
import { useCountUp } from "@trenova/shared/hooks/use-count-up";
import { formatNumber } from "@trenova/shared/i18n/format";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatPercent } from "@trenova/shared/lib/utils";
import { Figs } from "../kit/layout";
import { advisorFigures, formatCost, formatRunTime, formatSpend, percent } from "./analytics-model";

function CountUp({ value, format }: { value: number; format: (value: number) => string }) {
  return <>{format(useCountUp(value))}</>;
}

/** The scope in five figures; each counts to its new value when the scope changes. */
export function AnalyticsFigures({ totals }: { totals: AIAnalyticsTotals }) {
  const t = useT();
  const failureRate = totals.toolFailureRate ?? 0;
  const spend = Number(totals.costUsd) || 0;
  const perRun = totals.costPerCompletedRun === null ? null : Number(totals.costPerCompletedRun);
  const advisor = advisorFigures(totals.advisor);

  return (
    <Figs
      label={t("The scope in figures")}
      items={[
        {
          label: t("Runs"),
          value: (
            <CountUp value={totals.runs} format={(value) => formatNumber(Math.round(value))} />
          ),
          sub: t("{0, plural, one {# model call} other {# model calls}}", totals.modelCalls),
        },
        {
          label: t("Completed"),
          value: (
            <CountUp
              value={(totals.completedRate ?? 0) * 100}
              format={(value) => formatPercent(value, 1)}
            />
          ),
          sub: t(
            "{0} handed to a person · {1} failed",
            formatNumber(totals.handedOff),
            formatNumber(totals.failed),
          ),
        },
        {
          label: t("Median run"),
          value:
            totals.medianRunMs === null ? (
              "—"
            ) : (
              <CountUp value={totals.medianRunMs} format={formatRunTime} />
            ),
          sub:
            totals.p95RunMs === null
              ? t("No completed runs")
              : t("Slowest 5% took {0} or more", formatRunTime(totals.p95RunMs)),
        },
        {
          label: t("Tool calls failed"),
          value: <CountUp value={failureRate * 100} format={(value) => formatPercent(value, 1)} />,
          tone: totals.toolFailureAlarm ? "t-d" : undefined,
          sub:
            totals.toolCalls > 0
              ? t(
                  "{0} of {1} · {2} the model's mistake",
                  formatNumber(totals.toolCallsFailed),
                  formatNumber(totals.toolCalls),
                  percent(totals.modelShareOfToolFailures ?? 0),
                )
              : t("No tool calls"),
        },
        {
          label: t("Spend"),
          value: <CountUp value={spend} format={formatSpend} />,
          sub:
            perRun === null
              ? totals.pricedRuns === 0
                ? t("Every run was on a local provider")
                : t("Nothing completed")
              : t("{0} per completed run", formatCost(perRun)),
        },
        ...(advisor
          ? [
              {
                label: t("Advisor"),
                value: <CountUp value={advisor.costUsd} format={formatSpend} />,
                sub: t(
                  "{0, plural, one {# call} other {# calls}} · {1} prompt / {2} output tokens",
                  advisor.calls,
                  formatNumber(advisor.promptTokens),
                  formatNumber(advisor.outputTokens),
                ),
              },
            ]
          : []),
      ]}
    />
  );
}
