import { formatMillions, formatUsd } from "@/lib/ai-usage-format";
import { queries } from "@/lib/queries";
import { useT } from "@trenova/shared/i18n/use-t";
import { resolveUserTimezone } from "@trenova/shared/lib/date";
import { useQuery } from "@tanstack/react-query";
import { useMemo } from "react";
import { Ic } from "../kit/ic";
import { Figs } from "../kit/layout";
import { DailyCallBars } from "./daily-call-bars";

/** The window the overview's figures cover. */
export const OVERVIEW_WINDOW_DAYS = 7;
/** Usage moves call by call; a minute is fresh enough for figures read at a glance. */
const FIGURES_STALE_MS = 60_000;

type OverviewFiguresProps = {
  /** Opens a provider's editor, to give it a price. Absent when no provider is on. */
  onSetPrices?: () => void;
};

function seconds(ms: number, digits: number): string {
  return (ms / 1000).toFixed(digits);
}

/** The week in four figures: calls with their failures, response time, tokens and spend. */
export function OverviewFigures({ onSetPrices }: OverviewFiguresProps) {
  const t = useT();
  const timezone = useMemo(() => resolveUserTimezone(), []);
  const usageQuery = useQuery({
    ...queries.aiProvider.usage(OVERVIEW_WINDOW_DAYS),
    staleTime: FIGURES_STALE_MS,
  });
  const dailyQuery = useQuery({
    ...queries.aiControl.daily(OVERVIEW_WINDOW_DAYS, timezone),
    staleTime: FIGURES_STALE_MS,
  });

  const usage = usageQuery.data;
  const loading = usageQuery.isLoading;
  const calls = usage?.calls ?? 0;
  const failed = usage?.failed ?? 0;
  const okShare = calls > 0 ? ((calls - failed) / calls) * 100 : 100;
  const priced = (usage?.pricedCalls ?? 0) > 0;
  const inputTokens = usage?.inputTokens ?? 0;
  const outputTokens = usage?.outputTokens ?? 0;
  const pending = "…";

  return (
    <Figs
      items={[
        {
          label: t("Model calls · {0} days", OVERVIEW_WINDOW_DAYS),
          value: loading ? pending : calls.toLocaleString(),
          sub: (
            <span className="fsub">
              {dailyQuery.data && <DailyCallBars days={dailyQuery.data} />}
              <span>
                {failed > 0 && (
                  <>
                    <span className="t-d">{t("{0} failed", failed.toLocaleString())}</span>
                    {" · "}
                  </>
                )}
                {t("{0}% ok", okShare.toFixed(1))}
              </span>
            </span>
          ),
        },
        {
          label: t("Median response"),
          value: loading ? (
            pending
          ) : calls > 0 ? (
            <>
              {seconds(usage?.latencyP50Ms ?? 0, 2)}
              <small>s</small>
            </>
          ) : (
            "—"
          ),
          sub:
            calls > 0
              ? t("Slowest 5% took {0}s or more", seconds(usage?.latencyP95Ms ?? 0, 1))
              : t("No calls yet"),
        },
        {
          label: t("Tokens"),
          value: loading ? (
            pending
          ) : (
            <>
              {formatMillions(inputTokens + outputTokens)}
              <small>M</small>
            </>
          ),
          sub: t("{0}M in · {1}M out", formatMillions(inputTokens), formatMillions(outputTokens)),
        },
        {
          label: t("Spend"),
          value: loading ? (
            pending
          ) : priced ? (
            (formatUsd(usage?.costUsd) ?? "—")
          ) : onSetPrices ? (
            <button type="button" className="fset" onClick={onSetPrices}>
              <Ic n="plus" s={12} />
              {t("Set prices")}
            </button>
          ) : (
            "—"
          ),
          sub: priced ? t("Across every provider") : t("No provider has a price yet"),
          tone: priced ? undefined : "dim",
        },
      ]}
    />
  );
}
