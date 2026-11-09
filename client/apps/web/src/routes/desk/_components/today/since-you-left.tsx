import type { DeskActivityEntry } from "@/lib/graphql/desk-attention";
import { useT } from "@trenova/shared/i18n/use-t";
import { Link } from "react-router";

/**
 * What changed since the reader last looked, counted from the records and
 * never written by a model: what came in, what was cleared, the changes other
 * people approved, and the writes that did not go through. Each line is keyed
 * in the gutter like the ledger under it.
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
  const lines = [
    unseen > 0 && (
      <p key="unseen">
        <span className="dk-w-since-k">{t("Came in")}</span>
        <Link to="/desk/needs" className="dk-w-lnk ui-focus-ring">
          {t("{0, plural, one {# new thing needs a look.} other {# new things need a look.}}", unseen)}
        </Link>
      </p>
    ),
    cleared > 0 && (
      <p key="cleared">
        <span className="dk-w-since-k">{t("Cleared")}</span>
        <span>
          {t("{0, plural, one {# item was cleared, by people or on its own.} other {# items were cleared, by people or on their own.}}", cleared)}
        </span>
      </p>
    ),
    ...approvedByOthers.map((entry) => (
      <p key={entry.id}>
        <span className="dk-w-since-k">{entry.decidedBy?.name ?? t("Someone")}</span>
        <span>{t("Approved: {0}", entry.title)}</span>
      </p>
    )),
    failed > 0 && (
      <p key="failed">
        <span className="dk-w-since-k">{t("Failed")}</span>
        <span className="dk-w-bad">
          {t("{0, plural, one {# change didn't go through.} other {# changes didn't go through.}}", failed)}
        </span>
      </p>
    ),
  ].filter(Boolean);

  if (lines.length === 0) {
    return <p className="dk-w-quiet">{t("Nothing changed since you last looked.")}</p>;
  }

  return <div className="dk-w-since">{lines}</div>;
}
