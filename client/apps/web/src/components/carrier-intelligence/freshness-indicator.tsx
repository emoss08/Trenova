import { useNowSeconds } from "@/hooks/use-now-seconds";
import {
  DEFAULT_EXPIRED_AFTER_HOURS,
  DEFAULT_STALE_AFTER_HOURS,
  carrierIntelDepthMerge,
  carrierIntelFreshness,
  carrierIntelProviderLabel,
} from "@/lib/carrier-intelligence";
import type { CarrierIntelDepth } from "@trenova/graphql/generated/graphql";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import { formatRelativeTime } from "@trenova/shared/i18n/format";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixDateTimeMedium } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import { StatusDot } from "./status-dot";
import { useCarrierIntelLabels } from "./use-carrier-intel-labels";

export type FreshnessIndicatorProps = {
  effectiveAsOf: number;
  fetchedAt?: number | null;
  confirmedAt?: number | null;
  sourceAsOf?: number | null;
  provider?: string | null;
  depth?: CarrierIntelDepth | null;
  depthFetchedAt?: number | null;
  fetchedDepth?: CarrierIntelDepth | null;
  staleAfterHours?: number;
  expiredAfterHours?: number;
  assessStaleness?: boolean;
  className?: string;
};

export function FreshnessIndicator({
  effectiveAsOf,
  fetchedAt,
  confirmedAt,
  sourceAsOf,
  provider,
  depth,
  depthFetchedAt,
  fetchedDepth,
  staleAfterHours = DEFAULT_STALE_AFTER_HOURS,
  expiredAfterHours = DEFAULT_EXPIRED_AFTER_HOURS,
  assessStaleness = true,
  className,
}: FreshnessIndicatorProps) {
  const t = useT();
  const labels = useCarrierIntelLabels();
  const now = useNowSeconds();
  const merge = carrierIntelDepthMerge({ depth, depthFetchedAt, fetchedDepth, fetchedAt });
  const mergeNote = merge
    ? t(
        "{0} from {1}, refreshed with {2} {3}",
        labels.depth[merge.held],
        formatRelativeTime(merge.heldAt - now),
        labels.depthSource[merge.fetched],
        formatRelativeTime(merge.fetchedAt - now),
      )
    : null;

  const state = assessStaleness
    ? carrierIntelFreshness({ effectiveAsOf, now, staleAfterHours, expiredAfterHours })
    : "fresh";
  const relative = formatRelativeTime(effectiveAsOf - now);
  const label =
    state === "fresh"
      ? t("as of {0}", relative)
      : state === "stale"
        ? t("Stale · as of {0}", relative)
        : t("Out of date · as of {0}", relative);

  return (
    <Tooltip>
      <TooltipTrigger
        render={
          <span
            className={cn(
              "inline-flex cursor-default items-center gap-2 text-xs tabular-nums",
              state === "fresh" ? "text-muted-foreground" : "text-foreground",
              className,
            )}
            data-freshness={state}
            data-depth-merged={merge ? "true" : undefined}
          />
        }
      >
        {state !== "fresh" ? <StatusDot tone={state === "stale" ? "medium" : "critical"} /> : null}
        {label}
      </TooltipTrigger>
      <TooltipContent className="max-w-80">
        {mergeNote ? <p className="mb-2 text-xs">{mergeNote}</p> : null}
        <dl className="grid grid-cols-[auto_1fr] gap-x-3 gap-y-1 text-xs">
          {fetchedAt ? (
            <>
              <dt className="opacity-70">{t("Fetched")}</dt>
              <dd>{formatUnixDateTimeMedium(fetchedAt)}</dd>
            </>
          ) : null}
          {confirmedAt !== undefined ? (
            <>
              <dt className="opacity-70">{t("Confirmed")}</dt>
              <dd>{confirmedAt ? formatUnixDateTimeMedium(confirmedAt) : t("Not confirmed")}</dd>
            </>
          ) : null}
          {sourceAsOf ? (
            <>
              <dt className="opacity-70">{t("Source as of")}</dt>
              <dd>{formatUnixDateTimeMedium(sourceAsOf)}</dd>
            </>
          ) : null}
          <dt className="opacity-70">{t("Effective as of")}</dt>
          <dd>{formatUnixDateTimeMedium(effectiveAsOf)}</dd>
          {provider !== undefined ? (
            <>
              <dt className="opacity-70">{t("Provider")}</dt>
              <dd>{carrierIntelProviderLabel(provider)}</dd>
            </>
          ) : null}
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
