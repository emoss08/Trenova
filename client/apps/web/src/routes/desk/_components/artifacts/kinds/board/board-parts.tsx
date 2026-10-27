import { AnalysisTableView } from "@/components/assistant/analysis-result";
import { AnalysisChartSkeleton } from "@/components/assistant/analysis-chart-skeleton";
import { isRecordEntityType, recordPath } from "@/config/record-links";
import type {
  DeskBoardBlock,
  DeskBoardCheck,
  DeskBoardCompare,
  DeskBoardEvent,
  DeskBoardFigure,
  DeskBoardMeter,
  DeskBoardRow,
  DeskBoardSegment,
} from "@/lib/graphql/desk-board-card";
import { lazyComponent } from "@/lib/lazy-component";
import type { AnalysisChart } from "@/types/analysis";
import { LazyComponent } from "@trenova/shared/components/error-boundary";
import {
  CarrierComplianceStatusBadge,
  PlainInvoiceStatusBadge,
  ShipmentStatusBadge,
  StatusBadge,
} from "@trenova/shared/components/status-badge";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { shipmentStatusSchema } from "@trenova/shared/types/shipment";
import type { CSSProperties, ReactNode } from "react";
import { Link } from "react-router";
import { ArtIcon } from "../../desk-art-kinds";
import { CardIcon } from "../../record-card/record-card";
import { activityActor, activitySentence } from "../../record-card/record-activity";
import { cardMomentAfter } from "../../record-card/record-card-values";
import {
  boardTone,
  checkGroups,
  formatBoardValue,
  furthestOf,
  initialsOf,
  measureIsTrack,
  measureReading,
  noteLabel,
  sourceLine,
  toneClass,
  unreadableRecord,
} from "./board-card";

const boardChart = lazyComponent(
  () => import("@/components/assistant/analysis-chart"),
  (module) => module.AnalysisChartView,
);

/** Opens a record a board names in the board's place, in the order its block lists them. */
export type OpenRecord = (row: DeskBoardRow, order: readonly string[]) => void;

/** Whether a record of a kind opens in place here; a kind that does not opens on its page. */
export type OpensInPlace = (resource: string) => boolean;

const INVOICE_STATUSES = new Set(["Draft", "Posted", "Voided"]);
const COMPLIANCE_STATUSES = new Set(["Pending", "Qualified", "Disqualified", "Expired"]);

/** A record's status, drawn with the badge its own pages use. */
export function BoardStatus({ resource, status }: { resource: string; status: string }) {
  if (status === "") return null;
  if (resource === "shipment") {
    const parsed = shipmentStatusSchema.safeParse(status);
    if (parsed.success) return <ShipmentStatusBadge status={parsed.data} />;
  }
  if (resource === "invoice" && INVOICE_STATUSES.has(status)) {
    return <PlainInvoiceStatusBadge status={status as "Draft" | "Posted" | "Voided"} />;
  }
  if (resource === "carrier" && COMPLIANCE_STATUSES.has(status)) {
    return (
      <CarrierComplianceStatusBadge
        status={status as "Pending" | "Qualified" | "Disqualified" | "Expired"}
      />
    );
  }
  return <StatusBadge status={status} />;
}

/** An exhibit's numeral: a circled figure the verdict cites it by. */
export function Numeral({ n, struck = false }: { n: number; struck?: boolean }) {
  return <span className={cn("dk-bd-n", struck && "dk-off")}>{n}</span>;
}

/** The answer's words, with the phrases it rests on marked and citing their exhibit. */
export function Segments({ segments }: { segments: readonly DeskBoardSegment[] }) {
  return segments.map((segment, index) =>
    segment.numeral > 0 ? (
      // oxlint-disable-next-line react/no-array-index-key -- segments are a fixed run of the text
      <span key={index}>
        <span className={cn("dk-bd-hl", segment.tone === "warn" && "dk-warn")}>{segment.text}</span>
        <Numeral n={segment.numeral} struck={segment.struck} />
      </span>
    ) : (
      // oxlint-disable-next-line react/no-array-index-key -- segments are a fixed run of the text
      <span key={index}>{segment.text}</span>
    ),
  );
}

