import { Button } from "@trenova/shared/components/ui/button";
import { formatUnixTime } from "@trenova/shared/lib/date";
import {
  NOTIFICATION_STATUS_LABEL,
  URGENCY_LABEL,
  URGENCY_STYLES,
  formatCountdown,
  formatDetentionMinutes,
} from "@trenova/shared/lib/detention";
import { cn } from "@trenova/shared/lib/utils";
import type { DeskEntry } from "@trenova/shared/types/detention";
import { ChevronRightIcon } from "lucide-react";
import { memo } from "react";
import { DeskClockTrack } from "./desk-clock-track";
import { DeskMoney } from "./desk-money";
import { useSendDetentionNotice } from "./use-detention-actions";

type DetentionDeskRowProps = {
  entry: DeskEntry;
  nowSeconds: number;
  isSelected: boolean;
  onOpen: (occurrenceId: string) => void;
};

/**
 * One stop, one line. The columns are fixed so the same fact sits in the same
 * place on every row — which is what makes a desk scannable — and the only
 * action that has a deadline is the only control on it.
 */
export const DetentionDeskRow = memo(function DetentionDeskRow({
  entry,
  nowSeconds,
  isSelected,
  onOpen,
}: DetentionDeskRowProps) {
  const { occurrence } = entry;
  const styles = URGENCY_STYLES[entry.urgency];
  const sendNotice = useSendDetentionNotice(occurrence.id);

  const facility = occurrence.locationName || "Unknown facility";
  const isLost = entry.urgency === "Lost";
  const onSiteMinutes = Math.max(
    0,
    Math.round(((occurrence.clockStopAt ?? nowSeconds) - occurrence.clockStartAt) / 60),
  );

  const context = [
    occurrence.customerName || "Unknown customer",
    occurrence.shipmentProNumber ? `PRO ${occurrence.shipmentProNumber}` : null,
    occurrence.stopType,
    occurrence.arrivedLate ? `late ${formatDetentionMinutes(occurrence.lateByMinutes)}` : null,
  ]
    .filter(Boolean)
    .join(" · ");

  const noticePending =
    occurrence.notificationStatus === "Pending" || occurrence.notificationStatus === "Late";

  return (
    <div
      role="button"
      tabIndex={0}
      aria-label={`Open the claim file for ${facility}`}
      title={URGENCY_LABEL[entry.urgency]}
      onClick={() => onOpen(occurrence.id)}
      onKeyDown={(event) => {
        if (event.key === "Enter" || event.key === " ") {
          event.preventDefault();
          onOpen(occurrence.id);
        }
      }}
      className={cn(
        "group flex cursor-pointer items-center gap-3 px-3 py-2.5 text-left transition-colors",
        "focus-visible:ring-ring focus-visible:-outline-offset-2 focus-visible:ring-2 focus-visible:outline-none",
        isSelected ? "bg-muted" : "hover:bg-muted/50",
        isLost && "opacity-60",
      )}
    >
      <span className={cn("size-1.5 shrink-0 rounded-full", styles.dot)} />

      <div className="min-w-0 flex-1">
        <p className="truncate text-sm leading-tight">{facility}</p>
        <p className="truncate text-xs leading-tight text-muted-foreground">{context}</p>
      </div>

      <div className="hidden w-44 shrink-0 md:block">
        <DeskClockTrack entry={entry} nowSeconds={nowSeconds} />
        <p className="mt-1.5 truncate text-2xs leading-none text-muted-foreground tabular-nums">
          {formatDetentionMinutes(onSiteMinutes)} on site · free{" "}
          {entry.minutesUntilFreeEnds > 0 ? "ends" : "ended"}{" "}
          {formatUnixTime(occurrence.freeTimeExpiresAt)}
        </p>
      </div>

      <div className="hidden w-40 shrink-0 items-center gap-2 lg:flex">
        {occurrence.notificationStatus === "NotRequired" ? (
          <span className="text-xs text-muted-foreground">—</span>
        ) : noticePending ? (
          <>
            <span
              className={cn("flex-1 truncate text-xs tabular-nums", styles.text)}
              title="Send the customer notice before this deadline or the charge stops being collectable"
            >
              Notice {formatCountdown(entry.minutesUntilNoticeDue)}
            </span>
            {entry.noticeWindowOpen && (
              <Button
                size="xxs"
                variant="outline"
                className="text-2xs"
                isLoading={sendNotice.isPending}
                loadingText="Sending"
                onClick={(event) => {
                  event.stopPropagation();
                  sendNotice.mutate(undefined);
                }}
              >
                Send
              </Button>
            )}
          </>
        ) : (
          <span className="truncate text-xs text-muted-foreground">
            {NOTIFICATION_STATUS_LABEL[occurrence.notificationStatus]}
          </span>
        )}
      </div>

      <div className="w-24 shrink-0 text-right">
        <DeskMoney
          value={entry.amountAtRisk}
          currency={occurrence.currency}
          className="text-sm leading-tight font-medium"
        />
        <p className="text-2xs leading-tight text-muted-foreground tabular-nums">
          {occurrence.roundedMinutes > 0
            ? `${formatDetentionMinutes(occurrence.roundedMinutes)} billable`
            : "free time"}
        </p>
      </div>

      <ChevronRightIcon className="size-3.5 shrink-0 text-muted-foreground opacity-0 transition-opacity group-hover:opacity-100" />
    </div>
  );
});
