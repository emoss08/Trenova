import { WithheldValue } from "@trenova/shared/components/withheld-value";
import { formatList } from "@trenova/shared/i18n/format";
import { useT } from "@trenova/shared/i18n/use-t";
import type { MoveRow, MoveStanding } from "./carrier-settlement-card";

const STRIP_ORDER: readonly MoveStanding[] = ["matched", "off", "waiting", "none"];

/**
 * The carrier's invoices against the statement: one cell a move, matched
 * filled, off the buy rate in amber, waiting to be accepted outlined in
 * amber, not invoiced yet left open; the count of each keyed beneath; and the
 * moves that need someone listed, each with its invoice against the buy rate.
 * A matched move is counted, not listed.
 */
export function InvoiceReconciliation({
  tally,
  rows,
  offBy,
}: {
  tally: Record<MoveStanding, number>;
  rows: readonly MoveRow[];
  offBy: string;
}) {
  const t = useT();
  const labels: Record<MoveStanding, string> = {
    matched: t("Matched"),
    off: t("Off the buy rate"),
    waiting: t("Waiting to be accepted"),
    none: t("Not invoiced yet"),
  };
  const cells = STRIP_ORDER.flatMap((standing) =>
    Array.from({ length: tally[standing] }, (_, index) => ({
      key: `${standing}:${index}`,
      standing,
    })),
  );
  const keys = STRIP_ORDER.filter((standing) => tally[standing] > 0);
  return (
    <div className="dk-cset-recon">
      <div
        className="dk-cset-strip"
        role="img"
        aria-label={formatList(
          keys.map((standing) =>
            t("{0, plural, one {# move} other {# moves}} {1}", tally[standing], labels[standing]),
          ),
          "unit",
        )}
      >
        {cells.map((cell) => (
          <i key={cell.key} className={`dk-${cell.standing}`} />
        ))}
      </div>
      <dl className="dk-cset-keys">
        {keys.map((standing) => (
          <div key={standing} className={`dk-${standing}`}>
            <dt>{labels[standing]}</dt>
            <dd>
              {standing === "off" && offBy
                ? t("{0, plural, one {# move} other {# moves}} · {1}", tally[standing], offBy)
                : t("{0, plural, one {# move} other {# moves}}", tally[standing])}
            </dd>
          </div>
        ))}
      </dl>
      {rows.length > 0 && (
        <ul className="dk-cset-moves">
          {rows.map((row) => (
            <li key={row.moveId} className={`dk-${row.standing}`}>
              <span className="dk-what">
                <b>
                  {row.title}
                  {row.pro && <span> · {row.pro}</span>}
                </b>
                <span>{row.line}</span>
              </span>
              <span className="dk-cset-var">{row.variance}</span>
              <span className="dk-cset-tag">{row.status}</span>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}

/** Where the money goes: the remit-to address, how the carrier is paid, and when. */
export function Remittance({
  withheld,
  lines,
  method,
  terms,
}: {
  /** The carrier's payment details are withheld: the person may not read carriers. */
  withheld: boolean;
  lines: readonly string[];
  /** Null when withheld and the settlement sets no method of its own. */
  method: string | null;
  terms: string;
}) {
  const t = useT();
  return (
    <dl className="dk-cset-remit">
      <div>
        <dt>{t("Remit to")}</dt>
        <dd>
          {withheld ? (
            <WithheldValue kind="carrier" />
          ) : lines.length > 0 ? (
            lines.map((line) => <span key={line}>{line}</span>)
          ) : (
            t("No remit-to address on file — set it on the carrier record.")
          )}
        </dd>
      </div>
      {method === null ? (
        <div>
          <dt>{t("Method")}</dt>
          <dd>
            <WithheldValue kind="carrier" />
          </dd>
        </div>
      ) : (
        method && (
          <div>
            <dt>{t("Method")}</dt>
            <dd>{method}</dd>
          </div>
        )
      )}
      <div>
        <dt>{t("Terms")}</dt>
        <dd>{terms}</dd>
      </div>
    </dl>
  );
}
