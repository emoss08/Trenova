import { translate } from "@trenova/shared/i18n/runtime";
import { useT } from "@trenova/shared/i18n/use-t";
import { Button } from "@trenova/shared/components/ui/button";
import { Popover, PopoverContent, PopoverTrigger } from "@trenova/shared/components/ui/popover";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import {
  CAP_KIND_LABEL,
  formatDetentionMinutes,
  OCCURRENCE_STATUS_LABEL,
} from "@trenova/shared/lib/detention";
import { cn } from "@trenova/shared/lib/utils";
import type { DetentionOccurrence } from "@trenova/shared/types/detention";
import { TimerIcon } from "lucide-react";

type DetentionRisk = {
  rank: number;
  label: string;
  detail: string;
  className: string;
};

/**
 * The one thing about a detention charge that changes what someone does next:
 * whether it can actually be collected. Everything else is derivation and lives
 * in the claim file.
 */
function detentionRisk(occurrence: DetentionOccurrence): DetentionRisk | null {
  if (occurrence.suppressedByGate) {
    return {
      rank: 3,
      label: translate("Notice missed"),
      detail: translate(
        "No qualifying notice reached the customer inside the policy window — they have grounds to refuse this charge.",
      ),
      className: "bg-red-500/10 text-red-700 dark:text-red-400",
    };
  }

  if (occurrence.requiresApproval) {
    return {
      rank: 2,
      label: translate("Needs approval"),
      detail: translate(
        "This charge is over the policy approval threshold and cannot be billed until cleared.",
      ),
      className: "bg-amber-500/10 text-amber-700 dark:text-amber-400",
    };
  }

  if (occurrence.status === "Disputed") {
    return {
      rank: 1,
      label: translate("Disputed"),
      detail: translate(
        "The customer has rejected this charge. Work the claim before it is invoiced.",
      ),
      className: "bg-amber-500/10 text-amber-700 dark:text-amber-400",
    };
  }

  return null;
}

/**
 * One charge bills every detained stop on the shipment, so the row carries the
 * worst risk among them: a single missed notice is grounds to refuse the whole
 * line.
 */
function worstDetentionRisk(occurrences: DetentionOccurrence[]): DetentionRisk | null {
  let worst: DetentionRisk | null = null;
  for (const occurrence of occurrences) {
    const risk = detentionRisk(occurrence);
    if (risk && (!worst || risk.rank > worst.rank)) worst = risk;
  }
  return worst;
}

function stopCaption(occurrence: DetentionOccurrence, t: (key: string) => string) {
  return occurrence.locationName
    ? `${t(occurrence.stopType)} · ${occurrence.locationName}`
    : t(occurrence.stopType);
}

export function DetentionChargeLabel({
  code,
  occurrences,
}: {
  code: string;
  occurrences: DetentionOccurrence[];
}) {
  const t = useT();

  const risk = worstDetentionRisk(occurrences);

  return (
    <>
      <TimerIcon className="text-primary size-3 shrink-0" />
      {code}
      <span className="bg-primary/10 text-2xs text-primary rounded px-1 py-0.5">
        {t("Detention")}
      </span>
      {occurrences.length > 1 && (
        <span className="bg-muted text-2xs text-muted-foreground rounded px-1 py-0.5">
          {t("{0, plural, one {# stop} other {# stops}}", occurrences.length)}
        </span>
      )}
      {risk && (
        <Tooltip>
          <TooltipTrigger>
            <span className={cn("text-2xs rounded px-1 py-0.5", risk.className)}>
              {t(risk.label)}
            </span>
          </TooltipTrigger>
          <TooltipContent side="top" sideOffset={6}>
            <p className="max-w-56 text-xs">{risk.detail}</p>
          </TooltipContent>
        </Tooltip>
      )}
    </>
  );
}

