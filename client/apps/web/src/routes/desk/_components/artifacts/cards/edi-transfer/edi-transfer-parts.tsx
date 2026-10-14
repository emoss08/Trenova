import { ArrowDownLeftIcon, ArrowUpRightIcon } from "@trenova/shared/components/icons";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import type { TransferDirection } from "@/lib/edi/transfer-sides";
import { CardIcon, type CardFact } from "../../record-card/record-card";
import { cardMoment } from "../../record-card/record-card-values";
import type {
  ChargeLine,
  DecisionGate,
  FreightLine,
  MappingLedger,
  ReasonNote,
  RouteStop,
} from "./edi-transfer-card";

/**
 * The tender as a ticket torn from a book: its transaction set over the tear,
 * and under it the way it travels, in to the reader or out to the other side.
 */
export function TenderTicket({ direction }: { direction: TransferDirection }) {
  const Arrow = direction === "inbound" ? ArrowDownLeftIcon : ArrowUpRightIcon;
  return (
    <span className={cn("dk-tdr-ticket", `dk-${direction}`)} aria-hidden>
      <b>204</b>
      <i />
      <Arrow className="size-3.5" />
    </span>
  );
}

const GATE_ICON: Record<DecisionGate["state"], "check" | "x" | "pause" | "warn" | null> = {
  done: "check",
  now: null,
  warn: "warn",
  failed: "x",
  closed: "x",
  ahead: null,
  skipped: null,
};

/**
 * The tender's way as gates side by side, each a segment of one meter: a gate
 * passed filled, the one it waits at drawn in ink, one held on a mapping in
 * amber, one it failed at in red, one it closed at struck through, and one it
 * never reaches hatched.
 */
export function DecisionGates({ gates }: { gates: readonly DecisionGate[] }) {
  const t = useT();
  return (
    <ol className="dk-tdr-gates" aria-label={t("Decision")}>
      {gates.map((gate) => {
        const icon = GATE_ICON[gate.state];
        return (
          <li
            key={gate.key}
            className={cn("dk-tdr-gate", `dk-${gate.state}`)}
            aria-current={gate.state === "now" || gate.state === "warn" ? "step" : undefined}
          >
            <span className="dk-tdr-bar" aria-hidden />
            <span className="dk-tdr-gl">
              {icon && <CardIcon name={icon} size={11} stroke={2.6} />}
              <b>{gate.label}</b>
            </span>
            {gate.detail && <span className="dk-tdr-gd">{gate.detail}</span>}
            {gate.at ? <time>{cardMoment(gate.at)}</time> : null}
          </li>
        );
      })}
    </ol>
  );
}

/** Why the tender closed where it did, kept as its side wrote it. */
export function ReasonSlip({ note }: { note: ReasonNote }) {
  return (
    <div className={cn("dk-tdr-reason", `dk-${note.tone}`)} role="note" aria-label={note.label}>
      <b>{note.label}</b>
      <p>{note.text}</p>
    </div>
  );
}

/**
 * The tender's stops down one rail: a pickup ringed, a delivery filled, each
 * with its window, what it handles, and whether the partner's location is one
 * of ours yet.
 */
export function TenderRoute({ stops }: { stops: readonly RouteStop[] }) {
  const t = useT();
  return (
    <ol className="dk-tdr-route" aria-label={t("Stops")}>
      {stops.map((stop) => (
        <li key={stop.key} className={cn("dk-tdr-stop", `dk-${stop.role}`)}>
          <i className="dk-tdr-pin" aria-hidden />
          <div className="dk-tdr-sh">
            <span className="dk-tdr-kind">{stop.kind}</span>
            {stop.window && <time>{stop.window}</time>}
          </div>
          <b className="dk-tdr-place">{stop.place}</b>
          {stop.line && <span className="dk-tdr-line">{stop.line}</span>}
          {(stop.load || stop.mapping) && (
            <span className="dk-tdr-foot">
              {stop.load && <span>{stop.load}</span>}
              {stop.mapping && (
                <span className={cn("dk-tdr-map", `dk-${stop.mapping.state}`)}>
                  <CardIcon
                    name={stop.mapping.state === "mapped" ? "check" : "warn"}
                    size={10}
                    stroke={2.6}
                  />
                  {stop.mapping.state === "mapped"
                    ? t("Ours: {0}", stop.mapping.label)
                    : stop.mapping.label}
                </span>
              )}
            </span>
          )}
        </li>
      ))}
    </ol>
  );
}

/**
 * What the tender moves and what it pays, side by side: its facts and
 * commodities on one half, the rate as a ledger run down to its total on the
 * other.
 */
export function FreightAndRate({
  facts,
  commodities,
  lines,
  total,
}: {
  facts: readonly CardFact[];
  commodities: readonly FreightLine[];
  lines: readonly ChargeLine[];
  total: string;
}) {
  const t = useT();
  return (
    <div className="dk-tdr-freight">
      <div className="dk-tdr-load">
        {facts.length > 0 && (
          <dl className="dk-tdr-facts">
            {facts.map((fact) => (
              <div key={fact.key}>
                <dt>{fact.label}</dt>
                <dd>{fact.value}</dd>
              </div>
            ))}
          </dl>
        )}
        {commodities.length > 0 && (
          <ul className="dk-tdr-goods" aria-label={t("Commodities")}>
            {commodities.map((commodity) => (
              <li key={commodity.key}>
                <b>{commodity.label}</b>
                {commodity.load && <span>{commodity.load}</span>}
              </li>
            ))}
          </ul>
        )}
      </div>
      <div className="dk-tdr-rate">
        {lines.length > 0 || total ? (
          <dl aria-label={t("Rate")}>
            {lines.map((line) => (
              <div key={line.key}>
                <dt>{line.label}</dt>
                <dd>{line.amount}</dd>
              </div>
            ))}
            {total && (
              <div className="dk-tdr-total">
                <dt>{t("Total")}</dt>
                <dd>{total}</dd>
              </div>
            )}
          </dl>
        ) : (
          <p className="dk-rec-note">{t("The tender carries no rate.")}</p>
        )}
      </div>
    </div>
  );
}

/**
 * The partner's codes against our records: a meter of how many are mapped,
 * then each code, those still unmapped first, with the record it maps to.
 */
export function MappingLedgerView({ ledger }: { ledger: MappingLedger }) {
  const t = useT();
  const share = ledger.total > 0 ? ledger.resolved / ledger.total : 1;
  return (
    <div className="dk-tdr-maps">
      <div
        className="dk-tdr-meter"
        role="meter"
        aria-valuemin={0}
        aria-valuemax={ledger.total}
        aria-valuenow={ledger.resolved}
        aria-label={t("{0} of {1} mapped", ledger.resolved, ledger.total)}
      >
        <i style={{ inlineSize: `${Math.round(share * 100)}%` }} />
      </div>
      <ul className="dk-tdr-mlist">
        {ledger.rows.map((row) => (
          <li key={row.key} className={row.resolved ? "dk-ok" : "dk-gap"}>
            <span className="dk-tdr-ment">{row.entity}</span>
            <code>{row.source}</code>
            <CardIcon name="forward" size={11} />
            {row.resolved ? <b>{row.target || row.source}</b> : <em>{t("Not mapped")}</em>}
          </li>
        ))}
      </ul>
    </div>
  );
}
