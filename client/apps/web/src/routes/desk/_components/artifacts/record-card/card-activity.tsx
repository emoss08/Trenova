import type { ActivityEntry } from "@/lib/graphql/record-activity";
import { useT } from "@trenova/shared/i18n/use-t";
import { CardPagedList } from "./card-paged-list";
import type { PagedRows } from "./paged-rows";
import { CardSection, CardTimeline } from "./record-card";
import { cardMoment } from "./record-card-values";
import { activityActor, activityChanges, activitySentence } from "./record-activity";

/** The changes an entry shows under its sentence, when the card shows them. */
const SHOWN_CHANGES = 2;

/**
 * What has been done to a record, newest first, from its audit trail: each
 * entry as a sentence (created, changed which fields, approved) with who did
 * it and when, and older entries on request. Reads are left out. The card asks
 * for it only when the person may read the audit log.
 */
export function CardActivity({
  activity,
  emptyText,
  showChanges = false,
}: {
  activity: PagedRows<ActivityEntry>;
  emptyText: string;
  /** Show what each change changed, from what to what, under its sentence. */
  showChanges?: boolean;
}) {
  const t = useT();
  return (
    <CardSection
      title={t("Activity")}
      aside={
        activity.total ? (
          <span>{t("{0, plural, one {# change} other {# changes}}", activity.total)}</span>
        ) : undefined
      }
    >
      <CardPagedList
        list={activity}
        waitRows={3}
        emptyText={emptyText}
        errorText={t("The activity could not be read.")}
        nextErrorText={t("The earlier activity could not be read.")}
        loadingLabel={t("Reading earlier activity…")}
      >
        {(rows) => (
          <CardTimeline
            entries={rows.map((entry) => ({
              id: entry.id,
              text: entry.comment?.trim()
                ? t("{0}: {1}", activitySentence(entry, t), entry.comment.trim())
                : activitySentence(entry, t),
              meta: t("{0} · {1}", activityActor(entry, t), cardMoment(entry.timestamp)),
              detail: showChanges ? <ActivityChanges entry={entry} /> : undefined,
            }))}
          />
        )}
      </CardPagedList>
    </CardSection>
  );
}

/** What one change changed, a field to a line, from what to what. */
function ActivityChanges({ entry }: { entry: ActivityEntry }) {
  const t = useT();
  const changes = activityChanges(entry, SHOWN_CHANGES, t);
  if (changes.length === 0) return null;
  return (
    <dl className="dk-rec-diff">
      {changes.map((change) => (
        <div key={change.field}>
          <dt>{change.field}</dt>
          <dd>
            <s>{change.from}</s>
            <span aria-hidden>→</span>
            <b>{change.to}</b>
          </dd>
        </div>
      ))}
    </dl>
  );
}
