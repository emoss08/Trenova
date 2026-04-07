import type { CommodityPlacement, HazmatZoneResult } from "@/types/loading-optimization";

export function HazmatZoneOverlay({
  zones,
  placements,
  trailerLengthFeet,
  innerW,
  innerH,
}: {
  zones: HazmatZoneResult[];
  placements: CommodityPlacement[];
  trailerLengthFeet: number;
  innerW: number;
  innerH: number;
}) {
  const placementMap = new Map(placements.map((p) => [p.commodityId, p]));

  function ftToX(ft: number) {
    return (ft / trailerLengthFeet) * innerW;
  }

  return (
    <g>
      {zones.map((zone, idx) => {
        const a = placementMap.get(zone.commodityAId);
        const b = placementMap.get(zone.commodityBId);
        if (!a || !b) return null;

        const aCenter = ftToX(a.positionFeet + a.lengthFeet / 2);
        const bCenter = ftToX(b.positionFeet + b.lengthFeet / 2);
        const midX = (aCenter + bCenter) / 2;
        const satisfied = zone.satisfied;
        const y = innerH + 26;

        return (
          <g key={idx}>
            <line
              x1={aCenter}
              y1={y}
              x2={bCenter}
              y2={y}
              className={satisfied ? "stroke-emerald-500" : "stroke-destructive"}
              strokeWidth={1.5}
              strokeDasharray="4 3"
            />
            <circle cx={aCenter} cy={y} r={3} className={satisfied ? "fill-emerald-500" : "fill-destructive"} />
            <circle cx={bCenter} cy={y} r={3} className={satisfied ? "fill-emerald-500" : "fill-destructive"} />
            <rect
              x={midX - 32}
              y={y - 8}
              width={64}
              height={16}
              rx={4}
              className={`fill-background ${satisfied ? "stroke-emerald-500" : "stroke-destructive"}`}
              strokeWidth={1}
            />
            <text
              x={midX}
              y={y + 1}
              textAnchor="middle"
              dominantBaseline="middle"
              className={`text-[8px] font-semibold ${satisfied ? "fill-emerald-600 dark:fill-emerald-400" : "fill-destructive"}`}
            >
              {zone.actualDistanceFeet}ft
              {zone.requiredDistanceFeet != null ? ` / ${zone.requiredDistanceFeet}ft` : ""}
            </text>
          </g>
        );
      })}
    </g>
  );
}
