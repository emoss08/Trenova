import type { LeaveEntitlement } from "@/lib/graphql/worker-leave";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatLeaveHours } from "@trenova/shared/lib/leave";
import { cn } from "@trenova/shared/lib/utils";
import { useEffect, useState, type CSSProperties } from "react";
import { entitlementBar } from "./leave-model";

const TICKS_STYLE: CSSProperties = {
  backgroundImage:
    "repeating-linear-gradient(to right, transparent 0, transparent calc(var(--tick) - 1px), var(--color-card) calc(var(--tick) - 1px), var(--color-card) var(--tick))",
};

/**
 * The hours left of the entitlement as a ruled bar, one tick a week, with the week marks under
 * it. It fills once when first drawn.
 */
export function EntitlementBar({
  entitlement,
  scale = true,
}: {
  entitlement: LeaveEntitlement;
  /** Draws 0, the half and the whole in weeks under the bar. */
  scale?: boolean;
}) {
  const t = useT();
  const [drawn, setDrawn] = useState(false);
  useEffect(() => {
    if (drawn) return;
    const frame = requestAnimationFrame(() => setDrawn(true));
    return () => cancelAnimationFrame(frame);
  }, [drawn]);
  const bar = entitlementBar(entitlement);

  return (
    <div className="flex flex-col">
      <div
        role="img"
        aria-label={t(
          "{0} of {1} hours left, {2} weeks.",
          formatLeaveHours(entitlement.remainingHours),
          formatLeaveHours(entitlement.totalHours),
          formatLeaveHours(entitlement.remainingWeeks),
        )}
        className="relative h-2.5 overflow-hidden rounded-xs bg-surface-active"
        style={{ "--tick": `calc(100% / ${bar.weeks})` } as CSSProperties}
      >
        <span
          className={cn(
            "absolute inset-y-0 left-0 origin-left transition-transform duration-420 ease-settle motion-reduce:transition-none",
            entitlement.exhausted ? "bg-danger" : "bg-ink",
          )}
          style={{ width: `${bar.remainingPercent}%`, transform: `scaleX(${drawn ? 1 : 0})` }}
        />
        <span aria-hidden className="pointer-events-none absolute inset-0" style={TICKS_STYLE} />
      </div>
      {scale ? (
        <div className="mt-1.5 flex justify-between text-2xs text-foreground-subtle tabular-nums">
          <span>0</span>
          <span>{t("{0} weeks", formatLeaveHours(String(bar.weeks / 2)))}</span>
          <span>{t("{0} weeks", bar.weeks)}</span>
        </div>
      ) : null}
    </div>
  );
}
