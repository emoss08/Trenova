import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import {
  formatCountdown,
  formatDetentionMinutes,
  freeTimeConsumedPercent,
  SCORE_BAND_STYLES,
  URGENCY_LABEL,
  URGENCY_STYLES,
} from "@trenova/shared/lib/detention";
import { formatCurrency } from "@trenova/shared/lib/utils";
import { cn } from "@trenova/shared/lib/utils";
import type { DeskEntry } from "@trenova/shared/types/detention";
import { memo } from "react";

type DetentionDeskCardProps = {
  entry: DeskEntry;
  nowSeconds: number;
  onOpen: (occurrenceId: string) => void;
};

/**
 * One live detention clock. The budget bar shows free time consumed and, once
 * exhausted, the money accruing beyond it — so the moment a stop crosses from
 * "waiting" to "costing" is visible without reading a number.
 */
export const DetentionDeskCard = memo(function DetentionDeskCard({
  entry,
  nowSeconds,
  onOpen,
}: DetentionDeskCardProps) {
  const { occurrence } = entry;
  const styles = URGENCY_STYLES[entry.urgency];
  const consumed = freeTimeConsumedPercent(entry, nowSeconds);
  const onSiteMinutes = Math.max(
    0,
    Math.round((nowSeconds - occurrence.clockStartAt) / 60),
  );

  const showNotice =
    occurrence.notificationStatus !== "NotRequired" &&
    occurrence.notificationStatus !== "Sent";

  return (
    <button
      type="button"
      onClick={() => onOpen(occurrence.id)}
      className={cn(
        "group bg-card flex w-full flex-col gap-3 rounded-lg border border-l-4 p-4 text-left",
        "hover:bg-accent/40 transition-colors focus-visible:ring-2 focus-visible:outline-none",
        "focus-visible:ring-ring",
        styles.border,
      )}
    >
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0">
          <div className="flex items-center gap-2">
            <Badge className={cn("shrink-0 border-none", styles.badge)}>
              {URGENCY_LABEL[entry.urgency]}
            </Badge>
            {occurrence.arrivedLate && (
              <Badge variant="outline" className="shrink-0 text-xs">
                Arrived late
              </Badge>
            )}
          </div>
          <p className="mt-2 truncate text-sm font-medium">
            {occurrence.stopType} · {formatDetentionMinutes(onSiteMinutes)} on site
          </p>
          <p className="text-muted-foreground truncate text-xs">
            Free time {formatDetentionMinutes(occurrence.freeMinutesGranted)} ·
            expires {formatCountdown(entry.minutesUntilFreeEnds)}
          </p>
        </div>

        <div className="shrink-0 text-right">
          <p className="text-lg font-semibold tabular-nums">
            {formatCurrency(entry.amountAtRisk, occurrence.currency)}
          </p>
          {occurrence.roundedMinutes > 0 && (
            <p className="text-muted-foreground text-xs tabular-nums">
              {formatDetentionMinutes(occurrence.roundedMinutes)} billable
            </p>
          )}
        </div>
      </div>

      <div className="space-y-1">
        <div className="bg-muted h-2 w-full overflow-hidden rounded-full">
          <div
            className={cn("h-full rounded-full transition-all", styles.bar)}
            style={{ width: `${consumed}%` }}
          />
        </div>
        <div className="text-muted-foreground flex justify-between text-[11px]">
          <span>Free time {consumed}% used</span>
          {occurrence.driverPayAmount > 0 && (
            <span
              className={cn(
                "tabular-nums",
                occurrence.netMargin < 0 && "text-red-600 dark:text-red-400",
              )}
            >
              Margin {formatCurrency(occurrence.netMargin, occurrence.currency)}
            </span>
          )}
        </div>
      </div>

      {showNotice && (
        <div className="bg-muted/50 flex items-center justify-between gap-2 rounded-md px-2 py-1.5">
          <span className="text-xs">
            Notice {occurrence.notificationStatus.toLowerCase()} ·{" "}
            <span className="tabular-nums">
              {formatCountdown(entry.minutesUntilNoticeDue)}
            </span>
          </span>
          {entry.noticeWindowOpen && (
            <Button size="sm" variant="outline" className="h-6 text-xs">
              Send notice
            </Button>
          )}
        </div>
      )}

      {occurrence.collectabilityScore > 0 && (
        <div className="flex items-center gap-2">
          <Badge
            className={cn(
              "border-none text-[11px]",
              SCORE_BAND_STYLES[
                occurrence.collectabilityScore >= 85
                  ? "Strong"
                  : occurrence.collectabilityScore >= 65
                    ? "Adequate"
                    : occurrence.collectabilityScore >= 40
                      ? "Weak"
                      : "AtRisk"
              ],
            )}
          >
            Defensibility {occurrence.collectabilityScore}/100
          </Badge>
        </div>
      )}
    </button>
  );
});
