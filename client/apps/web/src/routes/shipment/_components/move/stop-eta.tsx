import { useT } from "@trenova/shared/i18n/use-t";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import { formatMinutesSpan, formatToUserTimezone } from "@trenova/shared/lib/date";
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

export function StopEtaLine({ eta }: { eta: StopEta | undefined }) {
  const t = useT();
  const labels = useEtaLabels();

  if (!eta || eta.arrived) return null;

  const slack = stopEtaSlackText(eta.slackMinutes, t);
  const arrival = eta.estimatedArrival
    ? formatToUserTimezone(eta.estimatedArrival, { showTimeZone: false, showSeconds: false })
    : null;

  return (
    <Tooltip>
      <TooltipTrigger
        render={
          <div
            className="flex shrink-0 flex-col items-end gap-0.5 pt-0.5 text-xs"
            data-testid="stop-eta-line"
          >
            <Badge variant={ETA_VERDICT_TONE[eta.verdict]}>{labels.verdict(eta.verdict)}</Badge>
            {arrival ? (
              <span className="font-mono tabular-nums">{t("ETA {0}", arrival)}</span>
            ) : null}
          </div>
        }
      />
      <TooltipContent side="top" className="max-w-xs">
        <div className="space-y-1 text-xs">
          {slack ? <p>{slack}</p> : null}
          <p>
            {t("From: {0}", labels.basis(eta.basis))}
            {eta.driveMethod !== "None" ? ` · ${labels.driveMethod(eta.driveMethod)}` : ""}
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
