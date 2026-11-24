import { Link } from "@trenova/shared/components/link";
import { AnalysisTableView } from "@/components/assistant/analysis-result";
import { isRecordEntityType, recordPath } from "@/config/record-links";
import type {
  DeskBoardBlock,
  DeskBoardCheck,
  DeskBoardCompare,
  DeskBoardEvent,
  DeskBoardFigure,
  DeskBoardMeter,
  DeskBoardRow,
  DeskBoardRowMeasure,
} from "@/lib/graphql/desk-board-card";
import { Badge, type BadgeTone } from "@trenova/shared/components/ui/badge";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import type { CSSProperties, ReactNode } from "react";
import { BoardWords, Numeral, type BoardReadings } from "./board-words";
import { ArtIcon } from "../../desk-art-kinds";
import { CardIcon } from "../../record-card/record-card";
import { activityActor, activitySentence } from "../../record-card/record-activity";
import { cardDayAfter, cardMomentAfter } from "../../record-card/record-card-values";
import { SegmentBar } from "./board-bar";
import {
  type BoardTone,
  blockSource,
  boardTone,
  checkGroups,
  formatBoardValue,
  figureRows,
  initialsOf,
  noteLabel,
  opened,
  toneClass,
  unreadableRecord,
  readableIds,
} from "./board-card";
import { fillWords, wholeWords } from "./board-words-fill";

/** Opens a record a board names in the board's place, in the order its block lists them. */
export type OpenRecord = (row: DeskBoardRow, order: readonly string[]) => void;

/** Whether a record of a kind opens in place here; a kind that does not opens on its page. */
export type OpensInPlace = (kind: string) => boolean;

const STATUS_VARIANT: Record<BoardTone, BadgeTone> = {
  bad: "danger",
  warn: "warning",
  info: "info",
  ok: "success",
  "": "neutral",
};

/**
 * A record's status as the product words it, drawn with the shared badge in the
 * tone the status's owner gave it on the server; the client keeps no list of
 * status names.
 */
