import type { AIAnalyticsSegment, AIAnalyticsSeries } from "@/lib/graphql/ai-analytics";
import { formatNumber } from "@trenova/shared/i18n/format";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import type { AiWorkOutcome } from "@trenova/graphql/generated/graphql";
import { memo, useMemo, useState, type CSSProperties } from "react";
import { Ic } from "../kit/ic";
import { Seg } from "../kit/layout";
import {
  formatRunTime,
  OUTCOMES,
  outcomeCount,
  RUN_TIME_TICKS_S,
  runTimeOf,
  runTimeOrder,
  runTimePosition,
  runTimeTick,
  type SeriesColors,
} from "./analytics-model";
import { AnalyticsCard, Swatch } from "./card";
import { highlightClass, hoverProps, useHighlight } from "./highlight";

const ROW_HEIGHT = 46;

/** A bar from one point of the axis to another, drawn by transform so a change slides it. */
function span(from: number, to: number, least = 0): CSSProperties {
  return { transform: `translateX(${from * 100}%) scaleX(${Math.max(least, to - from)})` };
}

type RunTimeCardProps = {
  series: readonly AIAnalyticsSeries[];
  colors: SeriesColors;
  note: readonly AIAnalyticsSegment[];
  onOpen: (key: string) => void;
  onOpenProvider: (providerId: string) => void;
};

/**
 * How long runs take per series on a log axis: the middle half as a bar, the slowest five
 * percent as a line, the median as a tick. Switching the outcome re-sorts the rows.
 */
export const RunTimeCard = memo(function RunTimeCard({
  series,
  colors,
  note,
  onOpen,
  onOpenProvider,
}: RunTimeCardProps) {
  const t = useT();
  const { hot, setHot } = useHighlight();
  const [outcome, setOutcome] = useState<AiWorkOutcome>("Completed");
  const order = useMemo(() => runTimeOrder(series, outcome), [outcome, series]);
  const rank = new Map(order.map((key, index) => [key, index]));
  const chosen = OUTCOMES.find((item) => item.outcome === outcome) ?? OUTCOMES[0];

  return (
    <AnalyticsCard
      title={t("How long runs take")}
      description={t(
        "Each bar spans the middle half of runs. The line runs out to the slowest 5%, and the tick marks the median.",
      )}
      className="an-rt"
      right={
        <Seg
          className="sm"
          v={outcome}
          opts={OUTCOMES.map((item) => [item.outcome, t(item.label)] as const)}
          onChange={setOutcome}
          label={t("Outcome")}
        />
      }
    >
      {note.length > 0 && (
        <p className="an-ins">
          <Ic n="clock" s={12} />
          <span>
            {note.map((segment, index) =>
              segment.target === "Provider" && segment.providerId ? (
                <button
                  key={index}
                  type="button"
                  className="an-open ui-focus-ring"
                  onClick={() => onOpenProvider(segment.providerId ?? "")}
                >
                  <b>{segment.text}</b>
                </button>
              ) : segment.strong ? (
                <b key={index}>{segment.text}</b>
              ) : (
                <span key={index}>{segment.text}</span>
              ),
            )}
          </span>
        </p>
      )}
      <div className="an-rth" aria-hidden>
        <span />
        <div className="an-rtax">
          {RUN_TIME_TICKS_S.map((tick) => (
            <em
              key={tick}
              className="mono"
              style={{ left: `${runTimePosition(tick * 1000) * 100}%` }}
            >
              {runTimeTick(tick)}
            </em>
          ))}
        </div>
        <span className="r">{t("Median")}</span>
        <span>{t("Outcome mix")}</span>
      </div>
      <div className="an-rtb" style={{ height: series.length * ROW_HEIGHT }} role="list">
        {series.map((item) => {
          const runTime = runTimeOf(item, outcome);
          const runs = outcomeCount(item, outcome);
          return (
            <div
              key={item.key}
              role="listitem"
              className={cn("an-rtr", highlightClass(hot, item.key))}
              style={{
                transform: `translateY(${(rank.get(item.key) ?? 0) * ROW_HEIGHT}px)`,
                height: ROW_HEIGHT,
              }}
              {...hoverProps(setHot, item.key)}
            >
              <button
                type="button"
                className="an-rtn ui-focus-ring"
                onClick={() => onOpen(item.key)}
              >
                <Swatch color={colors.get(item.key)} />
                <b>{item.label}</b>
              </button>
              <div className="an-rtt">
                {RUN_TIME_TICKS_S.map((tick) => (
                  <u key={tick} style={{ left: `${runTimePosition(tick * 1000) * 100}%` }} />
                ))}
                {runTime && runs > 0 ? (
                  <span
                    className="an-rg"
                    style={{ "--col": colors.get(item.key) } as CSSProperties}
                    title={t(
                      "Quarter {0} · median {1} · three quarters {2} · slowest 5% {3} · {4} runs",
                      formatRunTime(runTime.p25Ms),
                      formatRunTime(runTime.medianMs),
                      formatRunTime(runTime.p75Ms),
                      formatRunTime(runTime.p95Ms),
                      formatNumber(runs),
                    )}
                  >
                    <i
                      className="rg-w"
                      style={span(runTimePosition(runTime.p25Ms), runTimePosition(runTime.p95Ms))}
                    />
                    <i
                      className="rg-b"
                      style={span(
                        runTimePosition(runTime.p25Ms),
                        runTimePosition(runTime.p75Ms),
                        0.01,
                      )}
                    />
                    <i
                      className="rg-m"
                      style={{
                        transform: `translateX(${runTimePosition(runTime.medianMs) * 100}%)`,
                      }}
                    />
                  </span>
                ) : (
                  <span className="an-none">
                    {t("No {0} runs", t(chosen?.label ?? "").toLowerCase())}
                  </span>
                )}
              </div>
              <span className="an-rtv mono">
                {runTime && runs > 0 ? formatRunTime(runTime.medianMs) : "—"}
                <em>{runs > 0 ? t("{0, plural, one {# run} other {# runs}}", runs) : ""}</em>
              </span>
              <span className="an-mix" aria-hidden>
                {OUTCOMES.map(({ outcome: kind }) => {
                  const count = outcomeCount(item, kind);
                  return count > 0 ? (
                    <i
                      key={kind}
                      className={cn(`o-${kind}`, kind === outcome && "on")}
                      style={{ flexGrow: count }}
                    />
                  ) : null;
                })}
              </span>
            </div>
          );
        })}
      </div>
      <div className="an-lgd">
        {OUTCOMES.map((item) => (
          <span key={item.outcome} className={cn(item.outcome === outcome && "on")}>
            <i className={`o-${item.outcome}`} />
            {t(item.title)}
          </span>
        ))}
      </div>
    </AnalyticsCard>
  );
});
