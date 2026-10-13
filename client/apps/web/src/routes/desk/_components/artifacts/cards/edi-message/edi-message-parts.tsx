import { ArrowDownLeftIcon, ArrowUpRightIcon, Mail01Icon } from "@trenova/shared/components/icons";
import { formatNumber } from "@trenova/shared/i18n/format";
import { useT } from "@trenova/shared/i18n/use-t";
import { Button } from "@trenova/shared/components/ui/button";
import { cn } from "@trenova/shared/lib/utils";
import type { EDIInspectionDiagnostic, EDIMessage } from "@trenova/shared/types/edi";
import { useState, type ReactNode } from "react";
import { CardIcon, type CardCheckState } from "../../record-card/record-card";
import { cardMoment } from "../../record-card/record-card-values";
import {
  foldTape,
  journeyPosition,
  type EnvelopeLayer,
  type JourneyStation,
  type PlainReading,
  type TapeSegment,
} from "./edi-message-card";

/** The segments the tape shows before it folds the rest away. */
const TAPE_HEAD = 12;

/**
 * The message's stamp: its transaction set, franked with the way it travels,
 * out to the partner or in from them.
 */
export function MessageStamp({
  set,
  direction,
}: {
  set: EDIMessage["transactionSet"];
  direction: EDIMessage["direction"];
}) {
  const Arrow = direction === "Inbound" ? ArrowDownLeftIcon : ArrowUpRightIcon;
  return (
    <span className={cn("dk-edi-stamp", direction === "Inbound" ? "dk-in" : "dk-out")} aria-hidden>
      <b>{set}</b>
      <Arrow className="size-3" />
    </span>
  );
}

const STATION_ICON: Record<JourneyStation["state"], "check" | "x" | null> = {
  done: "check",
  now: null,
  failed: "x",
  ahead: null,
  skipped: null,
};

/**
 * The way the message travels as a wire between its stops, the envelope
 * resting at the furthest stop it reached. A stop it reached is ticked, one it
 * failed at is crossed, one ahead is an open ring and one it never meets is
 * dashed; the wire is drawn solid as far as the envelope has travelled.
 */
export function MessageWay({ stations }: { stations: readonly JourneyStation[] }) {
  const t = useT();
  const here = journeyPosition(stations);
  return (
    <ol className="dk-edi-way" aria-label={t("Its way")}>
      {stations.map((station, index) => {
        const icon = STATION_ICON[station.state];
        return (
          <li
            key={station.key}
            className={cn(
              "dk-edi-stop",
              `dk-${station.state}`,
              index <= here && "dk-travelled",
              index === here && "dk-here",
            )}
            aria-current={index === here ? "step" : undefined}
          >
            <span className="dk-edi-node" aria-hidden>
              {index === here && station.state !== "failed" ? (
                <Mail01Icon className="size-3.5" />
              ) : icon ? (
                <CardIcon name={icon} size={12} stroke={2.6} />
              ) : null}
            </span>
            <b>{station.label}</b>
            <span>{station.detail}</span>
            {station.at ? <time>{cardMoment(station.at)}</time> : null}
          </li>
        );
      })}
    </ol>
  );
}

/**
 * The message's X12 envelope drawn as envelopes inside one another: the
 * interchange holds the functional group, which holds the transaction set.
 * Each names its segment pair and control number, and says whether its trailer
 * closes it with what it opened with.
 */
export function EnvelopeNest({ layers }: { layers: readonly EnvelopeLayer[] }) {
  const t = useT();
  return layers.reduceRight<ReactNode>(
    (inner, layer) => (
      <div
        className={cn("dk-edi-nest", `dk-edi-${layer.key}`)}
        role="group"
        aria-label={t("{0} {1}", layer.label, layer.control)}
      >
        <div className="dk-edi-flap">
          <span className="dk-edi-pair">{layer.pair}</span>
          <b>{layer.label}</b>
          <code>{layer.control || "—"}</code>
          {layer.detail && <span className="dk-edi-flap-d">{layer.detail}</span>}
          {layer.check && (
            <span className={cn("dk-edi-close", `dk-${layer.check}`)}>
              <CardIcon name={layer.check === "ok" ? "check" : "x"} size={10} stroke={2.8} />
              {layer.checkText}
            </span>
          )}
        </div>
        {inner}
      </div>
    ),
    null,
  );
}

const READING_ICON: Record<CardCheckState, "check" | "warn" | "x" | "info"> = {
  ok: "check",
  warn: "warn",
  bad: "x",
  info: "info",
};

/**
 * What the message says, decoded: its headline in the colour of the news, the
 * place or party under it, a figure beside it, then the stops, charges or
 * faults it carries and its facts as fields on a stub.
 */
export function PlainReadingView({ reading }: { reading: PlainReading }) {
  const t = useT();
  return (
    <div className={cn("dk-edi-read", `dk-${reading.state}`)}>
      <div className="dk-edi-read-h">
        <span className="dk-edi-read-i" aria-hidden>
          <CardIcon name={READING_ICON[reading.state]} size={13} stroke={2.6} />
        </span>
        <span className="dk-edi-read-t">
          <b>{reading.headline}</b>
          {reading.line && <span>{reading.line}</span>}
        </span>
        {reading.figure && <span className="dk-edi-read-fig">{reading.figure}</span>}
      </div>
      {reading.stops.length > 0 && (
        <ol className="dk-edi-stops" aria-label={t("Stops")}>
          {reading.stops.map((stop) => (
            <li key={stop.key}>
              <i aria-hidden />
              <span>
                <b>{stop.place}</b>
                <span>{stop.window ? t("{0} · {1}", stop.kind, stop.window) : stop.kind}</span>
              </span>
            </li>
          ))}
        </ol>
      )}
      {reading.lines.length > 0 && (
        <dl className="dk-edi-lines" aria-label={t("Charges")}>
          {reading.lines.map((line) => (
            <div key={line.key}>
              <dt>{line.label}</dt>
              <dd>{line.amount}</dd>
            </div>
          ))}
        </dl>
      )}
      {reading.faults.length > 0 && (
        <ul className="dk-edi-faults" aria-label={t("Faults")}>
          {reading.faults.map((fault) => (
            <li key={fault.key}>
              <code>{fault.label}</code>
              <span>{fault.amount}</span>
            </li>
          ))}
        </ul>
      )}
      {reading.facts.length > 0 && (
        <dl className="dk-edi-pass">
          {reading.facts.map((fact) => (
            <div key={fact.key}>
              <dt>{fact.label}</dt>
              <dd>{fact.value}</dd>
            </div>
          ))}
        </dl>
      )}
    </div>
  );
}

