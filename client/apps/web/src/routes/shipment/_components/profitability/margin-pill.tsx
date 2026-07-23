import { getMarginTone, resolveTargetMarginPct } from "@/lib/profitability";
import { formatPercent } from "@/lib/utils";
import { toneVar } from "../analytics/kpi/tone";

export function MarginPill({
  marginPct,
  targetMarginPercent,
  className,
}: {
  marginPct: number;
  targetMarginPercent?: string | null;
  className?: string;
}) {
  const tone = getMarginTone(marginPct, resolveTargetMarginPct(targetMarginPercent));

  return (
    <span
      className={`inline-flex items-center gap-1.5 text-sm font-medium tabular-nums ${className ?? ""}`}
      style={{ color: toneVar(tone) }}
    >
      <span
        className="size-1.5 shrink-0 rounded-full"
        style={{ backgroundColor: toneVar(tone) }}
      />
      {formatPercent(marginPct)}
    </span>
  );
}
