import { FORECAST_MIN_LOADS, FORECAST_MIN_WEEKS } from "@/lib/dispatch-forecast";
import type { DispatchForecastQuality } from "@/lib/graphql/dispatch-console";
import { Progress } from "@trenova/shared/components/ui/progress";
import { Spinner } from "@trenova/shared/components/ui/spinner";
import { formatNumber } from "@trenova/shared/i18n/format";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixDateTimeMedium } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";

function HistoryMeter({ label, have, need }: { label: string; have: number; need: number }) {
  const t = useT();
  const met = have >= need;

  return (
    <>
      <span className="text-xs text-muted-foreground">{label}</span>
      <Progress
        size="sm"
        value={Math.min(have, need)}
        max={need}
        variant={met ? "success" : "default"}
        aria-label={label}
      />
      <span className="text-xs tabular-nums">
        {t("{0} of {1}", formatNumber(have), formatNumber(need))}
      </span>
    </>
  );
}

/**
 * How far the organization's history is from what the forecast needs before it is tested, as of
 * the last fit: the weeks that hold loads and the loads picked up.
 */
export function ForecastHistoryMeters({
  quality,
  className,
}: {
  quality: DispatchForecastQuality;
  className?: string;
}) {
  const t = useT();

  return (
    <div
      role="group"
      aria-label={t("History counted so far")}
      className={cn(
        "grid grid-cols-[auto_minmax(0,1fr)_auto] items-center gap-x-3 gap-y-2 text-left",
        className,
      )}
    >
      <HistoryMeter
        label={t("Weeks with loads")}
        have={quality.historyWeeks}
        need={FORECAST_MIN_WEEKS}
      />
      <HistoryMeter
        label={t("Loads picked up")}
        have={quality.historyLoads}
        need={FORECAST_MIN_LOADS}
      />
    </div>
  );
}

/**
 * When the forecast's figures were counted, and whether they are being counted again right now.
 * Nothing before the first fit unless one is under way.
 */
export function ForecastChecked({
  quality,
  className,
}: {
  quality: DispatchForecastQuality;
  className?: string;
}) {
  const t = useT();

  if (!quality.refitting && !quality.fittedAt) return null;

  return (
    <p
      role="status"
      className={cn("flex items-center gap-1.5 text-xs text-muted-foreground", className)}
    >
      {quality.refitting ? <Spinner aria-hidden className="size-3.5 shrink-0" /> : null}
      <span>
        {quality.refitting
          ? quality.fittedAt
            ? t(
                "Last checked {0}. Checking your shipment history again now; this updates when it finishes.",
                formatUnixDateTimeMedium(quality.fittedAt),
              )
            : t("Reading your shipment history now; this updates when it finishes.")
          : t(
              "Last checked {0}. Checked again every night, and whenever this is opened more than an hour later.",
              formatUnixDateTimeMedium(quality.fittedAt),
            )}
      </span>
    </p>
  );
}
