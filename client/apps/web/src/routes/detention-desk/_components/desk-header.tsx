import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@trenova/shared/components/ui/alert-dialog";
import { Button } from "@trenova/shared/components/ui/button";
import { formatSecondsAgo } from "@trenova/shared/lib/date";
import { formatCountdown } from "@trenova/shared/lib/detention";
import { cn, formatCurrency, pluralize } from "@trenova/shared/lib/utils";
import { MailIcon, RefreshCwIcon } from "lucide-react";
import { useState } from "react";
import type { DetentionDeskState } from "./use-detention-desk";
import { useSendDetentionNotices } from "./use-detention-actions";

/** How many stops the confirmation lists by name before it summarizes the rest. */
const NOTICE_PREVIEW_LIMIT = 5;

/**
 * Whether the numbers on screen can be trusted. A desk that quietly stops
 * refreshing is worse than one that says so, because every clock on it keeps
 * ticking as if the data were live.
 */
export function DeskLivePulse({ desk }: { desk: DetentionDeskState }) {
  const { isError, isLoading, isFetching, updatedSecondsAgo } = desk;

  const label = isError
    ? "Not refreshing"
    : isLoading
      ? "Connecting"
      : isFetching
        ? "Syncing"
        : `Updated ${formatSecondsAgo(updatedSecondsAgo ?? 0)}`;

  return (
    <span
      aria-live="polite"
      className={cn(
        "inline-flex items-center gap-1.5 text-xs",
        isError ? "text-red-600 dark:text-red-400" : "text-muted-foreground",
      )}
    >
      <span
        className={cn(
          "size-1.5 rounded-full",
          isError ? "bg-red-500" : "bg-emerald-500",
          isFetching && !isError && "animate-pulse motion-reduce:animate-none",
        )}
      />
      {label}
    </span>
  );
}

/**
 * The two things a clerk does to the whole desk rather than to one stop: clear
 * the notice queue before the deadlines run out, and pull fresh numbers.
 */
export function DeskHeaderActions({ desk }: { desk: DetentionDeskState }) {
  const [confirmOpen, setConfirmOpen] = useState(false);
  const sendNotices = useSendDetentionNotices();
  const { noticeQueue, isFetching, refetch } = desk;

  const queueTotal = noticeQueue.reduce((total, entry) => total + entry.amountAtRisk, 0);
  const preview = noticeQueue.slice(0, NOTICE_PREVIEW_LIMIT);
  const overflow = noticeQueue.length - preview.length;

  return (
    <>
      {noticeQueue.length > 0 && (
        <Button
          size="sm"
          className="gap-1.5 text-xs"
          onClick={() => setConfirmOpen(true)}
          disabled={sendNotices.isPending}
        >
          <MailIcon className="size-3.5" />
          Send {noticeQueue.length} {pluralize("notice", noticeQueue.length)}
        </Button>
      )}

      <Button
        variant="outline"
        size="icon-sm"
        aria-label="Refresh the detention desk"
        title="Refresh the detention desk"
        onClick={() => void refetch()}
        disabled={isFetching}
      >
        <RefreshCwIcon
          className={cn("size-3.5", isFetching && "animate-spin motion-reduce:animate-none")}
        />
      </Button>

      <AlertDialog open={confirmOpen} onOpenChange={setConfirmOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle className="text-lg font-semibold">
              Send {noticeQueue.length} detention {pluralize("notice", noticeQueue.length)}?
            </AlertDialogTitle>
            <AlertDialogDescription>
              Each customer is notified that detention has started and the notice is recorded as
              evidence, which is what keeps{" "}
              {formatCurrency(queueTotal, noticeQueue[0]?.occurrence.currency)} defensible in a
              dispute.
            </AlertDialogDescription>
          </AlertDialogHeader>

          <ul className="flex flex-col gap-1 rounded-lg border p-2 text-xs">
            {preview.map((entry) => (
              <li key={entry.occurrence.id} className="flex items-center justify-between gap-3">
                <span className="min-w-0 truncate">
                  {entry.occurrence.locationName || "Unknown facility"}
                  <span className="text-muted-foreground">
                    {" · "}
                    {entry.occurrence.customerName || "Unknown customer"}
                  </span>
                </span>
                <span className="text-muted-foreground shrink-0 tabular-nums">
                  {formatCountdown(entry.minutesUntilNoticeDue)}
                </span>
              </li>
            ))}
            {overflow > 0 && <li className="text-muted-foreground">and {overflow} more</li>}
          </ul>

          <AlertDialogFooter>
            <AlertDialogCancel variant="outline" size="default">
              Cancel
            </AlertDialogCancel>
            <AlertDialogAction
              size="default"
              isLoading={sendNotices.isPending}
              disabled={sendNotices.isPending}
              onClick={() =>
                sendNotices.mutate(
                  noticeQueue.map((entry) => entry.occurrence.id),
                  { onSettled: () => setConfirmOpen(false) },
                )
              }
            >
              Send {pluralize("notice", noticeQueue.length)}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  );
}
