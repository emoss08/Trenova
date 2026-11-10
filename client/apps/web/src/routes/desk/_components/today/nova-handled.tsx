import { DeskAgentTile } from "@/components/desk-chat/desk-agent-tile";
import type { DeskActivityEntry } from "@/lib/graphql/desk-attention";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixTime } from "@trenova/shared/lib/date";
import { useState } from "react";
import { Link } from "react-router";
import { LifeBadge, outcomeLife } from "../attention/life-badge";
import { decidedLine, handledPath, recordLine } from "../handled/handled-lines";

/** How many lines show before the rest are asked for. */
const SHOWN = 6;

/**
 * What agents settled, newest first, the prototype's ledger (`.lg`): the time
 * in the gutter, what was done, the agent and the records under it, and how
 * it ended on the right. Each line opens the write in full. Past the first
 * few, one quiet line offers the rest already read.
 */
export function NovaHandled({
  entries,
  timezone,
}: {
  entries: readonly DeskActivityEntry[];
  timezone: string;
}) {
  const t = useT();
  const [all, setAll] = useState(false);
  const shown = all ? entries : entries.slice(0, SHOWN);
  const hidden = entries.length - shown.length;

  return (
    <div className="dk-dy-rows">
      {shown.map((entry) => {
        const records = recordLine(t, entry);
        return (
          <Link key={entry.id} to={handledPath(entry)} className="dk-dy-lg ui-focus-ring">
            <time className="dk-dy-lg-tm">{formatUnixTime(entry.at, { timezone })}</time>
            <span className="dk-dy-lg-main">
              <span className="dk-dy-lg-ti">{entry.title}</span>
              <span className="dk-dy-lg-mt">
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
            <span className="dk-dy-lg-rt">
              <LifeBadge life={outcomeLife(entry.outcome)} />
            </span>
          </Link>
        );
      })}
      {hidden > 0 && (
        <button type="button" className="dk-dy-lg-more ui-focus-ring" onClick={() => setAll(true)}>
          {t("{0, plural, one {# more} other {# more}}", hidden)}
        </button>
      )}
    </div>
  );
}