function severityOf(diagnostic: EDIInspectionDiagnostic): "bad" | "warn" | "info" {
  return diagnostic.severity === "Error"
    ? "bad"
    : diagnostic.severity === "Warning"
      ? "warn"
      : "info";
}

/** What the validator said of a segment, pinned under it, with the fix it suggests. */
function SegmentMark({ diagnostic }: { diagnostic: EDIInspectionDiagnostic }) {
  const t = useT();
  const where =
    diagnostic.elementPosition > 0
      ? t("{0}, element {1}", diagnostic.segmentId ?? "", formatNumber(diagnostic.elementPosition))
      : "";
  return (
    <p className={cn("dk-edi-mark", `dk-${severityOf(diagnostic)}`)}>
      <CardIcon
        name={
          severityOf(diagnostic) === "bad"
            ? "x"
            : severityOf(diagnostic) === "warn"
              ? "warn"
              : "info"
        }
        size={11}
        stroke={2.6}
      />
      <span>
        <b>{diagnostic.message}</b>
        <span>
          {[
            where,
            diagnostic.code,
            diagnostic.suggestedFix ? t("Fix: {0}", diagnostic.suggestedFix) : "",
          ]
            .filter(Boolean)
            .join(" · ")}
        </span>
      </span>
    </p>
  );
}

/** One segment on the tape: its place, its id, each element in turn, and what was said of it. */
function TapeLine({ row }: { row: TapeSegment }) {
  const marked = new Set(row.marks.map((mark) => mark.elementPosition));
  const worst = row.marks.reduce<"bad" | "warn" | "info" | null>((current, mark) => {
    const severity = severityOf(mark);
    if (current === "bad" || severity === "bad") return "bad";
    if (current === "warn" || severity === "warn") return "warn";
    return severity;
  }, null);
  return (
    <div
      role="listitem"
      className={cn(
        "dk-edi-seg",
        `dk-${row.role}`,
        worst && `dk-${worst}`,
        row.malformed && "dk-malformed",
      )}
    >
      <span className="dk-edi-n" aria-hidden>
        {formatNumber(row.index + 1)}
      </span>
      <code title={row.name}>
        <b>{row.id}</b>
        {row.elements.map((element) => (
          <span key={element.position}>
            <i aria-hidden>*</i>
            <span
              className={cn(element.empty && "dk-empty", marked.has(element.position) && "dk-flag")}
              title={element.label}
            >
              {element.value}
            </span>
          </span>
        ))}
        <i aria-hidden>~</i>
      </code>
      {row.marks.map((mark, index) => (
        // oxlint-disable-next-line react/no-array-index-key -- a segment's diagnostics never reorder
        <SegmentMark key={`${mark.code}-${index}`} diagnostic={mark} />
      ))}
    </div>
  );
}

/**
 * The X12 itself as a tape of segments on paper: the envelope set back, the
 * body in ink, every element apart, each one the validator flagged underlined
 * and what it said pinned beneath. A long tape folds to its first segments and
 * the flagged ones past them, until it is unfolded.
 */
export function X12Tape({
  rows,
  loose,
}: {
  rows: readonly TapeSegment[];
  loose: readonly EDIInspectionDiagnostic[];
}) {
  const t = useT();
  const [unfolded, setUnfolded] = useState(false);
  const runs = foldTape(rows, TAPE_HEAD, unfolded);
  return (
    <div className="dk-edi-paper">
      {loose.length > 0 && (
        <div className="dk-edi-loose">
          {loose.map((mark, index) => (
            // oxlint-disable-next-line react/no-array-index-key -- the message's diagnostics never reorder
            <SegmentMark key={`${mark.code}-${index}`} diagnostic={mark} />
          ))}
        </div>
      )}
      <div className="dk-edi-tape" role="list" aria-label={t("X12 segments")}>
        {runs.map((run) =>
          run.kind === "fold" ? (
            <Button
              key={run.key}
              variant="bare"
              size="bare"
              className="dk-edi-fold"
              onClick={() => setUnfolded(true)}
            >
              {t("{0, plural, one {# segment folded} other {# segments folded}}", run.count)}
            </Button>
          ) : (
            <TapeLine key={run.row.index} row={run.row} />
          ),
        )}
      </div>
      {rows.length > TAPE_HEAD && (
        <Button
          variant="bare"
          size="bare"
          className="dk-edi-unfold"
          onClick={() => setUnfolded((current) => !current)}
          aria-expanded={unfolded}
        >
          <CardIcon name={unfolded ? "up" : "down"} size={11} />
          {unfolded
            ? t("Fold to the first {0}", formatNumber(TAPE_HEAD))
            : t("Show all {0, plural, one {# segment} other {# segments}}", rows.length)}
        </Button>
      )}
    </div>
  );
}
