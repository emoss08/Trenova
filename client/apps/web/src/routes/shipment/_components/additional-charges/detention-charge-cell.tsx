import { Button } from "@trenova/shared/components/ui/button";
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
      label: "Notice missed",
      detail:
        "No qualifying notice reached the customer inside the policy window — they have grounds to refuse this charge.",
      className: "bg-red-500/10 text-red-700 dark:text-red-400",
    };
  }

  if (occurrence.requiresApproval) {
    return {
      label: "Needs approval",
      detail:
        "This charge is over the policy approval threshold and cannot be billed until cleared.",
      className: "bg-amber-500/10 text-amber-700 dark:text-amber-400",
    };
  }

  if (occurrence.status === "Disputed") {
    return {
      label: "Disputed",
      detail: "The customer has rejected this charge. Work the claim before it is invoiced.",
      className: "bg-amber-500/10 text-amber-700 dark:text-amber-400",
    };
  }

  return null;
}

export function DetentionChargeLabel({
  code,
  occurrence,
}: {
  code: string;
  occurrence: DetentionOccurrence | undefined;
}) {
  const risk = occurrence ? detentionRisk(occurrence) : null;

  return (
    <>
      <TimerIcon className="text-primary size-3 shrink-0" />
      {code}
      <span className="bg-primary/10 text-2xs text-primary rounded px-1 py-0.5">Detention</span>
      {risk && (
        <Tooltip>
          <TooltipTrigger>
            <span className={cn("text-2xs rounded px-1 py-0.5", risk.className)}>{risk.label}</span>
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
 * Detention bills time, so the unit column reads as time. Falling back to the
 * stored unit keeps the row honest while the occurrence is still loading.
 */
export function DetentionChargeUnit({
  unit,
  occurrence,
}: {
  unit: number;
  occurrence: DetentionOccurrence | undefined;
}) {
  if (!occurrence) return <>{unit}</>;

  return (
    <Tooltip>
      <TooltipTrigger className="cursor-help underline decoration-dotted underline-offset-2">
        {formatDetentionMinutes(occurrence.roundedMinutes)}
      </TooltipTrigger>
      <TooltipContent side="top" sideOffset={6}>
        <p className="max-w-56 text-xs">
          {formatDetentionMinutes(occurrence.rawDwellMinutes)} on site,{" "}
          {formatDetentionMinutes(occurrence.freeMinutesGranted)} free
          {occurrence.capApplied !== "None" && (
            <> · {CAP_KIND_LABEL[occurrence.capApplied]} applied</>
          )}
        </p>
      </TooltipContent>
    </Tooltip>
  );
}

export function DetentionChargeAction({
  occurrence,
  onOpenClaimFile,
}: {
  occurrence: DetentionOccurrence | undefined;
  onOpenClaimFile: () => void;
}) {
  return (
    <Tooltip>
      <TooltipTrigger
        render={
          <Button
            type="button"
            variant="ghost"
            size="icon"
            className="size-7"
            onClick={onOpenClaimFile}
          >
            <TimerIcon className="text-primary size-3.5" />
          </Button>
        }
      />
      <TooltipContent side="top" sideOffset={6}>
        <p className="max-w-56 text-xs">
          {occurrence
            ? `${OCCURRENCE_STATUS_LABEL[occurrence.status]} — open the claim file for the derivation, evidence and notices`
            : "Open the detention claim file"}
        </p>
      </TooltipContent>
    </Tooltip>
  );
}
