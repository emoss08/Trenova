import type { JournalCardReversal } from "@/lib/graphql/journal-cards";
import { Button } from "@trenova/shared/components/ui/button";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { CardIcon } from "../../record-card/record-card";
import { cardDay, minorMoney } from "../../record-card/record-card-values";
import type { PairRow } from "./journal-entry-card";

/**
 * A reversed entry and the entry that reversed it, the original first, each
 * with what it moved the books by, bracketed to what the two leave behind.
 * The other entry opens in the card's place.
 */
export function JournalReversalPair({
  rows,
  onOpen,
}: {
  rows: readonly PairRow[];
  onOpen?: (entryId: string) => void;
}) {
  const t = useT();
  return (
    <div className="dk-jen-pair">
      <ol>
        {rows.map((row) => {
          const amount = minorMoney(row.entry.totalDebit);
          const body = (
            <>
              <em>
                {row.mine ? t("This entry") : row.sign < 0 ? t("Reversed by") : t("Reverses")}
              </em>
              <span className="dk-what">
                <b>{row.entry.entryNumber}</b>
                <span>
                  {row.entry.postedAt
                    ? t("Posted {0}", cardDay(row.entry.postedAt))
                    : t("Not posted yet")}
                </span>
              </span>
              <span className="dk-amt">
                {row.sign > 0 ? "+" : "−"}
                {amount}
              </span>
            </>
          );
          return (
            <li key={row.entry.id} className={cn(row.mine && "dk-this")}>
              {row.mine || !onOpen ? (
                <div className="dk-jen-pair-row">{body}</div>
              ) : (
                <Button
                  variant="bare"
                  size="bare"
                  className="dk-jen-pair-row"
                  onClick={() => onOpen(row.entry.id)}
                >
                  {body}
                </Button>
              )}
            </li>
          );
        })}
      </ol>
      <span className="dk-brace" aria-hidden="true" />
      <div className="dk-net">
        <b>{minorMoney(0)}</b>
        <span>{t("left in the books")}</span>
      </div>
    </div>
  );
}

/** Why an entry was reversed, kept as written: the request's code beside its words. */
export function JournalReversalWhy({ code, text }: { code: string; text: string }) {
  return (
    <p className="dk-jen-why">
      {code && <code>{code}</code>}
      {text}
    </p>
  );
}

/**
 * A reversal request that stands against the entry, or the last one refused:
 * what became of it, who asked and when, and why.
 */
export function JournalReversalSlip({
  reversal,
  title,
  refused,
}: {
  reversal: JournalCardReversal;
  title: string;
  refused: boolean;
}) {
  const said = refused
    ? reversal.status === "Rejected"
      ? reversal.rejectionReason
      : reversal.cancelReason
    : null;
  return (
    <div className={cn("dk-jen-slip", refused && "dk-refused")}>
      <span className="dk-jen-slip-i" aria-hidden="true">
        <CardIcon name={refused ? "x" : "undo"} size={13} stroke={2.2} />
      </span>
      <div className="dk-jen-slip-t">
        <b>{title}</b>
        <span>
          {reversal.reasonCode && <code>{reversal.reasonCode}</code>}
          {reversal.reasonText}
        </span>
        {said && <span className="dk-jen-slip-said">{said}</span>}
      </div>
    </div>
  );
}
