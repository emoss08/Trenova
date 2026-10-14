import type { AIAnalyticsSeries } from "@/lib/graphql/ai-analytics";
import { formatNumber } from "@trenova/shared/i18n/format";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import type { AiAnalyticsCompareBy } from "@trenova/graphql/generated/graphql";
import type { CSSProperties } from "react";
import type { SeriesColors } from "./analytics-model";
import { highlightClass, useHighlight } from "./highlight";

type ComparingChipsProps = {
  series: readonly AIAnalyticsSeries[];
  colors: SeriesColors;
  compareBy: AiAnalyticsCompareBy;
  hidden: readonly string[];
  onToggle: (key: string) => void;
  onShowAll: () => void;
};

/**
 * One chip per model or provider. Hovering one lights it up in every chart; clicking hides
 * it from the charts, though one always stays.
 */
export function ComparingChips({
  series,
  colors,
  compareBy,
  hidden,
  onToggle,
  onShowAll,
}: ComparingChipsProps) {
  const t = useT();
  const { hot, setHot } = useHighlight();
  const shown = series.length - hidden.length;

  return (
    <div className="an-field" role="group" aria-label={t("Compared")}>
      <span className="an-fl">
        {compareBy === "Provider"
          ? t("Comparing {0, plural, one {# provider} other {# providers}}", shown)
          : t("Comparing {0, plural, one {# model} other {# models}}", shown)}
      </span>
      {series.map((item) => {
        const off = hidden.includes(item.key);
        return (
          <button
            key={item.key}
            type="button"
            className={cn(
              "an-fc ui-focus-ring",
              off && "off",
              !off && highlightClass(hot, item.key),
            )}
            style={{ "--col": colors.get(item.key) } as CSSProperties}
            aria-pressed={!off}
            title={off ? t("Show in charts") : t("Hide from charts")}
            onMouseEnter={() => !off && setHot(item.key)}
            onMouseLeave={() => setHot(null)}
            onFocus={() => !off && setHot(item.key)}
            onBlur={() => setHot(null)}
            onClick={() => onToggle(item.key)}
          >
            <i className="an-sw" />
            <b>{item.label}</b>
            <span className="mono">{formatNumber(item.runs)}</span>
          </button>
        );
      })}
      {hidden.length > 0 && (
        <button type="button" className="lnk ui-focus-ring" onClick={onShowAll}>
          {t("Show all")}
        </button>
      )}
    </div>
  );
}