export function BoardStatus({ status, tone }: { status: string; tone: string }) {
  if (status === "") return null;
  return (
    <Badge variant={STATUS_VARIANT[boardTone(tone)]} dot className="max-h-5">
      {status}
    </Badge>
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
  const target = opened(row);
  if (opensInPlace(target.kind)) {
    return (
      <button type="button" className="dk-bd-ref" onClick={() => open(row, order)}>
        {row.label}
      </button>
    );
  }
  if (isRecordEntityType(target.kind)) {
    return (
      <Link className="dk-bd-ref" to={recordPath(target.kind, target.id)}>
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
  const order = readableIds(props.rows);
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

/** An exhibit's caption under its tab, drawn and sketched alike; its last line never a lone word. */
export const EXHIBIT_CAPTION_CLASS =
  "-mt-0.5 mb-2.5 text-dsk-base leading-normal text-pretty text-dsk-subtle";

/**
 * An exhibit: printed by Trenova on a tab that names what it shows, the line
 * the agent wrote under it when it wrote one, and where it read it from.
 */
export function Exhibit({
  block,
  board,
  stamp,
  half,
  lit = false,
  className,
  style,
  children,
}: {
  block: DeskBoardBlock;
  /** The board's readings, for a caption that places one. */
  board: BoardReadings;
  /** When it was read: "live · 9:15 AM". */
  stamp: string;
  half: boolean;
  /** A phrase citing it is pointed at or focused. */
  lit?: boolean;
  className?: string;
  style?: CSSProperties;
  children: ReactNode;
}) {
  const t = useT();
  const unreadable = block.state === "unreadable";
  const line = blockSource(block, t);
  return (
    <section
      className={cn(
        "dk-bd-ex",
        half && "dk-bd-half",
        unreadable && "dk-unread",
        lit && "dk-bd-lit",
        className,
      )}
      style={style}
      aria-label={block.title || t("Exhibit {0}", block.numeral)}
    >
      <div className="dk-bd-tab">
        <Numeral n={block.numeral} struck={unreadable} />
        <span>{block.title || t("Exhibit {0}", block.numeral)}</span>
      </div>
      {block.caption.length > 0 && (
        <p className={EXHIBIT_CAPTION_CLASS}>
          <BoardWords segments={block.caption} board={board} />
        </p>
      )}
      {!unreadable && (line !== "" || stamp !== "") && <SourceLine line={line} stamp={stamp} />}
      {children}
    </section>
  );
}

/**
 * Where an exhibit was read from and when, in one line under its tab. Its
 * skeleton draws the same line with a blank in each place, so the line is the
 * same height before the exhibit is read as after.
 */
export function SourceLine({ line, stamp }: { line: ReactNode; stamp: ReactNode }) {
  return (
    <div className="dk-bd-src">
      <span title={typeof line === "string" ? line : undefined}>{line}</span>
      <span>{stamp}</span>
    </div>
  );
}

/**
 * A blank set in a line of words while they are read: a bar on the line, with
 * a zero-width character beside it so the line keeps the height of its font,
 * as the words that replace it do.
 */
export function LineBlank({ className }: { className?: string }) {
  return (
    <>
      <i className={cn("dk-bd-wait-line ui-shimmer", className)} aria-hidden="true" />
      {"\u200b"}
    </>
  );
}

/**
 * A record's row while the list is read, in the row's own box and lines: its
 * number and the line under it, each a blank in its own font, and in a list
 * read by a track the track's place, so a row read is the height of the row
 * drawn. The row's lines are fixed heights (`.dk-bd-row`), so the blanks cannot
 * make it taller or shorter than a drawn row.
 */
export function RecordRowSketch({ track = false }: { track?: boolean }) {
  return (
    <div className="dk-bd-row">
      <span className="dk-bd-l1">
        <b className="dk-bd-id">
          <LineBlank className="w-28" />
        </b>
      </span>
      <span className="dk-bd-line">
        <span className="dk-bd-sub">
          <LineBlank className="w-40" />
        </span>
      </span>
      {track && <span className="dk-bd-track ui-shimmer" aria-hidden="true" />}
    </div>
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

/**
 * Figures, read now; one that moved since the answer says what it was. More
 * than four are set in balanced rows (figureRows), each its own run.
 */
/**
 * A figure as a readout draws it: a metric's figure may carry what it read
 * earlier and when (`wasAt`), which is said as that day, not as a move since the
 * answer.
 */
export type ReadoutFigure = DeskBoardFigure & { wasAt?: number | null };

export function Readout({
  figures,
  answeredAt,
  now,
}: {
  figures: ReadoutFigure[];
  answeredAt: number;
  now: number;
}) {
  const rows = figureRows(figures.length);
  if (rows.length <= 1) {
    return <ReadoutRow figures={figures} answeredAt={answeredAt} now={now} />;
  }

  const starts = rows.map((_, index) => rows.slice(0, index).reduce((sum, n) => sum + n, 0));
  return (
    <div className="dk-bd-reads">
      {rows.map((size, index) => (
        <ReadoutRow
          key={starts[index]}
          figures={figures.slice(starts[index], starts[index] + size)}
          answeredAt={answeredAt}
          now={now}
        />
      ))}
    </div>
  );
}

function ReadoutRow({
  figures,
  answeredAt,
  now,
}: {
  figures: ReadoutFigure[];
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
                  {figure.was && !figure.wasAt && (
                    <>
                      <s>{formatBoardValue(figure.was)}</s>
                      <span aria-hidden="true">→</span>
                    </>
                  )}
                  <span>{formatBoardValue(figure.value)}</span>
                </>
              )}
            </dd>
            {figure.was && figure.wasAt ? (
              <em>
                {t("was {0} on {1}", formatBoardValue(figure.was), cardDayAfter(now, figure.wasAt))}
              </em>
            ) : (
              figure.was && (
                <em className="dk-moved">
                  {t("since the answer at {0}", cardMomentAfter(now, answeredAt))}
                </em>
              )
            )}
          </div>
        );
      })}
    </dl>
  );
}

/** A record the reader may not read, drawn where it stands in the list. */
function GoneRow({ kind }: { kind: string }) {
  const t = useT();
  const why = t("It may have been removed, or you may not be allowed to see it.");
  return (
    <div className="dk-bd-row dk-gone">
      <span className="dk-bd-lock">
        <ArtIcon name="lock" size={12} stroke={2} />
      </span>
      <span className="dk-bd-l1">
        <b className="dk-bd-id">{unreadableRecord(kind, t)}</b>
      </span>
      <span className="dk-bd-line">
        <span className="dk-bd-sub" title={why}>
          {why}
        </span>
      </span>
    </div>
  );
}

/**
 * A row's second fact as siblings on its second line: the words with their moment
 * (`dk-bd-at`), which never give way, then the place it was measured at (`dk-bd-place`),
 * which gives way first. Each carries the whole fact in `title`.
 */
function SecondFact({ measure, now }: { measure: DeskBoardRowMeasure; now: number }) {
  const fact = measure.fact;
  if (!fact) return null;
  const shown = fillWords(fact, now);
  if (shown === "") return null;
  const whole = wholeWords(fact, now);
  // Words give way with an ellipsis; a moment, amount, duration or number in the {0} slot never
  // does. A slot holding text (value kind "text") is words like the rest.
  const [before, ...rest] = fact.text.split("{0}");
  const slotted = rest.length > 0 && fact.value?.kind !== "text";
  const after = rest.join("{0}");
  return (
    <>
      <span className={cn("dk-bd-at", !slotted && "dk-bd-said")} title={whole}>
        {slotted ? (
          <>
            {before !== "" && <span className="dk-bd-words">{before}</span>}
            <span className="dk-bd-slot">{fillWords({ ...fact, text: "{0}" }, now)}</span>
            {after !== "" && <span className="dk-bd-words">{after}</span>}
          </>
        ) : (
          <span className="dk-bd-words">{shown}</span>
        )}
      </span>
      {fact.place !== "" && (
        <span className="dk-bd-place" title={whole}>
          {` ${fact.place}`}
        </span>
      )}
    </>
  );
}

