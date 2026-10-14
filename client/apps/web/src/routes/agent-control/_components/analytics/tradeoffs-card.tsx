import type { AIAnalyticsSeries } from "@/lib/graphql/ai-analytics";
import { useElementWidth } from "@trenova/shared/hooks/use-element-width";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import type { AiAnalyticsCompareBy } from "@trenova/graphql/generated/graphql";
import { memo, useMemo, useState, type CSSProperties } from "react";
import {
  display,
  median,
  METRICS,
  niceTicks,
  placeLabels,
  shareAxis,
  TRADEOFF_ACROSS,
  TRADEOFF_UP,
  type MetricKey,
  type SeriesColors,
} from "./analytics-model";
import { AnalyticsCard, Swatch, Tip } from "./card";
import { highlightClass, useHighlight } from "./highlight";
import { LabeledSelect } from "./labeled-select";

const HEIGHT = 354;
const PAD = { left: 48, right: 24, top: 30, bottom: 40 };
const LABEL_CHAR_WIDTH = 6.4;

type Point = { series: AIAnalyticsSeries; x: number; y: number };

type TradeoffsCardProps = {
  series: readonly AIAnalyticsSeries[];
  colors: SeriesColors;
  compareBy: AiAnalyticsCompareBy;
  onOpen: (key: string) => void;
};

