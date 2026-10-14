import type { AIAnalyticsSeries } from "@/lib/graphql/ai-analytics";
import { cn } from "@trenova/shared/lib/utils";
import type { SeriesColors } from "./analytics-model";
import { Swatch } from "./card";
import { highlightClass, useHighlight } from "./highlight";

/** A heatmap's column heads: one per series, its colour over its short name. */
export function HeatHead({
  series,
  colors,
}: {
  series: readonly AIAnalyticsSeries[];
  colors: SeriesColors;
}) {
  const { hot } = useHighlight();

  return (
    <>
      {series.map((item) => (
        <span
          key={item.key}
          role="columnheader"
          className={cn("an-hh c", highlightClass(hot, item.key))}
          title={`${item.label} · ${item.providers.map((provider) => provider.name).join(" · ")}`}
        >
          <Swatch color={colors.get(item.key)} />
          <b>{item.shortLabel}</b>
        </span>
      ))}
    </>
  );
}
