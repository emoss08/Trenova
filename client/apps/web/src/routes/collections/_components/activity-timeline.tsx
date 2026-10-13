import { useT } from "@trenova/shared/i18n/use-t";
import { Badge } from "@trenova/shared/components/ui/badge";
import { formatMinor } from "@trenova/shared/lib/benefits";
import { formatUnixDateMedium, formatUnixDateTimeMedium } from "@trenova/shared/lib/date";
import { ACTIVITY_KIND_LABEL } from "@/lib/collection-labels";
import type { CollectionActivity } from "@/types/collection";

/** The account's history, newest first, as it was recorded. */
export function ActivityTimeline({ activities }: { activities: CollectionActivity[] }) {
  const t = useT();

  if (activities.length === 0) {
    return (
      <p className="text-muted-foreground px-3 py-3 text-xs">
        {t("Nobody has worked this account yet. Calls, emails, notes and promises appear here.")}
      </p>
    );
  }

  return (
    <ol className="divide-border divide-y">
      {activities.map((activity) => (
        <li key={activity.id} className="flex flex-col gap-1 px-3 py-2.5">
          <div className="flex items-center justify-between gap-2">
            <div className="flex min-w-0 items-center gap-2">
              <Badge variant={activity.kind === "PromiseToPay" ? "info" : "neutral"}>
                {ACTIVITY_KIND_LABEL[activity.kind]}
              </Badge>
              <span className="text-muted-foreground truncate text-xs">
                {activity.contactName
                  ? t("{0} with {1}", activity.createdByName, activity.contactName)
                  : activity.createdByName}
              </span>
            </div>
            <span className="text-muted-foreground shrink-0 text-xs">
              {formatUnixDateTimeMedium(activity.createdAt)}
            </span>
          </div>
          {activity.kind === "PromiseToPay" &&
          activity.promisedAmountMinor !== null &&
          activity.promisedOn !== null ? (
            <span className="text-sm tabular-nums">
              {t(
                "Promised {0} by {1}",
                formatMinor(activity.promisedAmountMinor, activity.currencyCode),
                formatUnixDateMedium(activity.promisedOn),
              )}
            </span>
          ) : null}
          {activity.summary ? (
            <p className="text-sm whitespace-pre-wrap">{activity.summary}</p>
          ) : null}
          {activity.kind === "FollowUpScheduled" && activity.followUpOn !== null ? (
            <span className="text-muted-foreground text-xs">
              {t("Follow up on {0}", formatUnixDateMedium(activity.followUpOn))}
            </span>
          ) : null}
          {activity.kind !== "FollowUpScheduled" && activity.followUpOn !== null ? (
            <span className="text-muted-foreground text-xs">
              {t("Next follow-up {0}", formatUnixDateMedium(activity.followUpOn))}
            </span>
          ) : null}
        </li>
      ))}
    </ol>
  );
}