/** A record named in the words or in a check: its label, opening the record. */
export function RecordChip({
  row,
  order,
  open,
  opensInPlace,
}: {
  row: DeskBoardRow;
  order: readonly string[];
  open: OpenRecord;
  opensInPlace: OpensInPlace;
}) {
  if (!row.readable) return null;
  if (opensInPlace(row.resource)) {
    return (
      <button type="button" className="dk-bd-ref" onClick={() => open(row, order)}>
        {row.label}
      </button>
    );
  }
  if (isRecordEntityType(row.resource)) {
    return (
      <Link className="dk-bd-ref" to={recordPath(row.resource, row.id)}>
        {row.label}
      </Link>
    );
  }
  return <span className="dk-bd-ref">{row.label}</span>;
}

export function RecordChips(props: {
  rows: readonly DeskBoardRow[];
  open: OpenRecord;
  opensInPlace: OpensInPlace;
}) {
  const readable = props.rows.filter((row) => row.readable);
  if (readable.length === 0) return null;
  const order = readable.map((row) => row.id);
  return (
    <div className="dk-bd-chips">
      {readable.map((row) => (
        <RecordChip
          key={row.id}
          row={row}
          order={order}
          open={props.open}
          opensInPlace={props.opensInPlace}
        />
      ))}
    </div>
  );
}

/** An exhibit: printed by Trenova on a tab that names what it shows, with where it read it from. */
export function Exhibit({
  block,
  stamp,
  half,
  children,
}: {
  block: DeskBoardBlock;
  /** When it was read: "live · 9:15 AM". */
  stamp: string;
  half: boolean;
  children: ReactNode;
}) {
  const t = useT();
  const unreadable = block.state === "unreadable";
  const line = sourceLine(block.source, t);
  return (
    <section
      className={cn("dk-bd-ex", half && "dk-bd-half", unreadable && "dk-unread")}
      aria-label={block.title || t("Exhibit {0}", block.numeral)}
    >
      <div className="dk-bd-tab">
        <Numeral n={block.numeral} struck={unreadable} />
        <span>{block.title || t("Exhibit {0}", block.numeral)}</span>
      </div>
      {!unreadable && (line !== "" || stamp !== "") && (
        <div className="dk-bd-src">
          <span>{line}</span>
          <span>{stamp}</span>
        </div>
      )}
      {children}
    </section>
  );
}

/** A block whose source could not be read: what the reader can still trust. */
export function Unreadable({ block }: { block: DeskBoardBlock }) {
  const t = useT();
  const kind = block.source.kind;
  const title =
    kind === "analysis"
      ? t("The analysis this came from could not be read")
      : kind === "table"
        ? t("The table this came from could not be read")
        : kind === "metrics"
          ? t("These figures could not be read")
          : kind === "activity"
            ? t("This record's history could not be read")
            : t("These records could not be read");
  return (
    <div className="dk-bd-unread">
      <span>
        <ArtIcon name="lock" size={13} stroke={2} />
      </span>
      <span>
        <b>{title}</b>
        {t(
          "It may have been removed, or you may not be allowed to see it. The rest of the board stands.",
        )}
      </span>
    </div>
  );
}

/** A block with nothing in it now. */
export function Empty({ block }: { block: DeskBoardBlock }) {
  const t = useT();
  const text =
    block.type === "timeline"
      ? t("Nothing has been done to this record yet.")
      : block.type === "chart"
        ? t("The chart has nothing to draw.")
        : t("The table has no rows now.");
  return (
    <div className="dk-bd-empty">
      <span>
        <ArtIcon name="check" size={14} stroke={2.4} />
      </span>
      <span>{text}</span>
    </div>
  );
}

/** Figures, read now; one that moved since the answer says what it was. */
export function Readout({
  figures,
  answeredAt,
  now,
}: {
  figures: DeskBoardFigure[];
  answeredAt: number;
  now: number;
}) {
  const t = useT();
  return (
    <dl className="dk-bd-read" style={{ "--n": figures.length } as CSSProperties}>
      {figures.map((figure, index) => {
        const tone = toneClass(figure.tone);
        return (
          // oxlint-disable-next-line react/no-array-index-key -- figures are a fixed run of the block
          <div key={index} className={tone}>
            <dt>
              {tone !== "" && <i />}
              {figure.label}
            </dt>
            <dd>
              {figure.value === null || figure.value === undefined ? (
                <span className="dk-bd-gone">{t("Could not be read")}</span>
              ) : (
                <>
                  {figure.was && (
                    <>
                      <s>{formatBoardValue(figure.was)}</s>
                      <span aria-hidden="true">→</span>
                    </>
                  )}
                  {formatBoardValue(figure.value)}
                </>
              )}
            </dd>
            {figure.was && (
              <em className="dk-moved">
                {t("since the answer at {0}", cardMomentAfter(now, answeredAt))}
              </em>
            )}
          </div>
        );
      })}
    </dl>
  );
}

