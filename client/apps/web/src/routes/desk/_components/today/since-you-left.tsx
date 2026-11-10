import type { DeskActivityEntry } from "@/lib/graphql/desk-attention";
import { useT } from "@trenova/shared/i18n/use-t";
import type { ReactNode } from "react";
import { Link } from "react-router";

function Line({ k, children }: { k: string; children: ReactNode }) {
  return (
    <p>
      <span className="dk-dy-since-k">{k}</span>
      <span>{children}</span>
    </p>
  );
}

/**
 * What changed since the reader last looked, the prototype's `.since`: each
 * line keyed in the gutter like the ledger under it. Counted from the records
 * and never written by a model: what came in, what was cleared, the changes
 * other people approved, and the writes that did not go through.
 */
export function SinceYouLeft({
  unseen,
  cleared,
  approvedByOthers,
  failed,
}: {
  unseen: number;
  cleared: number;
  approvedByOthers: readonly DeskActivityEntry[];
  failed: number;
}) {
  const t = useT();
  const lines: ReactNode[] = [];

  if (unseen > 0) {
    lines.push(
      <Line key="unseen" k={t("Came in")}>
        <Link to="/desk/needs" className="dk-dy-lnk ui-focus-ring">
          {t(
            "{0, plural, one {# new thing needs a look.} other {# new things need a look.}}",
            unseen,
          )}
        </Link>
      </Line>,
    );
  }
  if (cleared > 0) {
    lines.push(
      <Line key="cleared" k={t("Cleared")}>
        {t(
          "{0, plural, one {# item was cleared, by people or on its own.} other {# items were cleared, by people or on their own.}}",
          cleared,
        )}
      </Line>,
    );
  }
  for (const entry of approvedByOthers) {
    lines.push(
      <Line key={entry.id} k={entry.decidedBy?.name ?? t("Someone")}>
        {t("Approved: {0}", entry.title)}
      </Line>,
    );
  }
  if (failed > 0) {
    lines.push(
      <Line key="failed" k={t("Failed")}>
        <span className="dk-dy-bad">
          {t(
            "{0, plural, one {# change didn't go through.} other {# changes didn't go through.}}",
            failed,
          )}
        </span>
      </Line>,
    );
  }

  if (lines.length === 0) {
    return <p className="dk-dy-quiet">{t("Nothing changed since you last looked.")}</p>;
  }

  return <div className="dk-dy-since">{lines}</div>;
}
