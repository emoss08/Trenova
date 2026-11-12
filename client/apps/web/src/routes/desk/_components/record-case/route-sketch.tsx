import type { DeskCaseRoute } from "@/lib/graphql/desk-case";
import { useT } from "@trenova/shared/i18n/use-t";

const WIDTH = 300;
const HEIGHT = 92;
const PAD_X = 14;
const PAD_TOP = 16;
const PAD_BOTTOM = 22;

type Point = { x: number; y: number };

function place(x: number, y: number): Point {
  return {
    x: PAD_X + x * (WIDTH - 2 * PAD_X),
    y: PAD_TOP + y * (HEIGHT - PAD_TOP - PAD_BOTTOM),
  };
}

/** A smooth line through the points, in order: each bend eased by its neighbours. */
function pathThrough(points: readonly Point[]): string {
  if (points.length === 0) {
    return "";
  }
  const [first, ...rest] = points;
  let path = `M${first!.x.toFixed(1)} ${first!.y.toFixed(1)}`;
  let previous = first!;
  rest.forEach((point, index) => {
    const before = points[index - 1] ?? previous;
    const after = points[index + 2] ?? point;
    const c1 = {
      x: previous.x + (point.x - before.x) / 6,
      y: previous.y + (point.y - before.y) / 6,
    };
    const c2 = { x: point.x - (after.x - previous.x) / 6, y: point.y - (after.y - previous.y) / 6 };
    path += ` C${c1.x.toFixed(1)} ${c1.y.toFixed(1)}, ${c2.x.toFixed(1)} ${c2.y.toFixed(1)}, ${point.x.toFixed(1)} ${point.y.toFixed(1)}`;
    previous = point;
  });

  return path;
}

/**
 * The route as a sketch: the stops in order, the part already driven drawn
 * heavier, the truck where it last reported, and a stop the chosen option
 * would add drawn dashed. It is a picture of where things stand, not a map.
 */
export function RouteSketch({ route }: { route: DeskCaseRoute }) {
  const t = useT();
  const stops = route.points.map((point) => ({ ...point, at: place(point.x, point.y) }));
  const truck = route.truck ? place(route.truck.x, route.truck.y) : null;
  const lastDone = stops.reduce((last, stop, index) => (stop.state === "Done" ? index : last), -1);
  const driven = stops.slice(0, lastDone + 1).map((stop) => stop.at);
  if (truck && lastDone >= 0 && lastDone < stops.length - 1) {
    driven.push(truck);
  }
  const names = stops.map((stop) => stop.label).filter((label) => label !== "");
  const label =
    names.length >= 2
      ? t("Route from {0} to {1}", names[0]!, names[names.length - 1]!)
      : t("The route");

  return (
    <svg className="dk-cz-route" viewBox={`0 0 ${WIDTH} ${HEIGHT}`} role="img" aria-label={label}>
      <path
        d={pathThrough(stops.map((stop) => stop.at))}
        fill="none"
        stroke="var(--dsk-b-strong)"
        strokeWidth="2"
      />
      {driven.length >= 2 && (
        <path d={pathThrough(driven)} fill="none" stroke="var(--dsk-fg2)" strokeWidth="2.5" />
      )}
      {stops.map((stop, index) => {
        const proposed = stop.state === "Proposed";
        const anchor = index === 0 ? "start" : index === stops.length - 1 ? "end" : "middle";
        const above = index % 2 === 1;
        return (
          <g key={`${index}:${stop.label}`}>
            <circle
              cx={stop.at.x}
              cy={stop.at.y}
              r={proposed ? 5 : 3.5}
              fill={stop.state === "Done" ? "var(--dsk-fg2)" : "var(--dsk-card)"}
              stroke={proposed ? "var(--dsk-blue)" : "var(--dsk-b-strong)"}
              strokeWidth="1.5"
              strokeDasharray={proposed ? "2 2" : undefined}
            />
            <text
              x={stop.at.x}
              y={stop.at.y + (above ? -10 : 16)}
              textAnchor={anchor}
              fontSize="9.5"
              fill="var(--dsk-subtle)"
            >
              {stop.label}
            </text>
          </g>
        );
      })}
      {truck && (
        <g transform={`translate(${truck.x.toFixed(1)} ${truck.y.toFixed(1)})`}>
          <rect x="-6" y="-6" width="12" height="12" rx="3" fill="var(--dsk-ink)" />
          <path d="M-2 -2 L3 0 L-2 2 Z" fill="var(--dsk-ink-fg)" />
        </g>
      )}
      {route.callout !== "" && truck && (
        <text
          x={WIDTH - PAD_X}
          y={HEIGHT - 5}
          textAnchor="end"
          fontSize="9"
          fill={
            route.calloutTone === "Bad" ? "var(--danger-subtle-foreground)" : "var(--dsk-warn-fg)"
          }
          style={{ fontVariantNumeric: "tabular-nums" }}
        >
          {route.callout}
        </text>
      )}
    </svg>
  );
}
