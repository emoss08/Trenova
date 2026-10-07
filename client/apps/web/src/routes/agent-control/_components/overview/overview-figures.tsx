import { KpiStrip, KpiStripItem } from "@/components/kpi/kpi-strip";
import { formatLatency, formatTokens, formatUsd } from "@/lib/ai-usage-format";
import { queries } from "@/lib/queries";
import { PlusIcon } from "@trenova/shared/components/icons";
import { useT } from "@trenova/shared/i18n/use-t";
import { resolveUserTimezone } from "@trenova/shared/lib/date";
import { useQuery } from "@tanstack/react-query";
import { useMemo } from "react";
import { DailyCallBars } from "./daily-call-bars";

/** The window the overview's figures cover. */
export const OVERVIEW_WINDOW_DAYS = 7;
/** Usage moves call by call; a minute is fresh enough for figures read at a glance. */
const FIGURES_STALE_MS = 60_000;

type OverviewFiguresProps = {
  /** Opens a provider's editor, to give it a price. Absent when no provider is on. */
  onSetPrices?: () => void;
};

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

  return (
    <KpiStrip minItemWidth="11rem">
      <KpiStripItem
        size="lg"
        label={t("Model calls · {0} days", OVERVIEW_WINDOW_DAYS)}
        value={loading ? "…" : calls.toLocaleString()}
        sub={
          <span className="flex items-center gap-2">
            {dailyQuery.data && <DailyCallBars days={dailyQuery.data} />}
            <span>
              {failed > 0 && (
                <span className="text-danger">{t("{0} failed", failed.toLocaleString())} · </span>
              )}
              {t("{0}% ok", okShare.toFixed(1))}
            </span>
          </span>
        }
      />
      <KpiStripItem
        size="lg"
        label={t("Median response")}
        value={loading ? "…" : calls > 0 ? formatLatency(usage?.latencyP50Ms ?? 0) : "—"}
        sub={
          calls > 0
            ? t("Slowest 5% took {0} or more", formatLatency(usage?.latencyP95Ms ?? 0))
            : t("No calls yet")
        }
      />
      <KpiStripItem
        size="lg"
        label={t("Tokens")}
        value={
          loading ? "…" : formatTokens((usage?.inputTokens ?? 0) + (usage?.outputTokens ?? 0))
        }
        sub={t(
          "{0} in · {1} out",
          formatTokens(usage?.inputTokens ?? 0),
          formatTokens(usage?.outputTokens ?? 0),
        )}
      />
      <KpiStripItem
        size="lg"
        label={t("Spend")}
        value={
          loading ? (
            "…"
          ) : priced ? (
            (formatUsd(usage?.costUsd) ?? "—")
          ) : onSetPrices ? (
            <button
              type="button"
              onClick={onSetPrices}
              className="ui-focus-ring inline-flex items-center gap-1 rounded-control border border-dashed border-border-strong px-2 py-0.5 text-sm font-medium text-muted-foreground hover:text-foreground"
            >
              <PlusIcon className="size-3" />
              {t("Set prices")}
            </button>
          ) : (
            "—"
          )
        }
        sub={
          priced
            ? usage && usage.pricedCalls < usage.calls
              ? t("Partial: {0} of {1} calls unpriced", usage.calls - usage.pricedCalls, usage.calls)
              : t("Across every provider")
            : t("No provider has a price yet")
        }
      />
    </KpiStrip>
  );
}
