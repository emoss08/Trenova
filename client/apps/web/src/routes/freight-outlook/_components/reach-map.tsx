import type { FreightReach } from "@/lib/graphql/freight-outlook";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { geoAlbersUsa, geoPath } from "d3-geo";
import { feature, mesh } from "topojson-client";
import type { GeometryCollection, Topology } from "topojson-specification";
import atlas from "us-atlas/states-albers-10m.json";

type Atlas = Topology<{ states: GeometryCollection; nation: GeometryCollection }>;

const US = atlas as unknown as Atlas;
/** The projection us-atlas pre-projects its albers files with, so driver points land on it. */
const PROJECTION = geoAlbersUsa().scale(1300).translate([487.5, 305]);
const PATH = geoPath();
const NATION = PATH(feature(US, US.objects.nation)) ?? "";
const BORDERS = PATH(mesh(US, US.objects.states, (a, b) => a !== b)) ?? "";
const VIEW_WIDTH = 200;
const VIEW_HEIGHT = 100;
/** About the reach of a working day: the ring is drawn at a fixed size, not to scale. */
const RING = 32;

/**
 * Who can get to an area in time, drawn on the states around it: the area, a ring for a day's
 * reach, and each driver with a line in when they arrive in time and pay for the miles.
 */
export default function ReachMap({
  label,
  latitude,
  longitude,
  reach,
}: {
  label: string;
  latitude: number;
  longitude: number;
  reach: readonly FreightReach[];
}) {
  const t = useT();
  const center = PROJECTION([longitude, latitude]);
  if (!center) {
    return null;
  }
  const [cx, cy] = center;
  const k = VIEW_WIDTH / 480;
  const viewBox = `${cx - VIEW_WIDTH * 0.62} ${cy - VIEW_HEIGHT / 2} ${VIEW_WIDTH} ${VIEW_HEIGHT}`;

  return (
    <svg
      viewBox={viewBox}
      className="block h-auto w-full rounded-lg bg-muted/40"
      role="img"
      aria-label={t("Drivers who can reach {0}", label)}
    >
      <path d={NATION} className="fill-muted" />
      <path d={BORDERS} className="fill-none stroke-background" strokeWidth={2 * k} />
      <circle
        cx={cx}
        cy={cy}
        r={RING}
        className="fill-brand/5 stroke-brand"
        strokeWidth={1.2 * k}
        strokeDasharray={`${4 * k} ${4 * k}`}
      />
      {reach.map((row) => {
        const point = PROJECTION([row.longitude, row.latitude]);
        if (!point || row.reach === "Here") return null;
        const good = row.reach === "OnTime";
        return (
          <g key={row.workerId}>
            <line
              x1={point[0]}
              y1={point[1]}
              x2={cx}
              y2={cy}
              strokeWidth={1.6 * k}
              strokeDasharray={good ? undefined : `${3 * k} ${3 * k}`}
              className={good ? "stroke-success-foreground" : "stroke-foreground-faint"}
            />
            <circle
              cx={point[0]}
              cy={point[1]}
              r={4.5 * k}
              strokeWidth={1.5 * k}
              className={cn(
                "stroke-background",
                good ? "fill-success-foreground" : "fill-foreground-faint",
              )}
            />
            <text
              x={point[0] - 6 * k}
              y={point[1] - 7 * k}
              textAnchor="end"
              fontSize={11 * k}
              strokeWidth={3 * k}
              className={cn(
                "stroke-background [paint-order:stroke]",
                good ? "fill-foreground font-medium" : "fill-foreground-subtle",
              )}
            >
              {row.workerName.split(" ")[0]}
            </text>
          </g>
        );
      })}
      <circle
        cx={cx}
        cy={cy}
        r={7 * k}
        strokeWidth={1.5 * k}
        className="fill-warning stroke-warning-foreground"
      />
      <text
        x={cx + 10 * k}
        y={cy + 4 * k}
        fontSize={12.5 * k}
        strokeWidth={3 * k}
        className="fill-foreground stroke-background font-semibold [paint-order:stroke]"
      >
        {label.split(",")[0]}
      </text>
    </svg>
  );
}
