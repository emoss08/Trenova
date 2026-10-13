import type { InboundMessageDetail } from "@/lib/graphql/inbox";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixDateTimeShort, formatUnixTime } from "@trenova/shared/lib/date";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { memo } from "react";
import { USER_FORMATTERS } from "../action-copy";
import { activityLine } from "./activity-copy";

const DAY = 86_400;
const STAGGER_MS = 40;
const STAGGER_LIMIT = 12;

/** What happened to the message, oldest first, each line a sentence with its time. */
export const ActivityTrace = memo(function ActivityTrace({
  activity,
  now,
}: {
  activity: InboundMessageDetail["activity"];
  now: number;
}) {
  const t = useT();
  const meId = useAuthStore((state) => state.user?.id ?? null);
  if (activity.length === 0) {
    return null;
  }

  return (
    <ol
      aria-label={t("What happened")}
      className="border-border-subtle flex flex-col border-t border-dashed pt-1"
    >
      {activity.map((entry, index) => (
        <li
          key={entry.id}
          style={{ animationDelay: `${Math.min(index, STAGGER_LIMIT) * STAGGER_MS}ms` }}
          className="text-foreground-muted animate-rise grid grid-cols-[12px_minmax(0,1fr)_auto] items-center gap-2 py-1.25 text-sm"
        >
          <span aria-hidden className="bg-success size-1.5 justify-self-center rounded-full" />
          <span className="min-w-0">
            {activityLine(t, entry, { meId, format: USER_FORMATTERS })}
          </span>
          <time
            dateTime={new Date(entry.createdAt * 1000).toISOString()}
            className="text-foreground-subtle font-mono text-2xs tabular-nums"
          >
            {now - entry.createdAt < DAY
              ? formatUnixTime(entry.createdAt)
              : formatUnixDateTimeShort(entry.createdAt)}
          </time>
        </li>
      ))}
    </ol>
  );
});
