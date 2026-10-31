import { useT } from "@trenova/shared/i18n/use-t";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import { formatMinutesSpan, formatUnixDateTimeShort } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import type { StopEta } from "@trenova/shared/types/shipment";
import { ETA_VERDICT_TONE, useEtaLabels } from "@/hooks/use-eta-labels";

export function stopEtaSlackText(
  slackMinutes: number | null | undefined,
  t: ReturnType<typeof useT>,
): string | null {
  if (slackMinutes == null) return null;
  if (slackMinutes < 0) return t("{0} past the window", formatMinutesSpan(-slackMinutes));
  return t("{0} to spare", formatMinutesSpan(slackMinutes));
}

/** The dot beside a verdict, in the tone its badge would take. */
export const VERDICT_DOT: Record<StopEta["verdict"], string> = {
  OnTime: "bg-success",
  AtRisk: "bg-warning",
  Late: "bg-danger",
  Unknown: "bg-foreground-subtle",
};

/**
 * A stop's projected arrival in one quiet line: a dot in the verdict's tone, the verdict, and the
 * projected time; how it was worked out (slack, basis, rest, stale inputs) on hover. Nothing once
 * the truck is there or while there is no projection stored.
 */
export function StopEtaLine({ eta }: { eta: StopEta | undefined }) {
  const t = useT();
  const labels = useEtaLabels();

  if (!eta || eta.arrived) return null;

  const slack = stopEtaSlackText(eta.slackMinutes, t);
  const arrival = eta.estimatedArrival ? formatUnixDateTimeShort(eta.estimatedArrival) : null;

  return (
    <Tooltip>
      <TooltipTrigger
        render={
          <span
            tabIndex={0}
            className="ui-focus-ring text-2xs flex cursor-help items-center gap-1.5 rounded-sm"
            data-testid="stop-eta-line"
            data-tone={ETA_VERDICT_TONE[eta.verdict]}
          >
            <span aria-hidden className={cn("size-1.5 rounded-full", VERDICT_DOT[eta.verdict])} />
            <span className="font-medium">{labels.verdict(eta.verdict)}</span>
            {arrival ? (
              <span className="text-foreground-muted tabular-nums">{t("ETA {0}", arrival)}</span>
            ) : null}
          </span>
        }
      />
      <TooltipContent side="top" className="max-w-xs">
        <div className="space-y-1 text-xs">
          {slack ? <p>{slack}</p> : null}
          <p>
            {eta.driveMethod !== "None"
              ? t("From: {0} · {1}", labels.basis(eta.basis), labels.driveMethod(eta.driveMethod))
              : t("From: {0}", labels.basis(eta.basis))}
          </p>
          {eta.restMinutes > 0 ? (
            <p>
              {t("Includes {0} of rest for hours of service", formatMinutesSpan(eta.restMinutes))}
            </p>
          ) : null}
          {eta.inputStale ? <p>{t("Working from old information")}</p> : null}
          {eta.reason ? <p className="text-muted-foreground">{eta.reason}</p> : null}
        </div>
      </TooltipContent>
    </Tooltip>
  );
}