/** Every series on one chart against two measures, with the corner worth being in shaded. */
export const TradeoffsCard = memo(function TradeoffsCard({
  series,
  colors,
  compareBy,
  onOpen,
}: TradeoffsCardProps) {
  const t = useT();
  const [up, setUp] = useState<MetricKey>("completed");
  const [across, setAcross] = useState<MetricKey>("runTime");
  const [tip, setTip] = useState<{ point: Point; x: number; y: number } | null>(null);
  const [ref, width] = useElementWidth<HTMLDivElement>();
  const { hot, setHot } = useHighlight();
  const Y = METRICS[up];
  const X = METRICS[across];

  const points = useMemo(
    () =>
      series
        .map((item) => ({ series: item, x: X.value(item), y: Y.value(item) }))
        .filter((point): point is Point => point.x !== null && point.y !== null),
    [X, Y, series],
  );

  const innerWidth = Math.max(10, width - PAD.left - PAD.right);
  const innerHeight = HEIGHT - PAD.top - PAD.bottom;
  const xAxis = niceTicks(Math.max(...points.map((point) => point.x), 0) * 1.1 || 1);
  const yShare = shareAxis(points.map((point) => point.y));
  const yAxis = Y.share
    ? { low: yShare.low, high: 1, ticks: yShare.ticks }
    : (() => {
        const nice = niceTicks(Math.max(...points.map((point) => point.y), 0) * 1.1 || 1);
        return { low: 0, high: nice.top, ticks: nice.ticks };
      })();
  const sx = (value: number) => PAD.left + (value / (xAxis.top || 1)) * innerWidth;
  const sy = (value: number) =>
    PAD.top + (1 - (value - yAxis.low) / (yAxis.high - yAxis.low || 1)) * innerHeight;
  const midX = median(points.map((point) => point.x));
  const midY = median(points.map((point) => point.y));
  const mostRuns = Math.max(...points.map((point) => point.series.runs), 1);
  const corner = {
    x: X.higherIsBetter ? sx(midX) : PAD.left,
    width: X.higherIsBetter ? PAD.left + innerWidth - sx(midX) : sx(midX) - PAD.left,
    y: Y.higherIsBetter ? PAD.top : sy(midY),
    height: Y.higherIsBetter ? sy(midY) - PAD.top : PAD.top + innerHeight - sy(midY),
  };

  const labels =
    width === 0
      ? []
      : placeLabels(
          points.map((point) => ({
            key: point.series.key,
            cx: sx(point.x),
            cy: sy(point.y),
            r: 5 + Math.sqrt(point.series.runs / mostRuns) * 8,
            width: point.series.label.length * LABEL_CHAR_WIDTH + 4,
          })),
          PAD.left + innerWidth,
        );
  const byKey = new Map(points.map((point) => [point.series.key, point]));

  return (
    <AnalyticsCard
      title={t("{0} against {1}", t(Y.label), t(X.label).toLowerCase())}
      description={
        compareBy === "Provider"
          ? t("Every provider on one chart. The shaded corner is where you want one to be.")
          : t("Every model on one chart. The shaded corner is where you want a model to be.")
      }
      className="an-scw"
      right={
        <div className="an-ax2">
          <LabeledSelect
            label={t("Up")}
            items={TRADEOFF_UP.map((key) => ({ value: key, label: t(METRICS[key].label) }))}
            value={up}
            onChange={setUp}
          />
          <LabeledSelect
            label={t("Across")}
            items={TRADEOFF_ACROSS.map((key) => ({ value: key, label: t(METRICS[key].label) }))}
            value={across}
            onChange={setAcross}
          />
        </div>
      }
    >
      <div
        className="an-scp"
        ref={ref}
        onMouseLeave={() => {
          setTip(null);
          setHot(null);
        }}
      >
        {points.length === 0 ? (
          <p className="an-nil">{t("Nothing in this scope has both figures yet.")}</p>
        ) : (
          width > 0 && (
            <svg
              width={width}
              height={HEIGHT}
              className="an-svg"
              role="img"
              aria-label={t("{0} against {1}", t(Y.label), t(X.label).toLowerCase())}
            >
              {points.length > 1 && (
                <>
                  <rect
                    x={corner.x}
                    y={corner.y}
                    width={Math.max(0, corner.width)}
                    height={Math.max(0, corner.height)}
                    className="an-qr"
                    rx="4"
                  />
                  <text
                    x={X.higherIsBetter ? PAD.left + innerWidth - 8 : PAD.left + 8}
                    y={Y.higherIsBetter ? PAD.top + 16 : PAD.top + innerHeight - 8}
                    textAnchor={X.higherIsBetter ? "end" : "start"}
                    className="an-qt"
                  >
                    {t("Best here")}
                  </text>
                </>
              )}
              {yAxis.ticks.map((value) => (
                <g key={value}>
                  <line
                    x1={PAD.left}
                    x2={PAD.left + innerWidth}
                    y1={sy(value)}
                    y2={sy(value)}
                    className="an-gl"
                  />
                  <text x={PAD.left - 10} y={sy(value) + 3.5} textAnchor="end" className="an-tk">
                    {display(t, Y, value)}
                  </text>
                </g>
              ))}
              {xAxis.ticks.map((value) => (
                <text
                  key={value}
                  x={sx(value)}
                  y={PAD.top + innerHeight + 18}
                  textAnchor="middle"
                  className="an-tk"
                >
                  {value === 0 && across !== "cost" ? "0" : display(t, X, value)}
                </text>
              ))}
              <text x={PAD.left + innerWidth} y={HEIGHT - 4} textAnchor="end" className="an-al">
                {t(X.label)} {X.higherIsBetter ? "→" : t("← better")}
              </text>
              <text x={PAD.left - 40} y={PAD.top - 16} className="an-al">
                {t(Y.label)} {Y.higherIsBetter ? "↑" : ""}
              </text>
              {labels.map((label) => {
                const point = byKey.get(label.key);
                if (!point) {
                  return null;
                }
                const textX = label.left ? label.cx - label.r - 7 : label.cx + label.r + 7;
                return (
                  <g
                    key={label.key}
                    className={cn("an-pt", highlightClass(hot, label.key))}
                    style={{ "--col": colors.get(label.key) } as CSSProperties}
                    onMouseEnter={() => {
                      setHot(label.key);
                      setTip({ point, x: label.cx, y: label.cy - label.r - 8 });
                    }}
                    onClick={() => onOpen(label.key)}
                  >
                    <circle cx={label.cx} cy={label.cy} r={label.r} className="an-dot" />
                    {Math.abs(label.y - label.cy) > 6 && (
                      <line
                        x1={label.left ? label.cx - label.r - 1 : label.cx + label.r + 1}
                        y1={label.cy}
                        x2={label.left ? label.cx - label.r - 6 : label.cx + label.r + 6}
                        y2={label.y}
                        className="an-ld"
                      />
                    )}
                    <text
                      x={textX}
                      y={label.y + 4}
                      textAnchor={label.left ? "end" : "start"}
                      className="an-pl"
                    >
                      {point.series.label}
                    </text>
                  </g>
                );
              })}
            </svg>
          )
        )}
        {tip && (
          <Tip x={tip.x} y={Math.max(70, tip.y)} bound={width}>
            <div className="an-tth">
              <Swatch color={colors.get(tip.point.series.key)} />
              <b>{tip.point.series.label}</b>
            </div>
            <span className="an-ttr">
              <em>{t(Y.label)}</em>
              <b className="mono">{display(t, Y, tip.point.y)}</b>
            </span>
            <span className="an-ttr">
              <em>{t(X.label)}</em>
              <b className="mono">{display(t, X, tip.point.x)}</b>
            </span>
            <span className="an-ttr">
              <em>{t("Runs")}</em>
              <b className="mono">{tip.point.series.runs.toLocaleString()}</b>
            </span>
          </Tip>
        )}
      </div>
      <p className="an-cap">{t("Bigger dots ran more. Click a dot for the full breakdown.")}</p>
    </AnalyticsCard>
  );
});
