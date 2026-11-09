import { DeskAgentTile } from "@/components/desk-chat/desk-agent-tile";
import type { DeskActivityEntry } from "@/lib/graphql/desk-attention";
import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import { formatUnixTime } from "@trenova/shared/lib/date";
import { Link } from "react-router";
import { LifeBadge, outcomeLife } from "../attention/life-badge";

/** The records a line names, the rest counted. */
export function recordLine(t: TranslateFn, entry: DeskActivityEntry): string {
  const named = entry.records.filter((record) => record.readable && record.label !== "");
  const unnamed = entry.records.length - named.length + entry.recordsMore;
  const names = named.map((record) => record.label).join(", ");
  if (names === "") {
    return unnamed > 0 ? t("{0, plural, one {# record} other {# records}}", unnamed) : "";
  }
  return unnamed > 0 ? t("{0} and {1, plural, one {# more} other {# more}}", names, unnamed) : names;
}

/** Who stood behind a write: the person who approved it, or the rule it ran under. */
export function decidedLine(t: TranslateFn, entry: DeskActivityEntry): string {
  switch (entry.outcome) {
    case "Approved":
      return entry.decidedBy ? t("Approved by {0}", entry.decidedBy.name) : t("Approved");
    case "OnItsOwn":
      return t("Ran under AI control");
    case "Previewed":
      return t("Previewed, never made");
    default:
      return entry.detail;
  }
}

/** Where a line of the ledger opens: the write, with why it could run and what it changed. */
export function handledPath(entry: Pick<DeskActivityEntry, "id">): string {
  return `/desk/handled/${encodeURIComponent(entry.id)}`;
}

/**
 * What agents settled, newest first, as a ledger with the time in the gutter:
 * what ran on its own under a rule AI control set, what ran after a person
 * approved it, what was only previewed, and what did not go through. Every
 * line is a write the server recorded settling; a run of the same write folds
 * into one line.
 */
export function NovaHandled({
  entries,
  timezone,
}: {
  entries: readonly DeskActivityEntry[];
  timezone: string;
}) {
  const t = useT();

  return (
    <div className="dk-w-rows">
      {entries.map((entry) => {
        const records = recordLine(t, entry);
        return (
          <Link key={entry.id} to={handledPath(entry)} className="dk-w-lg ui-focus-ring">
            <time className="dk-w-lg-tm">{formatUnixTime(entry.at, { timezone })}</time>
            <span className="dk-w-lg-main">
              <span className="dk-w-lg-ti">{entry.title}</span>
              <span className="dk-w-lg-mt">
                {entry.agent && <DeskAgentTile agent={entry.agent} size="xs" />}
                <span>{entry.agent?.name ?? t("An agent")}</span>
                {records !== "" && (
                  <>
                    <span>·</span>
                    <span>{records}</span>
                  </>
                )}
                <span>·</span>
                <span>{decidedLine(t, entry)}</span>
              </span>
            </span>
            <span className="dk-w-lg-rt">
              <LifeBadge life={outcomeLife(entry.outcome)} />
            </span>
          </Link>
        );
      })}
    </div>
  );
}
