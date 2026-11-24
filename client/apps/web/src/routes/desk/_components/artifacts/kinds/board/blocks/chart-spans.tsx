import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import { ChartFrame } from "./chart-frame";
import { fitText, linear, SPANS_PITCH, spansHeight, tickText } from "./chart-values";
import { circleBox, placeLabels, textWidth, thinTicks, type Box, type Spot } from "./chart-labels";
import type { DeskBoardChart } from "./metric-types";

const NAME_COLUMN = 150;
const MIN_COLUMN = 80;
const NAME_CHAR = 6.9;
/** The room left between a cover's end and its bound mark, and after the mark. */
const CAP = 3;
const AFTER = 7.6;

/** Whether a spans chart has a row to draw. */
export function spansDraw(chart: DeskBoardChart): boolean {
  return chart.rows.length > 0;
}

/** The last moment the chart reaches: its last tick, span or dot, and never before now. */
function horizon(chart: DeskBoardChart, now: number): number {
  const moments = [
    now,
    ...chart.ticks.flatMap((tick) => (tick.at ? [tick.at] : [])),
    ...chart.rows.flatMap((row) => row.spans.map((span) => span.to)),
    ...chart.rows.flatMap((row) => row.dots.flatMap((dot) => (dot.at ? [dot.at] : []))),
  ];
  return Math.max(...moments);
}

/** What the spans show and their headline number: how many marks stand past their bound. */
export function spansLabel(chart: DeskBoardChart, title: string, t: TranslateFn): string {
  const marked = chart.rows.flatMap((row) =>
    row.dots.filter((dot) => dot.marked).map((dot) => (dot.label === "" ? row.label : dot.label)),
  );
  if (marked.length === 0) {
    return t(
      "{0}: {1, plural, one {# row} other {# rows}}, nothing past its bound",
      title,
      chart.rows.length,
    );
  }
  return t(
    "{0}: {1, plural, one {# row} other {# rows}}; {2, plural, one {# past its bound} other {# past their bound}}: {3}",
    title,
    chart.rows.length,
    marked.length,
    new Intl.ListFormat(undefined, { type: "unit" }).format(marked),
  );
}

/** The share of the width the row names may take before they are cut. */
const NAME_SHARE = 0.4;

