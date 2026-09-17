import { useT } from "@trenova/shared/i18n/use-t";
import { useNowSeconds } from "@/hooks/use-now-seconds";
import {
  DEFAULT_EXPIRED_AFTER_HOURS,
  DEFAULT_STALE_AFTER_HOURS,
  carrierIntelFreshness,
} from "@/lib/carrier-intelligence";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import { formatRelativeTime } from "@trenova/shared/i18n/format";
import { formatUnixDateTimeMedium } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import { ClockAlertIcon, ClockIcon, TriangleAlertIcon } from "lucide-react";

export type FreshnessIndicatorProps = {
  fetchedAt: number;
  confirmedAt?: number | null;
  effectiveAsOf: number;
  sourceAsOf?: number | null;
  staleAfterHours?: number;
  expiredAfterHours?: number;
  now?: number;
  className?: string;
};

const FRESHNESS_CLASSES = {
  fresh: "text-muted-foreground",
  stale: "text-yellow-700 dark:text-yellow-400",
  expired: "text-red-700 dark:text-red-400",
} as const;

export function FreshnessIndicator({
  fetchedAt,
  confirmedAt,
  effectiveAsOf,
  sourceAsOf,
  staleAfterHours = DEFAULT_STALE_AFTER_HOURS,
  expiredAfterHours = DEFAULT_EXPIRED_AFTER_HOURS,
  now: nowOverride,
  className,
}: FreshnessIndicatorProps) {
  const t = useT();
  const clock = useNowSeconds();
  const now = nowOverride ?? clock;

  const state = carrierIntelFreshness({ effectiveAsOf, now, staleAfterHours, expiredAfterHours });
  const relative = formatRelativeTime(effectiveAsOf - now);
  const Icon =
    state === "fresh" ? ClockIcon : state === "stale" ? ClockAlertIcon : TriangleAlertIcon;

  const label =
    state === "fresh"
      ? t("As of {0}", relative)
      : state === "stale"
        ? t("Stale · as of {0}", relative)
        : t("Out of date · as of {0}", relative);

  return (
    <Tooltip>
      <TooltipTrigger
        render={
          <span
            className={cn(
              "inline-flex cursor-help items-center gap-1 text-xs",
              FRESHNESS_CLASSES[state],
              className,
            )}
            data-freshness={state}
          />
        }
      >
        <Icon className="size-3.5" aria-hidden />
        {label}
      </TooltipTrigger>
      <TooltipContent className="max-w-80">
        <dl className="grid grid-cols-[auto_1fr] gap-x-3 gap-y-1 text-xs">
          <dt className="text-muted-foreground">{t("Fetched")}</dt>
          <dd>{formatUnixDateTimeMedium(fetchedAt)}</dd>
          <dt className="text-muted-foreground">{t("Confirmed")}</dt>
          <dd>{confirmedAt ? formatUnixDateTimeMedium(confirmedAt) : t("Not confirmed")}</dd>
          {sourceAsOf ? (
            <>
              <dt className="text-muted-foreground">{t("Source as of")}</dt>
              <dd>{formatUnixDateTimeMedium(sourceAsOf)}</dd>
            </>
          ) : null}
          <dt className="text-muted-foreground">{t("Effective as of")}</dt>
          <dd>{formatUnixDateTimeMedium(effectiveAsOf)}</dd>
        </dl>
        {state === "stale" ? (
          <p className="mt-2 text-xs">
            {t(
              "Older than {0} hours. Tendering refreshes it first when pre-tender refresh is on.",
              staleAfterHours,
            )}
          </p>
        ) : null}
        {state === "expired" ? (
          <p className="mt-2 text-xs">
            {t(
              "Older than {0} hours, past the maximum age the tender gate accepts. Vet the carrier again.",
              expiredAfterHours,
            )}
          </p>
        ) : null}
      </TooltipContent>
    </Tooltip>
  );
}