/**
 * Detention bills time, so the unit column reads as time: the billable minutes
 * across every stop the charge covers, with the per-stop split in the tooltip.
 * Falling back to the stored unit keeps the row honest while the occurrences
 * are still loading.
 */
export function DetentionChargeUnit({
  unit,
  occurrences,
}: {
  unit: number;
  occurrences: DetentionOccurrence[];
}) {
  const t = useT();

  if (occurrences.length === 0) return <>{unit}</>;

  const totalMinutes = occurrences.reduce((sum, occurrence) => sum + occurrence.roundedMinutes, 0);

  return (
    <Tooltip>
      <TooltipTrigger className="cursor-help underline decoration-dotted underline-offset-2">
        {formatDetentionMinutes(totalMinutes)}
      </TooltipTrigger>
      <TooltipContent side="top" sideOffset={6}>
        <div className="max-w-64 space-y-1 text-xs">
          {occurrences.map((occurrence) => (
            <p key={occurrence.id}>
              {occurrences.length > 1 && (
                <span className="font-medium">{stopCaption(occurrence, t)}: </span>
              )}
              {formatDetentionMinutes(occurrence.rawDwellMinutes)} {t("on site,")}{" "}
              {formatDetentionMinutes(occurrence.freeMinutesGranted)} {t("free")}
              {occurrence.capApplied !== "None" && (
                <> {t("· {0} applied", CAP_KIND_LABEL[occurrence.capApplied])}</>
              )}
            </p>
          ))}
        </div>
      </TooltipContent>
    </Tooltip>
  );
}

function ClaimFileButton({ onClick, disabled }: { onClick?: () => void; disabled?: boolean }) {
  return (
    <Button
      type="button"
      variant="ghost"
      size="icon"
      className="size-7"
      onClick={onClick}
      disabled={disabled}
    >
      <TimerIcon className="text-primary size-3.5" />
    </Button>
  );
}

export function DetentionChargeAction({
  occurrences,
  onOpenClaimFile,
}: {
  occurrences: DetentionOccurrence[];
  onOpenClaimFile: (occurrenceId: string) => void;
}) {
  const t = useT();

  if (occurrences.length > 1) {
    return (
      <Popover>
        <PopoverTrigger render={<ClaimFileButton />} />
        <PopoverContent align="end" className="w-72 p-2">
          <p className="text-muted-foreground px-2 pb-1 text-xs">
            {t("This charge bills {0} detained stops. Open a claim file:", occurrences.length)}
          </p>
          <div className="divide-y">
            {occurrences.map((occurrence) => (
              <button
                key={occurrence.id}
                type="button"
                onClick={() => onOpenClaimFile(occurrence.id)}
                className="hover:bg-muted flex w-full items-center justify-between gap-2 rounded px-2 py-1.5 text-left text-xs"
              >
                <span className="truncate">
                  <span className="font-medium">{stopCaption(occurrence, t)}</span>
                  <span className="text-muted-foreground">
                    {" "}
                    · {t(OCCURRENCE_STATUS_LABEL[occurrence.status])}
                  </span>
                </span>
                <span className="shrink-0 tabular-nums">
                  ${Number(occurrence.billableAmount).toFixed(2)}
                </span>
              </button>
            ))}
          </div>
        </PopoverContent>
      </Popover>
    );
  }

  const occurrence = occurrences[0];

  return (
    <Tooltip>
      <TooltipTrigger
        render={
          <ClaimFileButton
            disabled={!occurrence}
            onClick={occurrence ? () => onOpenClaimFile(occurrence.id) : undefined}
          />
        }
      />
      <TooltipContent side="top" sideOffset={6}>
        <p className="max-w-56 text-xs">
          {occurrence
            ? t(
                "{0} — open the claim file for the derivation, evidence and notices",
                OCCURRENCE_STATUS_LABEL[occurrence.status],
              )
            : t("Loading the detention claim file")}
        </p>
      </TooltipContent>
    </Tooltip>
  );
}