function SpansMarks({
  chart,
  width,
  now,
  t,
}: {
  chart: DeskBoardChart;
  width: number;
  now: number;
  t: TranslateFn;
}) {
  const right = width - 10;
  const longest = Math.max(0, ...chart.rows.map((row) => textWidth(row.label, "nm")));
  const column = Math.round(
    Math.min(NAME_COLUMN, width * NAME_SHARE, Math.max(MIN_COLUMN, longest + 12)),
  );
  const end = horizon(chart, now);
  const x = linear(now, end, column, right);
  const at = (moment: number) => Math.min(right, Math.max(column, x(moment)));
  const axis = chart.rows.length * SPANS_PITCH + 12;

  const obstacles: Box[] = [];
  const asks: { text: string; cls: "lb"; spots: Spot[] }[] = [];
  const named: { row: number; dot: number }[] = [];
  chart.rows.forEach((row, index) => {
    const middle = 22 + index * SPANS_PITCH;
    obstacles.push({ x0: 0, x1: column, y0: middle - 9, y1: middle + 9 });
    for (const span of row.spans) {
      if (span.kind === "cover" && span.to < end) {
        obstacles.push({
          x0: at(span.to) - 1,
          x1: at(span.to) + 1,
          y0: middle - 9,
          y1: middle + 9,
        });
      }
    }
    row.dots.forEach((dot, dotIndex) => {
      if (!dot.at) return;
      const cx = at(dot.at);
      obstacles.push(circleBox(cx, middle, dot.marked ? 5.5 : 5));
      if (dot.marked && dot.label !== "") {
        named.push({ row: index, dot: dotIndex });
        asks.push({
          text: dot.label,
          cls: "lb",
          spots: [
            { x: cx + 12.8, y: middle + 4, anchor: "start" },
            { x: cx - 12.8, y: middle + 4, anchor: "end" },
            { x: cx + 8, y: middle - 10, anchor: "start" },
            { x: cx - 8, y: middle - 10, anchor: "end" },
            { x: cx + 8, y: middle + 19, anchor: "start" },
            { x: cx - 8, y: middle + 19, anchor: "end" },
          ],
        });
      }
    });
  });
  const spots = placeLabels(asks, obstacles, { x0: column + 2, x1: width, y0: 0, y1: axis - 1 });
  const labelOf = new Map(
    named.flatMap((entry, index) => {
      const spot = spots[index];
      return spot ? [[`${entry.row}:${entry.dot}`, { spot, text: asks[index].text }] as const] : [];
    }),
  );
  const axisTicks = thinTicks(
    [
      { x: column, text: t("Today"), cls: "lb" as const, first: true },
      ...chart.ticks.flatMap((tick) =>
        tick.at && tick.at > now
          ? [{ x: at(tick.at), text: tickText(tick), cls: "tk" as const, first: tick.marked }]
          : [],
      ),
    ],
    { x0: column, x1: width },
  );
  const tickTexts = [
    { text: t("Today"), cls: "lb" },
    ...chart.ticks
      .filter((tick) => tick.at && tick.at > now)
      .map((tick) => ({ text: tickText(tick), cls: "tk" })),
  ];

  return (
    <>
      {chart.rows.map((row, index) => {
        const middle = 22 + index * SPANS_PITCH;
        const name = fitText(row.label, column - 10, NAME_CHAR);
        const covers = row.spans.filter((span) => span.kind === "cover");
        const gaps = row.spans.filter((span) => span.kind !== "cover");
        return (
          // oxlint-disable-next-line react/no-array-index-key -- rows keep their order
          <g key={index}>
            <text className="nm" x={0} y={middle + 4}>
              {name !== row.label && <title>{row.label}</title>}
              {name}
            </text>
            {covers.map((span) => {
              const from = at(span.from) + CAP;
              const to = at(span.to);
              return (
                <g key={`${span.from}-${span.to}`}>
                  {to - CAP > from && (
                    <line className="cov" x1={from} y1={middle} x2={to - CAP} y2={middle} />
                  )}
                  {span.to < end && (
                    <>
                      <line className="bnd" x1={to} y1={middle - 9} x2={to} y2={middle + 9} />
                      {gaps.length === 0 && to + AFTER < right && (
                        <line className="unc" x1={to + AFTER} y1={middle} x2={right} y2={middle} />
                      )}
                    </>
                  )}
                </g>
              );
            })}
            {gaps.map((span) => (
              <line
                key={`${span.from}-${span.to}`}
                className="unc"
                x1={at(span.from) + AFTER}
                y1={middle}
                x2={at(span.to)}
                y2={middle}
              />
            ))}
            {row.dots.map((dot, dotIndex) => {
              if (!dot.at) return null;
              const label = labelOf.get(`${index}:${dotIndex}`);
              return (
                // oxlint-disable-next-line react/no-array-index-key -- a row's dots keep their order
                <g key={dotIndex}>
                  <circle
                    className={dot.marked ? "dot w" : "dot q"}
                    cx={at(dot.at)}
                    cy={middle}
                    r={dot.marked ? 5.5 : 5}
                  />
                  {label && (
                    <text
                      className="lb w"
                      x={label.spot.x}
                      y={label.spot.y}
                      textAnchor={label.spot.anchor}
                    >
                      {label.text}
                    </text>
                  )}
                </g>
              );
            })}
          </g>
        );
      })}
      <line className="ax" x1={column} y1={axis + 0.5} x2={right} y2={axis + 0.5} />
      {axisTicks.map(({ index, spot }) => (
        <text
          key={index}
          className={tickTexts[index].cls}
          x={spot.x}
          y={axis + 16}
          textAnchor={spot.anchor}
        >
          {tickTexts[index].text}
        </text>
      ))}
    </>
  );
}

/**
 * Things against a bound: each row's cover as a solid line from today to its end, a dark mark
 * where it ends and a dotted warn line after it; its loads as dots where they land, the one
 * past the end marked and named on whichever side has room, inside the plot and never over the
 * row names. Today stands at the left of the axis; the other ticks thin until none touch, a
 * marked tick (the day a cover ends) kept before a regular one.
 */
export function SpansChart({
  chart,
  title,
  now,
}: {
  chart: DeskBoardChart;
  title: string;
  now: number;
}) {
  const t = useT();
  return (
    <ChartFrame height={spansHeight(chart.rows.length)} label={spansLabel(chart, title, t)}>
      {(width) => <SpansMarks chart={chart} width={width} now={now} t={t} />}
    </ChartFrame>
  );
}
