import type React from "react";
import { TooltipProvider } from "@/components/ui/tooltip";
import type { LoadingOptimizationResult } from "@/types/loading-optimization";
import { COMMODITY_PALETTE } from "./constants";
import { HazmatZoneOverlay } from "./hazmat-zone-overlay";

export function TrailerTopView({ data, scoreBadge }: { data: LoadingOptimizationResult; scoreBadge?: React.ReactNode }) {
  const trailerLenFt = data.trailerLengthFeet;
  const W = 640;
  const H = 110;
  const hasStopDividers = (data.stopDividers?.length ?? 0) > 0;
  const pad = { top: hasStopDividers ? 36 : 24, bottom: 28, left: 40, right: 24 };
  const innerW = W - pad.left - pad.right;
  const innerH = H;
  const hazmatExtra = data.hazmatZones.length > 0 ? 32 : 0;
  const svgW = W;
  const svgH = H + pad.top + pad.bottom + hazmatExtra;

  function ftToX(ft: number) {
    return (ft / trailerLenFt) * innerW;
  }

  return (
    <div className="rounded-lg border border-border p-3">
      <div className="mb-1 flex items-center justify-between">
        <span className="text-2xs font-medium tracking-wider text-muted-foreground uppercase">
          Trailer Layout
        </span>
        <div className="flex items-center gap-2">
          {scoreBadge}
          <span className="text-2xs text-muted-foreground">{trailerLenFt}ft</span>
        </div>
      </div>
      <div className="w-full overflow-x-auto">
        <TooltipProvider>
          <svg viewBox={`0 0 ${svgW} ${svgH}`} className="w-full" style={{ minWidth: 460 }}>
            <g transform={`translate(${pad.left}, ${pad.top})`}>
              {/* Trailer shell */}
              <rect
                x={-1}
                y={-1}
                width={innerW + 2}
                height={innerH + 2}
                rx={4}
                className="fill-background stroke-border"
                strokeWidth={2}
              />

              {/* Nose (left) */}
              <path
                d={`M0,0 L-12,${innerH * 0.25} L-12,${innerH * 0.75} L0,${innerH}`}
                className="fill-background stroke-muted-foreground"
                strokeWidth={2}
                strokeLinejoin="round"
              />

              {/* Door lines (right) */}
              <line x1={innerW} y1={4} x2={innerW} y2={innerH * 0.44} className="stroke-foreground/50" strokeWidth={3} strokeLinecap="round" />
              <line x1={innerW} y1={innerH * 0.56} x2={innerW} y2={innerH - 4} className="stroke-foreground/50" strokeWidth={3} strokeLinecap="round" />

              {/* Hatched empty area */}
              <defs>
                <pattern id="hatch" width="6" height="6" patternUnits="userSpaceOnUse" patternTransform="rotate(45)">
                  <line x1="0" y1="0" x2="0" y2="6" className="stroke-border" strokeWidth={1} />
                </pattern>
              </defs>
              {(() => {
                const usedEnd = data.placements.reduce((m, p) => Math.max(m, p.positionFeet + p.lengthFeet), 0);
                const emptyX = ftToX(usedEnd);
                const emptyW = innerW - emptyX;
                if (emptyW < 4) return null;
                return <rect x={emptyX} y={0} width={emptyW} height={innerH} fill="url(#hatch)" rx={2} />;
              })()}

              {/* Clip for overflow */}
              <clipPath id="trailerClip">
                <rect x={0} y={0} width={innerW} height={innerH} rx={4} />
              </clipPath>

              {/* Commodity blocks */}
              <g clipPath="url(#trailerClip)">
                {data.placements.map((p, idx) => {
                  const palette = COMMODITY_PALETTE[idx % COMMODITY_PALETTE.length];
                  const bx = ftToX(p.positionFeet);
                  const bw = Math.max(ftToX(p.lengthFeet), 36);
                  const m = 2;

                  return (
                    <g key={p.commodityId}>
                      <rect
                        x={bx + m}
                        y={m}
                        width={bw - m * 2}
                        height={innerH - m * 2}
                        className={`${palette.fill} ${palette.stroke}`}
                        strokeWidth={2}
                        strokeDasharray={p.fragile ? "6 3" : undefined}
                        rx={5}
                      />
                      {/* Top accent */}
                      <rect
                        x={bx + m + 4}
                        y={m + 3}
                        width={bw - m * 2 - 8}
                        height={3}
                        rx={1.5}
                        className={palette.stroke.replace("stroke-", "fill-")}
                        opacity={0.4}
                      />
                      {/* Name */}
                      <text
                        x={bx + bw / 2}
                        y={innerH / 2 - 10}
                        textAnchor="middle"
                        dominantBaseline="middle"
                        className={`${palette.text} text-[11px] font-bold`}
                      >
                        {p.commodityName.length > Math.floor(bw / 8)
                          ? p.commodityName.slice(0, Math.floor(bw / 8)) + "\u2026"
                          : p.commodityName}
                      </text>
                      {/* Weight */}
                      <text
                        x={bx + bw / 2}
                        y={innerH / 2 + 4}
                        textAnchor="middle"
                        dominantBaseline="middle"
                        className={`${palette.text} text-[9px] opacity-70`}
                      >
                        {p.weight.toLocaleString()} lbs
                      </text>
                      {/* Length + pieces */}
                      <text
                        x={bx + bw / 2}
                        y={innerH / 2 + 17}
                        textAnchor="middle"
                        dominantBaseline="middle"
                        className={`${palette.text} text-[8px] opacity-50`}
                      >
                        {p.lengthFeet}ft{p.estimatedLength ? "*" : ""} &middot; {p.pieces}pc
                      </text>
                      {/* Hazmat badge */}
                      {p.isHazmat && (
                        <g>
                          <circle cx={bx + m + 14} cy={m + 18} r={8} className="fill-amber-400 dark:fill-amber-600" />
                          <text x={bx + m + 14} y={m + 19} textAnchor="middle" dominantBaseline="middle" className="text-[10px]">{"\u2623"}</text>
                        </g>
                      )}
                      {/* Fragile badge */}
                      {p.fragile && !p.isHazmat && (
                        <g>
                          <circle cx={bx + m + 14} cy={m + 18} r={8} className="fill-red-400 dark:fill-red-600" />
                          <text x={bx + m + 14} y={m + 19} textAnchor="middle" dominantBaseline="middle" className="text-[10px]">{"\u26a0"}</text>
                        </g>
                      )}
                    </g>
                  );
                })}
              </g>

              {/* Stop dividers */}
              {data.stopDividers?.map((divider) => {
                const dx = ftToX(divider.positionFeet);
                return (
                  <g key={divider.stopNumber}>
                    <line x1={dx} y1={-4} x2={dx} y2={innerH + 4} className="stroke-primary/60" strokeWidth={2} strokeDasharray="6 4" />
                    <rect x={dx - 40} y={-16} width={80} height={14} rx={3} className="fill-primary/15" />
                    <text x={dx} y={-7} textAnchor="middle" className="fill-primary text-[8px] font-semibold">
                      {divider.label}
                    </text>
                  </g>
                );
              })}

              {/* Overflow indicator */}
              {data.totalLinearFeet > trailerLenFt && (
                <g>
                  <line x1={innerW} y1={-4} x2={innerW} y2={innerH + 4} className="stroke-destructive" strokeWidth={2} strokeDasharray="6 3" />
                  <text x={innerW - 4} y={innerH + 12} textAnchor="end" className="fill-destructive text-[8px] font-semibold">OVER</text>
                </g>
              )}

              {/* Hazmat overlay */}
              {data.hazmatZones.length > 0 && (
                <HazmatZoneOverlay zones={data.hazmatZones} placements={data.placements} trailerLengthFeet={trailerLenFt} innerW={innerW} innerH={innerH} />
              )}

              {/* Labels */}
              <text x={-6} y={-8} textAnchor="middle" className="fill-muted-foreground text-[8px] font-semibold">NOSE</text>
              <text x={innerW + 4} y={-8} textAnchor="middle" className="fill-muted-foreground text-[8px] font-semibold">DOORS</text>

              {/* Ruler */}
              <g transform={`translate(0, ${innerH + 6})`}>
                <line x1={0} y1={0} x2={innerW} y2={0} className="stroke-border" strokeWidth={0.5} />
                <line x1={0} y1={-2} x2={0} y2={2} className="stroke-border" strokeWidth={1} />
                <line x1={innerW} y1={-2} x2={innerW} y2={2} className="stroke-border" strokeWidth={1} />
                {Array.from({ length: Math.floor(trailerLenFt / 10) }, (_, i) => {
                  const tick = (i + 1) * 10;
                  const tx = ftToX(tick);
                  return (
                    <g key={tick}>
                      <line x1={tx} y1={-2} x2={tx} y2={2} className="stroke-border" strokeWidth={0.5} />
                      <text x={tx} y={11} textAnchor="middle" className="fill-muted-foreground text-[7px]">{tick}</text>
                    </g>
                  );
                })}
                <text x={innerW / 2} y={20} textAnchor="middle" className="fill-muted-foreground text-[8px]">
                  {trailerLenFt} ft total
                </text>
              </g>
            </g>
          </svg>
        </TooltipProvider>
      </div>
    </div>
  );
}
