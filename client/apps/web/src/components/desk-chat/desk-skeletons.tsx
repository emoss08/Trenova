import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import type { CSSProperties } from "react";

/**
 * The Desk's loading shapes, drawn in the Desk's own palette (`.dk-sk`) so a
 * theme the person chose holds while content arrives. Each is the outline of
 * what replaces it, at its size, so nothing moves when the content lands.
 */

/** The shape an artifact's contents take while they are read. */
export type DeskArtifactShape = "table" | "document" | "record" | "lines";

function width(percent: number): CSSProperties {
  return { width: `${percent}%` };
}

const QUESTION_WIDTHS = [62, 44] as const;
const REPLY_LINES = [
  [96, 92, 98, 64],
  [94, 88, 52],
] as const;

/**
 * A conversation while its messages are first read: two turns, each a
 * question over the agent's reply, laid out on the transcript's own grid.
 */
export function DeskTranscriptSkeleton() {
  const t = useT();

  return (
    <>
      <span role="status" className="sr-only">
        {t("Reading the conversation…")}
      </span>
      {QUESTION_WIDTHS.map((questionWidth, turn) => (
        // oxlint-disable-next-line react/no-array-index-key -- a fixed outline that never reorders
        <div key={turn} className={cn("dk-r dk-skt", turn === 0 && "dk-first")} aria-hidden>
          <div className="dk-g">
            <i className="dk-sk dk-sk-time" />
          </div>
          <div className="dk-c">
            <i className="dk-sk dk-sk-q" style={width(questionWidth)} />
            <i className="dk-sk dk-sk-who" />
            {REPLY_LINES[turn].map((lineWidth, line) => (
              // oxlint-disable-next-line react/no-array-index-key -- a fixed outline that never reorders
              <i key={line} className="dk-sk dk-sk-ln" style={width(lineWidth)} />
            ))}
          </div>
          <div className="dk-m" />
        </div>
      ))}
    </>
  );
}

const TABLE_COLUMNS = [34, 22, 26, 18] as const;
const TABLE_ROWS = 8;

function TableShape() {
  return (
    <div className="dk-skb-tb">
      <div className="dk-skb-tr dk-skb-th">
        {TABLE_COLUMNS.map((column, index) => (
          // oxlint-disable-next-line react/no-array-index-key -- a fixed outline that never reorders
          <i key={index} className="dk-sk" style={width(column * 0.6)} />
        ))}
      </div>
      {Array.from({ length: TABLE_ROWS }, (_, row) => (
        // oxlint-disable-next-line react/no-array-index-key -- a fixed outline that never reorders
        <div key={row} className="dk-skb-tr">
          {TABLE_COLUMNS.map((column, index) => (
            <i
              // oxlint-disable-next-line react/no-array-index-key -- a fixed outline that never reorders
              key={index}
              className="dk-sk"
              style={width(column * (0.7 + ((row + index) % 3) * 0.1))}
            />
          ))}
        </div>
      ))}
    </div>
  );
}

const DOCUMENT_PARAGRAPHS = [
  [98, 94, 97, 58],
  [96, 91, 72],
  [95, 98, 89, 40],
] as const;

function DocumentShape() {
  return (
    <div className="dk-skb-doc">
      <i className="dk-sk dk-skb-title" style={width(54)} />
      {DOCUMENT_PARAGRAPHS.map((lines, paragraph) => (
        // oxlint-disable-next-line react/no-array-index-key -- a fixed outline that never reorders
        <div key={paragraph} className="dk-skb-para">
          {lines.map((lineWidth, line) => (
            // oxlint-disable-next-line react/no-array-index-key -- a fixed outline that never reorders
            <i key={line} className="dk-sk dk-skb-ln" style={width(lineWidth)} />
          ))}
        </div>
      ))}
    </div>
  );
}

const RECORD_FIELDS = [
  [38, 64],
  [30, 52],
  [42, 70],
  [26, 46],
  [34, 58],
  [40, 36],
] as const;

function RecordShape() {
  return (
    <div className="dk-skb-rec">
      <i className="dk-sk dk-skb-title" style={width(46)} />
      <i className="dk-sk dk-skb-sub" style={width(30)} />
      <div className="dk-skb-fields">
        {RECORD_FIELDS.map(([label, value], index) => (
          // oxlint-disable-next-line react/no-array-index-key -- a fixed outline that never reorders
          <div key={index} className="dk-skb-field">
            <i className="dk-sk dk-skb-label" style={width(label)} />
            <i className="dk-sk dk-skb-ln" style={width(value)} />
          </div>
        ))}
      </div>
    </div>
  );
}

const LINES = [92, 86, 95, 60] as const;

function LinesShape() {
  return (
    <div className="dk-skb-para">
      {LINES.map((lineWidth, line) => (
        // oxlint-disable-next-line react/no-array-index-key -- a fixed outline that never reorders
        <i key={line} className="dk-sk dk-skb-ln" style={width(lineWidth)} />
      ))}
    </div>
  );
}

/** An artifact's contents while they are read, in the shape its kind is drawn in. */
export function DeskArtifactBodySkeleton({ shape }: { shape: DeskArtifactShape }) {
  const t = useT();

  return (
    <div className="dk-skb" aria-busy>
      <span role="status" className="sr-only">
        {t("Loading the artifact…")}
      </span>
      <div aria-hidden>
        {shape === "table" ? (
          <TableShape />
        ) : shape === "document" ? (
          <DocumentShape />
        ) : shape === "record" ? (
          <RecordShape />
        ) : (
          <LinesShape />
        )}
      </div>
    </div>
  );
}

/**
 * The workspace while it opens: the card of the open artifact at the top,
 * its contents below and the line of actions at the foot, where each will be.
 */
export function DeskWorkspaceSkeleton({ shape = "lines" }: { shape?: DeskArtifactShape }) {
  return (
    <div className="dk-apx" aria-busy>
      <div className="dk-ax-top" aria-hidden>
        <div className="dk-ax-stack">
          <div className="dk-ax-card dk-front dk-skw-card">
            <i className="dk-sk dk-skw-icon" />
            <span className="dk-skw-text">
              <i className="dk-sk dk-skw-title" />
              <i className="dk-sk dk-skw-meta" />
            </span>
          </div>
        </div>
      </div>
      <div className="dk-ax-body">
        <DeskArtifactBodySkeleton shape={shape} />
      </div>
      <div className="dk-ax-foot" aria-hidden>
        <i className="dk-sk dk-skw-link" />
      </div>
    </div>
  );
}
