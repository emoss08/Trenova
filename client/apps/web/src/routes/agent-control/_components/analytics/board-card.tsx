import type { AIAnalyticsSeries } from "@/lib/graphql/ai-analytics";
import { useRichT } from "@trenova/shared/i18n/rich";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { memo, useMemo, type CSSProperties } from "react";
import {
  board,
  boardSentence,
  display,
  METRICS,
  type MetricKey,
  type SeriesColors,
} from "./analytics-model";
import { AnalyticsCard, Better } from "./card";
import { hoverProps, useSetHot } from "./highlight";
import { ProviderMarks } from "./provider-marks";

type BoardCardProps = {
  metric: MetricKey;
  series: readonly AIAnalyticsSeries[];
  colors: SeriesColors;
  onOpen: (key: string) => void;
};

/**
 * One leaderboard metric: a sentence comparing the best and the worst, then a bar per series
 * best first, with series too small to rank greyed at the end.
 */
export const BoardCard = memo(function BoardCard({
  metric: key,
  series,
  colors,
  onOpen,
}: BoardCardProps) {
  const t = useT();
  const rt = useRichT();
  const setHot = useSetHot();
  const metric = METRICS[key];
  const entries = useMemo(() => board(series, metric), [metric, series]);
  const sentence = useMemo(() => boardSentence(t, rt, metric, entries), [entries, metric, rt, t]);
  const max = Math.max(...entries.map((entry) => entry.value), 0) || 1;

  return (
    <AnalyticsCard
      title={t(metric.label)}
      description={t(metric.description)}
      right={<Better higher={metric.higherIsBetter} />}
      className="an-bd"
    >
      {sentence !== null && <p className="an-say">{sentence}</p>}
      {entries.length === 0 ? (
        <p className="an-nil">{t("Nothing in this scope has this figure yet.")}</p>
      ) : (
        <div className="an-vb">
          {entries.map((entry, index) => {
            const value = display(t, metric, entry.value);
            return (
              <button
                key={entry.series.key}
                type="button"
                data-series={entry.series.key}
                className={cn(
                  "an-vc ui-focus-ring",
                  entry.series.few && "few",
                  entry.best && "is-best",
                )}
                title={
                  entry.series.few
                    ? t("{0} · {1} · too few runs to rank", entry.series.label, value)
                    : `${entry.series.label} · ${value}`
                }
                onClick={() => onOpen(entry.series.key)}
                {...hoverProps(setHot, entry.series.key)}
              >
                <span className="an-vv mono">{value}</span>
                <span className="an-vt">
                  <i
                    style={
                      {
                        "--col": colors.get(entry.series.key),
                        height: `${entry.value === 0 ? 1.5 : Math.max(3, (entry.value / max) * 100)}%`,
                        animationDelay: `${index * 50}ms`,
                      } as CSSProperties
                    }
                  />
                </span>
                <span className="an-vl">
                  <ProviderMarks providers={entry.series.providers} s={16} />
                  <em>{entry.series.shortLabel}</em>
                </span>
              </button>
            );
          })}
        </div>
      )}
    </AnalyticsCard>
  );
});
