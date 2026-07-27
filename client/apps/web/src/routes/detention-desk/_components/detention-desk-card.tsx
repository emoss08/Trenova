import { useApiMutation } from "@/hooks/use-api-mutation";
import { queries } from "@/lib/queries";
import { apiService } from "@/services/api";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import {
  URGENCY_LABEL,
  URGENCY_STYLES,
  formatCountdown,
  formatDetentionMinutes,
  freeTimeConsumedPercent,
  scoreBand,
  SCORE_BAND_STYLES,
} from "@trenova/shared/lib/detention";
import { cn, formatCurrency } from "@trenova/shared/lib/utils";
import type { DeskEntry, DetentionOccurrence } from "@trenova/shared/types/detention";
import { useQueryClient } from "@tanstack/react-query";
import { MailIcon } from "lucide-react";
import { memo } from "react";
import { toast } from "sonner";

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
  const onSiteMinutes = Math.max(0, Math.round((nowSeconds - occurrence.clockStartAt) / 60));

  const showNotice =
    occurrence.notificationStatus !== "NotRequired" &&
    occurrence.notificationStatus !== "Sent";

  const queryClient = useQueryClient();
  const sendNotice = useApiMutation<DetentionOccurrence, undefined>({
    mutationFn: () => apiService.detentionService.sendNotice(occurrence.id),
    onSuccess: () => {
      toast.success("Notice sent", {
        description: "The customer notice went out and was recorded as evidence.",
      });
      void queryClient.invalidateQueries({ queryKey: queries.detention.desk().queryKey });
    },
    resourceName: "Detention Notice",
  });

  return (
    <div
      role="button"
      tabIndex={0}
      aria-label={`Open detention details for ${occurrence.locationName || "unknown facility"}`}
      onClick={() => onOpen(occurrence.id)}
      onKeyDown={(event) => {
        if (event.key === "Enter" || event.key === " ") {
          event.preventDefault();
          onOpen(occurrence.id);
        }
      }}
      className={cn(
        "group bg-card flex w-full cursor-pointer flex-col gap-3 rounded-lg border p-4 text-left",
        "transition-all duration-200 hover:-translate-y-px hover:border-muted-foreground/30 hover:shadow-sm",
        "focus-visible:ring-ring focus-visible:ring-2 focus-visible:outline-none",
      )}
    >
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0">
          <div className="flex items-center gap-2">
            <span className={cn("size-1.5 shrink-0 rounded-full", styles.dot)} />
            <Badge className={cn("shrink-0 border-none", styles.badge)}>
              {URGENCY_LABEL[entry.urgency]}
            </Badge>
            {occurrence.arrivedLate && (
              <Badge variant="outline" className="shrink-0 text-[10px]">
                Arrived late
              </Badge>
            )}
          </div>
          <p className="mt-2 truncate text-sm font-medium">
            {occurrence.locationName || "Unknown facility"}
          </p>
          <p className="text-muted-foreground truncate text-xs">
            {occurrence.customerName || "Unknown customer"}
            {occurrence.shipmentProNumber && <> · PRO {occurrence.shipmentProNumber}</>} ·{" "}
            {occurrence.stopType}
          </p>
          <p className="text-muted-foreground truncate text-xs">
            {formatDetentionMinutes(onSiteMinutes)} on site · free time expires{" "}
            {formatCountdown(entry.minutesUntilFreeEnds)}
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
        <div className="bg-muted h-1.5 w-full overflow-hidden rounded-full">
          <div
            className={cn("h-full rounded-full transition-all duration-500", styles.bar)}
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
            <Button
              size="sm"
              variant="outline"
              className="h-6 gap-1 text-xs"
              disabled={sendNotice.isPending}
              onClick={(event) => {
                event.stopPropagation();
                sendNotice.mutate(undefined);
              }}
            >
              <MailIcon className="size-3" />
              {sendNotice.isPending ? "Sending…" : "Send notice"}
            </Button>
          )}
        </div>
      )}

      {occurrence.collectabilityScore > 0 && (
        <div className="flex items-center gap-2">
          <Badge
            className={cn(
              "border-none text-[11px]",
              SCORE_BAND_STYLES[scoreBand(occurrence.collectabilityScore)],
            )}
          >
            Defensibility {occurrence.collectabilityScore}/100
          </Badge>
        </div>
      )}
    </div>
  );
});
