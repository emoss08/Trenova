import type { AIAnalyticsSeries } from "@/lib/graphql/ai-analytics";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { memo, useMemo } from "react";
import { CAUSES, percent, whoseRows, type SeriesColors } from "./analytics-model";
import { AnalyticsCard, Swatch } from "./card";
import { highlightClass, hoverProps, useHighlight } from "./highlight";

/**
 * Failed tool calls split by whose mistake they were: the model's to the left, the tool's
 * or a rule's to the right, each stacked by cause, as a share of the series' calls.
 */
export const WhoseMistakeCard = memo(function WhoseMistakeCard({
  series,
  colors,
}: {
  series: readonly AIAnalyticsSeries[];
  colors: SeriesColors;
}) {
  const t = useT();
  const { hot, setHot } = useHighlight();
  const rows = useMemo(() => whoseRows(series), [series]);
  const max = Math.max(...rows.map((row) => Math.max(row.modelTotal, row.toolTotal)), 0) || 1;
  const label = (cause: string) => t(CAUSES.find((entry) => entry.cause === cause)?.label ?? cause);

  return (
    <AnalyticsCard
      title={t("Whose mistake was it")}
      description={t("Failed tool calls split by cause, as a share of all calls.")}
      className="an-wh"
    >
      <div className="an-whh" aria-hidden>
        <span>{t("The model")}</span>
        <span />
        <span>{t("The tool or a rule")}</span>
      </div>
      {rows.map((row, index) => (
        <div
          key={row.series.key}
          className={cn("an-whr", highlightClass(hot, row.series.key))}
          {...hoverProps(setHot, row.series.key)}
        >
          <span className="an-whv mono">{percent(row.modelTotal, 1)}</span>
          <span className="an-whb l">
            {row.model.map((part) =>
              part.share > 0 ? (
                <i
                  key={part.cause}
                  className={`r-${part.cause}`}
                  style={{
                    width: `${(part.share / max) * 100}%`,
                    animationDelay: `${index * 50}ms`,
                  }}
                  title={`${label(part.cause)}: ${percent(part.share, 2)}`}
                />
              ) : null,
            )}
          </span>
          <span className="an-whn">
            <Swatch color={colors.get(row.series.key)} />
            {row.series.shortLabel}
          </span>
          <span className="an-whb r">
            {row.tool.map((part) =>
              part.share > 0 ? (
                <i
                  key={part.cause}
                  className={`r-${part.cause}`}
                  style={{
                    width: `${(part.share / max) * 100}%`,
                    animationDelay: `${index * 50}ms`,
                  }}
                  title={`${label(part.cause)}: ${percent(part.share, 2)}`}
                />
              ) : null,
            )}
          </span>
          <span className="an-whv mono r">{percent(row.toolTotal, 1)}</span>
        </div>
      ))}
      <div className="an-wk">
        <div>
          {CAUSES.filter((entry) => entry.blame === "Model").map((entry) => (
            <span key={entry.cause}>
              <i className={`r-${entry.cause}`} />
              {t(entry.label)}
            </span>
          ))}
        </div>
        <div>
          {CAUSES.filter((entry) => entry.blame === "Tool").map((entry) => (
            <span key={entry.cause}>
              <i className={`r-${entry.cause}`} />
              {t(entry.label)}
            </span>
          ))}
        </div>
      </div>
    </AnalyticsCard>
  );
});