/** A record the reader may not read, drawn where it stands in the list. */
function GoneRow({ resource }: { resource: string }) {
  const t = useT();
  return (
    <div className="dk-bd-row dk-gone">
      <span className="dk-bd-lock">
        <ArtIcon name="lock" size={12} stroke={2} />
      </span>
      <span className="dk-bd-who">
        <b>{unreadableRecord(resource, t)}</b>
        <span>{t("It may have been removed, or you may not be allowed to see it.")}</span>
      </span>
    </div>
  );
}

function RowBody({ row, furthest, now }: { row: DeskBoardRow; furthest: number; now: number }) {
  const t = useT();
  const reading = row.measure ? measureReading(row.measure, furthest, now, t) : null;
  const track = row.measure && measureIsTrack(row.measure.measure) && reading;
  return (
    <>
      <span className="dk-bd-who">
        <b>{row.label}</b>
        {row.subtitle !== "" && <span>{row.subtitle}</span>}
      </span>
      {track ? (
        <span className={cn("dk-bd-behind", toneClass(reading.tone))}>
          <span>
            <span>{reading.against}</span>
            <b>{reading.standing}</b>
          </span>
          <span className="dk-bd-track">
            <i
              className={cn(reading.held && "dk-bd-held")}
              style={{ width: `${Math.round(reading.fraction * 100)}%` }}
            />
          </span>
        </span>
      ) : reading ? (
        <span className={cn("dk-bd-amt", toneClass(reading.tone))}>
          {reading.standing}
          {reading.against !== "" && <span>{reading.against}</span>}
        </span>
      ) : (
        <span />
      )}
      <BoardStatus resource={row.resource} status={row.status} />
    </>
  );
}

/** A block's records, each a row that opens the record, measured against the furthest. */
export function RecordRows({
  rows,
  now,
  open,
  opensInPlace,
}: {
  rows: DeskBoardRow[];
  now: number;
  open: OpenRecord;
  opensInPlace: OpensInPlace;
}) {
  const furthest = furthestOf(rows);
  const order = rows.filter((row) => row.readable).map((row) => row.id);
  return (
    <div className="dk-bd-rows">
      {rows.map((row, index) => {
        if (!row.readable) {
          // oxlint-disable-next-line react/no-array-index-key -- an unread row has only its place
          return <GoneRow key={`${row.id}-${index}`} resource={row.resource} />;
        }
        const body = <RowBody row={row} furthest={furthest} now={now} />;
        if (opensInPlace(row.resource)) {
          return (
            <button
              key={row.id}
              type="button"
              className="dk-bd-row"
              onClick={() => open(row, order)}
            >
              {body}
            </button>
          );
        }
        return isRecordEntityType(row.resource) ? (
          <Link key={row.id} className="dk-bd-row" to={recordPath(row.resource, row.id)}>
            {body}
          </Link>
        ) : (
          <div key={row.id} className="dk-bd-row">
            {body}
          </div>
        );
      })}
    </div>
  );
}

/** A record's history, entry by entry, oldest first. */
export function LogBook({ events, now }: { events: DeskBoardEvent[]; now: number }) {
  const t = useT();
  return (
    <ol className="dk-bd-log">
      {events.map((event, index) => {
        const operation = event.operation.toLowerCase();
        const tone =
          operation === "delete" || operation === "cancel" || operation === "reject"
            ? "dk-bad"
            : "";
        return (
          // oxlint-disable-next-line react/no-array-index-key -- the log is a fixed run of entries
          <li key={index} className={tone}>
            <time dateTime={new Date(event.at * 1000).toISOString()}>
              {cardMomentAfter(now, event.at)}
            </time>
            <i />
            <div>
              {event.comment !== "" ? event.comment : activitySentence(event, t)}
              <em>
                {event.actor !== ""
                  ? event.actor
                  : activityActor({ user: null, principalType: event.principalType }, t)}
              </em>
            </div>
          </li>
        );
      })}
    </ol>
  );
}

