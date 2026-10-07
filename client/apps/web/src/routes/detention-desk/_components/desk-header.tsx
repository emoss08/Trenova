import { useT } from "@trenova/shared/i18n/use-t";
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
import { cn, formatCurrency } from "@trenova/shared/lib/utils";
import { Mail01Icon, RefreshCw02Icon } from "@trenova/shared/components/icons";
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
  const t = useT();
  const { isError, isLoading, isFetching, updatedSecondsAgo } = desk;

  const label = isError
    ? t("Not refreshing")
    : isLoading
      ? t("Connecting")
      : isFetching
        ? t("Syncing")
        : t("Updated {0}", formatSecondsAgo(updatedSecondsAgo ?? 0));

  return (
    <span
      aria-live="polite"
      className={cn(
        "inline-flex items-center gap-1.5 text-xs",
        isError ? "text-danger-foreground" : "text-muted-foreground",
      )}
    >
      <span
        className={cn(
          "size-1.5 rounded-full",
          isError ? "bg-danger" : "bg-success",
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
  const t = useT();

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
          <Mail01Icon className="size-3.5" />
          {t("{0, plural, one {Send # notice} other {Send # notices}}", noticeQueue.length)}
        </Button>
      )}

      <Button
        variant="outline"
        size="icon-sm"
        aria-label={t("Refresh the detention desk")}
        title={t("Refresh the detention desk")}
        onClick={() => void refetch()}
        disabled={isFetching}
      >
        <RefreshCw02Icon
          className={cn("size-3.5", isFetching && "animate-spin motion-reduce:animate-none")}
        />
      </Button>

      <AlertDialog open={confirmOpen} onOpenChange={setConfirmOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle className="text-lg font-semibold">
              {t(
                "{0, plural, one {Send # detention notice?} other {Send # detention notices?}}",
                noticeQueue.length,
              )}
            </AlertDialogTitle>
            <AlertDialogDescription>
              {t(
                "Each customer is notified that detention has started and the notice is recorded as evidence, which is what keeps {0} defensible in a dispute.",
                formatCurrency(queueTotal, noticeQueue[0]?.occurrence.currency),
              )}
            </AlertDialogDescription>
          </AlertDialogHeader>

          <ul className="flex flex-col gap-1 rounded-lg border p-2 text-xs">
            {preview.map((entry) => (
              <li key={entry.occurrence.id} className="flex items-center justify-between gap-3">
                <span className="min-w-0 truncate">
                  {entry.occurrence.locationName || t("Unknown facility")}
                  <span className="text-muted-foreground">
                    {" · "}
                    {entry.occurrence.customerName || t("Unknown customer")}
                  </span>
                </span>
                <span className="text-muted-foreground shrink-0 tabular-nums">
                  {formatCountdown(entry.minutesUntilNoticeDue)}
                </span>
              </li>
            ))}
            {overflow > 0 && (
              <li className="text-muted-foreground">{t("and {0} more", overflow)}</li>
            )}
          </ul>

          <AlertDialogFooter>
            <AlertDialogCancel variant="outline" size="default">
              {t("Cancel")}
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
              {t("{0, plural, one {Send notice} other {Send notices}}", noticeQueue.length)}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  );
}
