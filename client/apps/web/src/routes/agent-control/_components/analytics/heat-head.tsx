import type { AIAnalyticsSeries } from "@/lib/graphql/ai-analytics";
import type { SeriesColors } from "./analytics-model";
import { Swatch } from "./card";

/** A heatmap's column heads: one per series, its colour over its short name. */
export function HeatHead({
  series,
  colors,
}: {
  series: readonly AIAnalyticsSeries[];
  colors: SeriesColors;
}) {
  return (
    <>
      {series.map((item) => (
        <span
          key={item.key}
          role="columnheader"
          className="an-hh c"
          data-series={item.key}
          title={`${item.label} · ${item.providers.map((provider) => provider.name).join(" · ")}`}
        >
          <Swatch color={colors.get(item.key)} />
          <b>{item.shortLabel}</b>
        </span>
      ))}
    </>
  );
}
