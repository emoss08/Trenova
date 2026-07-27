"use no memo";
import { EmptyState } from "@/components/empty-state";
import { queries } from "@/lib/queries";
import { CalculationReceipt } from "@/routes/detention-desk/_components/calculation-receipt";
import { Badge } from "@trenova/shared/components/ui/badge";
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@trenova/shared/components/ui/collapsible";
import { FormSection } from "@trenova/shared/components/ui/form";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import {
  formatDetentionMinutes,
  OCCURRENCE_STATUS_STYLES,
  SCORE_BAND_STYLES,
  scoreBand,
} from "@trenova/shared/lib/detention";
import { cn, formatCurrency } from "@trenova/shared/lib/utils";
import type { DetentionOccurrence } from "@trenova/shared/types/detention";
import type { Shipment } from "@trenova/shared/types/shipment";
import { useQuery } from "@tanstack/react-query";
import { ChevronDownIcon, TimerIcon } from "lucide-react";
import { useFormContext, useWatch } from "react-hook-form";

function OccurrenceRow({ occurrence }: { occurrence: DetentionOccurrence }) {
  const snapshot = occurrence.policySnapshot;

  return (
    <Collapsible className="rounded-md border">
      <div className="flex items-start justify-between gap-3 p-3">
        <div className="min-w-0 space-y-1">
          <div className="flex flex-wrap items-center gap-1.5">
            <Badge className={cn("border-none", OCCURRENCE_STATUS_STYLES[occurrence.status])}>
              {occurrence.status}
            </Badge>
            <span className="text-sm font-medium">{occurrence.stopType}</span>
            {occurrence.arrivedLate && (
              <Badge variant="outline" className="text-[10px]">
                Late by {formatDetentionMinutes(occurrence.lateByMinutes)}
              </Badge>
            )}
            {occurrence.capApplied !== "None" && (
              <Badge variant="outline" className="text-[10px]">
                {occurrence.capApplied}
              </Badge>
            )}
            {occurrence.suppressedByGate && (
              <Badge className="border-none bg-red-500/15 text-[10px] text-red-700 dark:text-red-400">
                Notice missed
              </Badge>
            )}
          </div>

          <p className="text-muted-foreground text-xs">
            {formatDetentionMinutes(occurrence.rawDwellMinutes)} on site ·{" "}
            {formatDetentionMinutes(occurrence.freeMinutesGranted)} free ·{" "}
            {formatDetentionMinutes(occurrence.roundedMinutes)} billable
          </p>

          {snapshot && (
            <p className="text-muted-foreground/80 text-[11px]">
              Governed by {snapshot.policyName} ({snapshot.policyCode})
            </p>
          )}
        </div>

        <div className="shrink-0 space-y-1 text-right">
          <p className="text-base font-semibold tabular-nums">
            {formatCurrency(occurrence.billableAmount, occurrence.currency)}
          </p>
          {occurrence.driverPayAmount > 0 && (
            <p
              className={cn(
                "text-xs tabular-nums",
                occurrence.netMargin < 0
                  ? "text-red-600 dark:text-red-400"
                  : "text-muted-foreground",
              )}
            >
              margin {formatCurrency(occurrence.netMargin, occurrence.currency)}
            </p>
          )}
          {occurrence.collectabilityScore > 0 && (
            <Badge
              className={cn(
                "border-none text-[10px]",
                SCORE_BAND_STYLES[scoreBand(occurrence.collectabilityScore)],
              )}
            >
              {occurrence.collectabilityScore}/100
            </Badge>
          )}
        </div>
      </div>

      <CollapsibleTrigger className="text-muted-foreground hover:bg-accent/40 flex h-7 w-full items-center justify-start gap-1 border-t px-3 text-xs transition-colors">
        <ChevronDownIcon className="size-3" />
        How this was calculated
      </CollapsibleTrigger>

      <CollapsibleContent className="border-t p-3">
        <CalculationReceipt
          trace={occurrence.calculationTrace}
          currency={occurrence.currency}
        />
      </CollapsibleContent>
    </Collapsible>
  );
}

/**
 * Detention on a shipment: one row per stop, each expanding into the receipt
 * that shows exactly which contract term produced the number. This is the view
 * a billing clerk opens when a customer questions the charge.
 */
export default function ShipmentDetentionSection() {
  const { control } = useFormContext<Shipment>();
  const shipmentId = useWatch({ control, name: "id" });

  const { data, isLoading } = useQuery({
    ...queries.detention.byShipment(shipmentId as string),
    enabled: Boolean(shipmentId),
  });

  const occurrences = data ?? [];
  const total = occurrences.reduce((sum, o) => sum + o.billableAmount, 0);
  const margin = occurrences.reduce((sum, o) => sum + o.netMargin, 0);

  return (
    <FormSection
      title="Detention"
      description="Per-stop detention with the contract terms and derivation that produced each charge"
    >
      {isLoading ? (
        <div className="space-y-2">
          <Skeleton className="h-24 w-full" />
          <Skeleton className="h-24 w-full" />
        </div>
      ) : occurrences.length === 0 ? (
        <EmptyState
          icons={[TimerIcon]}
          title="No detention recorded"
          description="Detention appears here once an arrival is recorded at a stop covered by a detention policy."
        />
      ) : (
        <div className="space-y-3">
          <div className="flex items-center justify-between rounded-md border p-3">
            <div>
              <p className="text-sm font-medium">
                {occurrences.length} detention{" "}
                {occurrences.length === 1 ? "event" : "events"}
              </p>
              <p className="text-muted-foreground text-xs">
                Net margin after driver detention pay
              </p>
            </div>
            <div className="text-right">
              <p className="text-lg font-semibold tabular-nums">
                {formatCurrency(total)}
              </p>
              <p
                className={cn(
                  "text-xs tabular-nums",
                  margin < 0 ? "text-red-600 dark:text-red-400" : "text-muted-foreground",
                )}
              >
                {formatCurrency(margin)} margin
              </p>
            </div>
          </div>

          {occurrences.map((occurrence) => (
            <OccurrenceRow key={occurrence.id} occurrence={occurrence} />
          ))}
        </div>
      )}
    </FormSection>
  );
}