/** A compared record at the head of its column, opening the record. */
function Face({
  column,
  order,
  open,
  opensInPlace,
}: {
  column: DeskBoardRow;
  order: readonly string[];
  open: OpenRecord;
  opensInPlace: OpensInPlace;
}) {
  const body = (
    <>
      <span className="dk-bd-mono">{initialsOf(column.label)}</span>
      <b>{column.label}</b>
      {column.subtitle !== "" && <small>{column.subtitle}</small>}
      <BoardStatus resource={column.resource} status={column.status} />
    </>
  );
  if (opensInPlace(column.resource)) {
    return (
      <button type="button" className="dk-bd-face" onClick={() => open(column, order)}>
        {body}
      </button>
    );
  }
  if (isRecordEntityType(column.resource)) {
    return (
      <Link className="dk-bd-face" to={recordPath(column.resource, column.id)}>
        {body}
      </Link>
    );
  }
  return <span className="dk-bd-face dk-still">{body}</span>;
}

/** Records side by side: each a column under its own face, each row's best marked. */
export function HeadToHead({
  compare,
  open,
  opensInPlace,
}: {
  compare: DeskBoardCompare;
  open: OpenRecord;
  opensInPlace: OpensInPlace;
}) {
  const t = useT();
  const order = compare.columns.filter((column) => column.readable).map((column) => column.id);
  const lines = compare.lines.length;
  return (
    <table className="dk-bd-h2h">
      <colgroup>
        <col className="dk-lab" />
        {compare.columns.map((column) => (
          <col key={column.id} />
        ))}
      </colgroup>
      <thead>
        <tr>
          <th scope="col">{t("Measure")}</th>
          {compare.columns.map((column) => (
            <th key={column.id} scope="col">
              {column.readable ? (
                <Face column={column} order={order} open={open} opensInPlace={opensInPlace} />
              ) : (
                <span className="dk-bd-face dk-gone">
                  <span className="dk-bd-mono">
                    <ArtIcon name="lock" size={12} stroke={2} />
                  </span>
                  <b>{unreadableRecord(column.resource, t)}</b>
                </span>
              )}
            </th>
          ))}
        </tr>
      </thead>
      <tbody>
        {compare.lines.map((line, index) => (
          // oxlint-disable-next-line react/no-array-index-key -- the comparison's rows are fixed
          <tr key={index}>
            <th scope="row">
              {line.label}
              {line.better === "lower" && <span>{t("lower is better")}</span>}
            </th>
            {line.cells.map((cell, cellIndex) => {
              const win = line.wins[cellIndex] === true;
              const text = cell ? formatBoardValue(cell) : "—";
              return (
                // oxlint-disable-next-line react/no-array-index-key -- one cell per column
                <td key={cellIndex} className={cn(win && "dk-win")}>
                  {win ? (
                    <span>
                      <CardIcon name="check" size={10} stroke={2.6} />
                      {text}
                    </span>
                  ) : (
                    text
                  )}
                </td>
              );
            })}
          </tr>
        ))}
      </tbody>
      {compare.wins.length > 0 && lines > 0 && (
        <tfoot>
          <tr>
            <th scope="row">{t("Rows won")}</th>
            {compare.wins.map((wins, index) => (
              // oxlint-disable-next-line react/no-array-index-key -- one tally per column
              <td key={index}>
                <span
                  className="dk-bd-wins"
                  role="img"
                  aria-label={t("{0} of {1} rows won", wins, lines)}
                >
                  {Array.from({ length: lines }, (_, pip) => (
                    // oxlint-disable-next-line react/no-array-index-key -- pips are positional
                    <i key={pip} className={cn(pip < wins && "dk-on")} />
                  ))}
                </span>
              </td>
            ))}
          </tr>
        </tfoot>
      )}
    </table>
  );
}

const CHECK_ICONS = { bad: "x", warn: "warn", info: "info", ok: "check" } as const;

/** Checks grouped by how much they ask, each with its count and the records it counts. */
export function Checks({
  checks,
  open,
  opensInPlace,
}: {
  checks: DeskBoardCheck[];
  open: OpenRecord;
  opensInPlace: OpensInPlace;
}) {
  const t = useT();
  return (
    <div className="dk-bd-checks">
      {checkGroups(checks, t).map((group) => (
        <div key={group.tone} className="contents">
          <p>{group.label}</p>
          {group.checks.map((check, index) => {
            const tone = boardTone(check.tone) || "info";
            return (
              // oxlint-disable-next-line react/no-array-index-key -- a group's checks are fixed
              <div key={index} className={cn("dk-bd-ck", toneClass(tone))}>
                <span className="dk-rec-ci">
                  <CardIcon
                    name={CHECK_ICONS[tone]}
                    size={tone === "warn" ? 12 : 11}
                    stroke={tone === "warn" ? 2 : 2.8}
                  />
                </span>
                <b>{check.title}</b>
                <span className="dk-bd-count">
                  {check.count ? formatBoardValue(check.count) : t("unread")}
                  {check.was && <s>{formatBoardValue(check.was)}</s>}
                </span>
                {check.detail !== "" && <span className="dk-bd-ck-d">{check.detail}</span>}
                {check.hidden > 0 && (
                  <span className="dk-bd-ck-d">
                    {t(
                      "{0, plural, one {# more you may not read} other {# more you may not read}}",
                      check.hidden,
                    )}
                  </span>
                )}
                <RecordChips rows={check.records} open={open} opensInPlace={opensInPlace} />
              </div>
            );
          })}
        </div>
      ))}
    </div>
  );
}

