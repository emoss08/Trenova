import { useT } from "@trenova/shared/i18n/use-t";
import { SegmentedControl } from "@trenova/shared/components/ui/segmented-control";

export const TREND_RANGE_WEEKS = [13, 26, 52] as const;

export type TrendRangeWeeks = (typeof TREND_RANGE_WEEKS)[number];

export const DEFAULT_TREND_RANGE: TrendRangeWeeks = 13;

function toRange(value: string): TrendRangeWeeks {
  return TREND_RANGE_WEEKS.find((weeks) => String(weeks) === value) ?? DEFAULT_TREND_RANGE;
}

export function RangeToggle({
  value,
  onChange,
}: {
  value: TrendRangeWeeks;
  onChange: (value: TrendRangeWeeks) => void;
}) {
  const t = useT();

  return (
    <SegmentedControl
      aria-label={t("Trend range")}
      value={String(value)}
      onValueChange={(next) => onChange(toRange(next))}
      items={TREND_RANGE_WEEKS.map((weeks) => ({
        value: String(weeks),
        label: t("{0}w", weeks),
      }))}
    />
  );
}
