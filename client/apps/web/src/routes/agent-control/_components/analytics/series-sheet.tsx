import { describeToolCall } from "@/components/assistant/tool-presentation";
import type { AIAnalyticsSeries, AIAnalyticsTool } from "@/lib/graphql/ai-analytics";
import { formatNumber } from "@trenova/shared/i18n/format";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixMonthDay } from "@trenova/shared/lib/date";
import type { AiAnalyticsCompareBy } from "@trenova/graphql/generated/graphql";
import { Ic } from "../kit/ic";
import { ReadSheet } from "../kit/read-sheet";
import { KV } from "../kit/values";
import {
  advisorFigures,
  formatCost,
  formatLatency,
  formatRunTime,
  formatSpend,
  isLocal,
  OUTCOMES,
  outcomeCount,
  percent,
  runTimeOf,
} from "./analytics-model";
import { ProviderMarks } from "./provider-marks";

const TOOLS_SHOWN = 4;
const TOOL_BAR_CEILING = 0.15;

type SeriesSheetProps = {
  open: boolean;
  series: AIAnalyticsSeries | null;
  tools: readonly AIAnalyticsTool[];
  compareBy: AiAnalyticsCompareBy;
  onOpenProvider: (providerId: string) => void;
  onOpenRuns: () => void;
  onClose: () => void;
};

/** One model or provider read whole: its figures, run time by outcome, and the tools it gets wrong. */
export function SeriesSheet({
  open,
  series,
  tools,
  compareBy,
  onOpenProvider,
  onOpenRuns,
  onClose,
}: SeriesSheetProps) {
  const t = useT();
  if (!series) {
    return null;
  }
  const served =
    compareBy === "Provider"
      ? series.models.join(" · ")
      : series.providers.map((provider) => provider.name).join(" · ");
  const quiet = series.quiet;
  const advisor = advisorFigures(series.advisor);
  const wrong = tools
    .map((tool) => ({ tool, cell: tool.cells.find((cell) => cell.seriesKey === series.key) }))
    .filter((entry) => entry.cell && !entry.cell.few)
    .map((entry) => ({
      tool: entry.tool,
      calls: entry.cell?.calls ?? 0,
      rate: entry.cell?.failureRate ?? 0,
    }))
    .sort((left, right) => right.rate - left.rate)
    .slice(0, TOOLS_SHOWN);

  return (
    <ReadSheet
      open={open}
      onClose={onClose}
      label={series.label}
      head={
        <>
          <ProviderMarks providers={series.providers} s={30} limit={3} />
          <div className="sh-t">
            <b>{series.label}</b>
            <span>
              {served}
              {quiet && series.lastRanAt
                ? ` · ${t("last ran {0}", formatUnixMonthDay(series.lastRanAt))}`
                : ""}
            </span>
          </div>
        </>
      }
    >
      <KV
        items={[
          [t("Runs"), formatNumber(series.runs)],
          [t("Completed"), percent(series.completedRate)],
          [
            t("Tool calls failed"),
            series.toolCalls > 0
              ? t(
                  "{0} · {1} the model's mistake",
                  percent(series.toolFailureRate ?? 0, 1),
                  percent(series.modelMistakeRate ?? 0),
                )
              : t("No tool calls"),
          ],
          [
            t("Approved as proposed"),
            series.decided > 0
              ? t("{0} of {1}", percent(series.approvalRate), formatNumber(series.decided))
              : "—",
          ],
          [t("Rated helpful"), series.rated > 0 ? percent(series.helpfulRate) : "—"],
          [
            t("Cost"),
            isLocal(series)
              ? t("Local")
              : t(
                  "{0} · {1} a run",
                  formatSpend(Number(series.costUsd) || 0),
                  formatCost((Number(series.costUsd) || 0) / Math.max(1, series.runs)),
                ),
          ],
          advisor && [
            t("Advisor"),
            t(
              "{0, plural, one {# call} other {# calls}} · {1} prompt / {2} output tokens · {3}",
              advisor.calls,
              formatNumber(advisor.promptTokens),
              formatNumber(advisor.outputTokens),
              formatSpend(advisor.costUsd),
            ),
          ],
          [
            t("First token"),
            series.firstTokenMedianMs === null || series.firstTokenMedianMs === undefined
              ? "—"
              : t("{0} median", formatLatency(series.firstTokenMedianMs)),
          ],
        ]}
      />
      <div className="sh-p">
        <h4 className="sh-k">{t("Run time by outcome")}</h4>
        <table className="ut an-mini">
          <thead>
            <tr>
              <th>{t("Outcome")}</th>
              <th className="r">{t("Runs")}</th>
              <th className="r">{t("Median")}</th>
              <th className="r">{t("Slowest 5%")}</th>
            </tr>
          </thead>
          <tbody>
            {OUTCOMES.map((item) => {
              const runTime = runTimeOf(series, item.outcome);
              return (
                <tr key={item.outcome}>
                  <td>
                    <span className="an-ol">
                      <i className={`o-${item.outcome}`} />
                      {t(item.title)}
                    </span>
                  </td>
                  <td className="r mono">{formatNumber(outcomeCount(series, item.outcome))}</td>
                  <td className="r mono">{runTime ? formatRunTime(runTime.medianMs) : "—"}</td>
                  <td className="r mono">{runTime ? formatRunTime(runTime.p95Ms) : "—"}</td>
                </tr>
              );
            })}
          </tbody>
        </table>
      </div>
      {wrong.length > 0 && (
        <div className="sh-p">
          <h4 className="sh-k">{t("Tools it gets wrong most")}</h4>
          {wrong.map((entry) => (
            <div key={entry.tool.name} className="an-wr">
              <span className="an-wl">
                <b>{describeToolCall(entry.tool.name, null).title}</b>
                <span className="mono">
                  {t("{0} of {1}", percent(entry.rate, 1), formatNumber(entry.calls))}
                </span>
              </span>
              <span className="ubar">
                <i
                  className="f"
                  style={{ width: `${Math.min(100, (entry.rate / TOOL_BAR_CEILING) * 100)}%` }}
                />
              </span>
            </div>
          ))}
        </div>
      )}
      <div className="ad-bar sh-f">
        {series.providers
          .filter((provider) => provider.id && !provider.removed)
          .map((provider) => (
            <button
              key={provider.id}
              type="button"
              className="xa ui-focus-ring"
              onClick={() => onOpenProvider(provider.id ?? "")}
            >
              <Ic n="server" s={13} />
              {provider.name}
            </button>
          ))}
        <span className="sp" />
        <button type="button" className="xa ui-focus-ring" onClick={onOpenRuns}>
          {t("Its runs")}
        </button>
      </div>
    </ReadSheet>
  );
}
