import type { AIAnalyticsTotals } from "@/lib/graphql/ai-analytics";
import { useCountUp } from "@trenova/shared/hooks/use-count-up";
import { useT } from "@trenova/shared/i18n/use-t";
import { Figs } from "../kit/layout";
import { formatRunTime, formatSpend, percent } from "./analytics-model";

/** Above this share of failed tool calls the figure reads as a problem. */
const TOOL_FAILURE_ALARM = 0.03;

function CountUp({ value, format }: { value: number; format: (value: number) => string }) {
  return <>{format(useCountUp(value))}</>;
}

/** The scope in five figures; each counts to its new value when the scope changes. */
export function AnalyticsFigures({ totals }: { totals: AIAnalyticsTotals }) {
  const t = useT();
  const failureRate = totals.toolFailureRate ?? 0;
  const spend = Number(totals.costUsd) || 0;
  const perRun = totals.costPerCompletedRun === null ? null : Number(totals.costPerCompletedRun);

  return (
    <Figs
      label={t("The scope in figures")}
      items={[
        {
          label: t("Runs"),
          value: (
            <CountUp value={totals.runs} format={(value) => Math.round(value).toLocaleString()} />
          ),
          sub: t("{0, plural, one {# model call} other {# model calls}}", totals.modelCalls),
        },
        {
          label: t("Completed"),
          value: (
            <CountUp
              value={(totals.completedRate ?? 0) * 100}
              format={(value) => `${value.toFixed(1)}%`}
            />
          ),
          sub: t(
            "{0} handed to a person · {1} failed",
            totals.handedOff.toLocaleString(),
            totals.failed.toLocaleString(),
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
          value: <CountUp value={failureRate * 100} format={(value) => `${value.toFixed(1)}%`} />,
          tone: failureRate > TOOL_FAILURE_ALARM ? "t-d" : undefined,
          sub:
            totals.toolCalls > 0
              ? t(
                  "{0} of {1} · {2} the model's mistake",
                  totals.toolCallsFailed.toLocaleString(),
                  totals.toolCalls.toLocaleString(),
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
              : t("{0} per completed run", `$${perRun.toFixed(3)}`),
        },
      ]}
    />
  );
}
