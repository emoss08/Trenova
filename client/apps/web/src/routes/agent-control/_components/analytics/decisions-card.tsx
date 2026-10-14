import type { AIAnalyticsSeries } from "@/lib/graphql/ai-analytics";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { memo, useMemo } from "react";
import { percent, type SeriesColors } from "./analytics-model";
import { AnalyticsCard, Swatch } from "./card";
import { highlightClass, hoverProps, useHighlight } from "./highlight";

/** What people did with each series' proposals: a read on judgment, not just completion. */
export const DecisionsCard = memo(function DecisionsCard({
  series,
  colors,
  onOpen,
}: {
  series: readonly AIAnalyticsSeries[];
  colors: SeriesColors;
  onOpen: (key: string) => void;
}) {
  const t = useT();
  const { hot, setHot } = useHighlight();
  const rows = useMemo(
    () =>
      series
        .filter((item) => item.decided > 0)
        .sort((left, right) => (right.approvalRate ?? 0) - (left.approvalRate ?? 0)),
    [series],
  );

  return (
    <AnalyticsCard
      title={t("What people did with its proposals")}
      description={t("A read on judgment, not just completion.")}
      className="an-dc"
    >
      {rows.length === 0 ? (
        <p className="an-nil">{t("Nobody decided a proposal in this scope.")}</p>
      ) : (
        rows.map((item, index) => (
          <button
            key={item.key}
            type="button"
            className={cn("an-dcr ui-focus-ring", highlightClass(hot, item.key))}
            onClick={() => onOpen(item.key)}
            {...hoverProps(setHot, item.key)}
          >
            <span className="an-dcn">
              <Swatch color={colors.get(item.key)} />
              <b>{item.label}</b>
              <em className="mono">{item.decided.toLocaleString()}</em>
            </span>
            <span
              className="an-stk"
              role="img"
              aria-label={t(
                "{0} approved as proposed, {1} with changes, {2} rejected",
                item.accepted,
                item.modified,
                item.rejected,
              )}
            >
              {(
                [
                  ["a", item.accepted],
                  ["m", item.modified],
                  ["r", item.rejected],
                ] as const
              ).map(([kind, count]) =>
                count > 0 ? (
                  <i
                    key={kind}
                    className={`p-${kind}`}
                    style={{ flexGrow: count, animationDelay: `${index * 50}ms` }}
                  />
                ) : null,
              )}
            </span>
            <span className="an-dcv mono">{percent(item.approvalRate)}</span>
          </button>
        ))
      )}
      <div className="an-lgd">
        <span>
          <i className="p-a" />
          {t("Approved as proposed")}
        </span>
        <span>
          <i className="p-m" />
          {t("With changes")}
        </span>
        <span>
          <i className="p-r" />
          {t("Rejected")}
        </span>
      </div>
    </AnalyticsCard>
  );
});
