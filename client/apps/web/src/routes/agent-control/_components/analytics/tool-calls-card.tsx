import { describeToolCall } from "@/components/assistant/tool-presentation";
import type { AIAnalyticsSeries, AIAnalyticsTool } from "@/lib/graphql/ai-analytics";
import { formatNumber } from "@trenova/shared/i18n/format";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { memo, type CSSProperties } from "react";
import { failureIntensity, failureShare, type SeriesColors } from "./analytics-model";
import { AnalyticsCard } from "./card";
import { HeatHead } from "./heat-head";

type ToolCallsCardProps = {
  tools: readonly AIAnalyticsTool[];
  series: readonly AIAnalyticsSeries[];
  colors: SeriesColors;
  onOpenCell: (seriesKey: string, toolName: string) => void;
};

/** The share of each tool's calls that failed, by series, worst tool first; a cell opens its failures. */
export const ToolCallsCard = memo(function ToolCallsCard({
  tools,
  series,
  colors,
  onOpenCell,
}: ToolCallsCardProps) {
  const t = useT();
  const keys = new Set(series.map((item) => item.key));
  const rows = tools.filter((tool) =>
    tool.cells.some((cell) => keys.has(cell.seriesKey) && cell.calls > 0),
  );

  return (
    <AnalyticsCard
      title={t("Where tool calls fail")}
      description={t("Share of each tool's calls that failed. Click a cell to see real examples.")}
      className="an-tl"
      right={
        <span className="an-hk" aria-hidden>
          {[0.1, 0.4, 0.7, 1].map((level) => (
            <i key={level} style={{ "--i": level } as CSSProperties} />
          ))}
          <span>{t("0 → 12%+")}</span>
        </span>
      }
    >
      {rows.length === 0 ? (
        <p className="an-nil">{t("No tool was called in this scope.")}</p>
      ) : (
        <div className="an-hmw">
          <div
            className="an-hm"
            role="table"
            aria-label={t("Failed tool calls by tool and series")}
            style={{ "--c": series.length } as CSSProperties}
          >
            <span className="an-hh" role="columnheader" />
            <HeatHead series={series} colors={colors} />
            {rows.map((tool, rowIndex) => {
              const title = describeToolCall(tool.name, null).title;
              return (
                <div key={tool.name} role="row" className="an-hr">
                  <span className="an-tn" role="rowheader" title={title}>
                    <b>{title}</b>
                    <span className="mono">{tool.name}</span>
                  </span>
                  {series.map((item, columnIndex) => {
                    const cell = tool.cells.find((candidate) => candidate.seriesKey === item.key);
                    if (!cell || cell.calls === 0) {
                      return (
                        <span
                          key={item.key}
                          role="cell"
                          className="an-c z"
                          data-series={item.key}
                          aria-label={t("No calls")}
                        >
                          —
                        </span>
                      );
                    }
                    const label = t(
                      "{0} · {1} of {2} calls failed",
                      item.label,
                      formatNumber(cell.failed),
                      formatNumber(cell.calls),
                    );
                    return (
                      <button
                        key={item.key}
                        type="button"
                        role="cell"
                        data-series={item.key}
                        className={cn("an-c ui-focus-ring", cell.few && "few")}
                        style={
                          {
                            "--i": cell.few ? 0 : failureIntensity(cell.failureRate),
                            animationDelay: `${rowIndex * 30 + columnIndex * 20}ms`,
                          } as CSSProperties
                        }
                        title={label}
                        aria-label={label}
                        onClick={() => onOpenCell(item.key, tool.name)}
                      >
                        <span className="mono">
                          {cell.few ? t("few") : failureShare(cell.failureRate)}
                        </span>
                      </button>
                    );
                  })}
                </div>
              );
            })}
          </div>
        </div>
      )}
    </AnalyticsCard>
  );
});