/** One pip for every check, clear first, so how much is done reads at a glance. */
export function Pips({ meter }: { meter: DeskBoardMeter }) {
  const t = useT();
  if (meter.percent) {
    const fraction = Math.min(1, Math.max(0, Number(meter.percent.number) || 0));
    return (
      <div className="dk-bd-pct">
        <b>{formatBoardValue(meter.percent)}</b>
        <span
          className="dk-bd-bar"
          role="meter"
          aria-valuemin={0}
          aria-valuemax={100}
          aria-valuenow={Math.round(fraction * 100)}
        >
          <i style={{ width: `${Math.round(fraction * 100)}%` }} />
        </span>
      </div>
    );
  }
  const counts = { bad: 0, warn: 0, info: 0 };
  for (const tone of meter.tones) {
    if (tone === "bad" || tone === "warn" || tone === "info") counts[tone] += 1;
  }
  const share = meter.total > 0 ? Math.round((meter.clear / meter.total) * 100) : 0;
  return (
    <>
      <div className="dk-bd-pips-h">
        <b>
          {meter.clear}
          <small>
            {t("of {0, plural, one {# check clear} other {# checks clear}}", meter.total)}
          </small>
        </b>
        <span className="dk-bd-src">{t("{0}% clear", share)}</span>
      </div>
      <div
        className="dk-bd-pips"
        style={{ "--n": meter.total } as CSSProperties}
        role="meter"
        aria-label={t("Checks clear")}
        aria-valuemin={0}
        aria-valuemax={meter.total}
        aria-valuenow={meter.clear}
      >
        {meter.tones.map((tone, index) => (
          // oxlint-disable-next-line react/no-array-index-key -- pips are positional
          <i key={index} className={cn(tone !== "ok" && toneClass(tone))} />
        ))}
      </div>
      <div className="dk-bd-pips-key">
        <span>
          <i />
          {t("Clear {0}", meter.clear)}
        </span>
        {counts.info > 0 && (
          <span>
            <i className="dk-info" />
            {t("Worth knowing {0}", counts.info)}
          </span>
        )}
        {counts.warn > 0 && (
          <span>
            <i className="dk-warn" />
            {t("Needs a look {0}", counts.warn)}
          </span>
        )}
        {counts.bad > 0 && (
          <span>
            <i className="dk-bad" />
            {t("Blocks it {0}", counts.bad)}
          </span>
        )}
      </div>
    </>
  );
}

/** An analysis chart, drawn as the analysis card draws it. */
export function BoardChart({ chart }: { chart: AnalysisChart }) {
  const t = useT();
  return (
    <LazyComponent fallback={<AnalysisChartSkeleton title={chart.title || t("Chart")} />}>
      <boardChart.Component chart={chart} />
    </LazyComponent>
  );
}

export { AnalysisTableView as BoardTable };

/** A note: the agent's own words on a slip, with the records it names. */
export function Note({
  block,
  half,
  open,
  opensInPlace,
}: {
  block: DeskBoardBlock;
  half: boolean;
  open: OpenRecord;
  opensInPlace: OpensInPlace;
}) {
  const t = useT();
  const tone = boardTone(block.noteTone) || "info";
  return (
    <aside className={cn("dk-bd-note", toneClass(tone), half && "dk-bd-half")}>
      <div className="dk-bd-note-h">
        <CardIcon name={tone === "bad" ? "x" : tone === "warn" ? "warn" : "check"} size={11} />
        {noteLabel(tone, t)}
      </div>
      <b>{block.title}</b>
      <p>
        <Segments segments={block.text} />
      </p>
      <RecordChips rows={block.chips} open={open} opensInPlace={opensInPlace} />
    </aside>
  );
}