/**
 * A record's row, one anatomy at every width (`.dk-bd-row` in desk-v2.css), each line its own
 * box so neither narrows the other: the label and the measure's standing words (`dk-bd-l1`);
 * the subtitle and the measure's second fact as one phrase, then the status (`dk-bd-line`);
 * the bar the server laid out under them. A measure the reader may not read says so where the
 * standing would be.
 */
function RowBody({ row, now }: { row: DeskBoardRow; now: number }) {
  const t = useT();
  const measure = row.measure;
  const standing = measure
    ? measure.value
      ? fillWords(measure.standing, now)
      : t("Not yours to read")
    : "";
  return (
    <>
      <span className="dk-bd-l1">
        <b className="dk-bd-id">{row.label}</b>
        {standing !== "" && (
          <span className={cn("dk-bd-val", toneClass(measure?.tone ?? ""))}>{standing}</span>
        )}
      </span>
      <span className="dk-bd-line">
        {row.subtitle !== "" && (
          <span className="dk-bd-sub" title={row.subtitle}>
            {row.subtitle}
          </span>
        )}
        {measure && <SecondFact measure={measure} now={now} />}
        {row.status !== "" && (
          <span className="dk-bd-st">
            <BoardStatus status={row.status} tone={row.statusTone} />
          </span>
        )}
      </span>
      {measure?.bar && <SegmentBar bar={measure.bar} />}
    </>
  );
}

/** A block's records, each a row that opens the record, measured against the furthest. */
export function RecordRows({
  rows,
  now,
  open,
  opensInPlace,
  more,
  order = readableIds(rows),
}: {
  rows: DeskBoardRow[];
  /** The records a record opened from here steps through; the rows drawn unless the list was cut. */
  order?: readonly string[];
  now: number;
  open: OpenRecord;
  opensInPlace: OpensInPlace;
  /** The line after the rows saying how many more the board holds, when the list was cut. */
  more?: ReactNode;
}) {
  return (
    <div className="dk-bd-rows">
      {rows.map((row, index) => {
        if (!row.readable) {
          // oxlint-disable-next-line react/no-array-index-key -- an unread row has only its place
          return <GoneRow key={`${row.id}-${index}`} kind={row.kind} />;
        }
        const body = <RowBody row={row} now={now} />;
        const target = opened(row);
        if (opensInPlace(target.kind)) {
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
        return isRecordEntityType(target.kind) ? (
          <Link key={row.id} className="dk-bd-row" to={recordPath(target.kind, target.id)}>
            {body}
          </Link>
        ) : (
          <div key={row.id} className="dk-bd-row">
            {body}
          </div>
        );
      })}
      {more}
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
      <b title={column.label}>{column.label}</b>
      {column.subtitle !== "" && <small title={column.subtitle}>{column.subtitle}</small>}
      <BoardStatus status={column.status} tone={column.statusTone} />
    </>
  );
  const target = opened(column);
  if (opensInPlace(target.kind)) {
    return (
      <button type="button" className="dk-bd-face" onClick={() => open(column, order)}>
        {body}
      </button>
    );
  }
  if (isRecordEntityType(target.kind)) {
    return (
      <Link className="dk-bd-face" to={recordPath(target.kind, target.id)}>
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
  const order = readableIds(compare.columns);
  const lines = compare.lines.length;
  return (
    <table className="dk-bd-h2h" style={{ "--cols": compare.columns.length } as CSSProperties}>
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
                  <b>{unreadableRecord(column.kind, t)}</b>
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
                {(check.count || check.was || check.hidden > 0) && (
                  <span className="dk-bd-count">
                    {check.count ? formatBoardValue(check.count) : t("unread")}
                    {check.was && <s>{formatBoardValue(check.was)}</s>}
                  </span>
                )}
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
          {t("All clear {0}", meter.clear)}
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

export { AnalysisTableView as BoardTable };

/** A note: the agent's own words on a slip, with the records it names. */
export function Note({
  block,
  board,
  half,
  open,
  opensInPlace,
  className,
  style,
}: {
  block: DeskBoardBlock;
  board: BoardReadings;
  half: boolean;
  open: OpenRecord;
  opensInPlace: OpensInPlace;
  className?: string;
  style?: CSSProperties;
}) {
  const t = useT();
  const tone = boardTone(block.noteTone) || "info";
  return (
    <aside
      className={cn("dk-bd-note", toneClass(tone), half && "dk-bd-half", className)}
      style={style}
    >
      <div className="dk-bd-note-h">
        <CardIcon name={tone === "bad" ? "x" : tone === "warn" ? "warn" : "check"} size={11} />
        {noteLabel(tone, t)}
      </div>
      <b>{block.title}</b>
      <p>
        <BoardWords segments={block.text} board={board} />
      </p>
      <RecordChips rows={block.chips} open={open} opensInPlace={opensInPlace} />
    </aside>
  );
}
