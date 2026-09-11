import { translate } from "@trenova/shared/i18n/runtime";
import { HoverCardTimestamp } from "@/components/hover-card-timestamp";
import { findChoice, stopTypeChoices } from "@/lib/choices";
import { cn } from "@trenova/shared/lib/utils";
import type { ServiceFailureStopSummary } from "@/types/service-failure";
import type { StopType } from "@trenova/shared/types/shipment";
import type { ReactNode } from "react";

type ServiceFailureStopContextProps = {
  summary: ServiceFailureStopSummary;
  variant?: "card" | "panel" | "row";
  trailing?: ReactNode;
  className?: string;
};

type ServiceFailureStopSource = {
  type?: StopType | null;
  sequence?: number | null;
  locationId?: string | null;
  location?: {
    name?: string | null;
    code?: string | null;
    city?: string | null;
    state?: { abbreviation?: string | null } | null;
  } | null;
};

export type ServiceFailureStopSummarySource = {
  id?: string;
  shipmentId: string;
  shipmentMoveId: string;
  stopId: string;
  stopType: StopType;
  scheduledCutoff: number;
  actualArrival: number;
  gracePeriodMinutes: number;
  lateMinutes: number;
  stop?: ServiceFailureStopSource | null;
};

export function serviceFailureStopSummaryFromFailure(
  failure: ServiceFailureStopSummarySource,
): ServiceFailureStopSummary {
  const stop = failure.stop;
  const location = stop?.location ?? undefined;

  return {
    shipmentId: failure.shipmentId,
    shipmentMoveId: failure.shipmentMoveId,
    stopId: failure.stopId,
    stopSequence: stop?.sequence ?? null,
    stopType: stop?.type ?? failure.stopType,
    locationId: stop?.locationId ?? undefined,
    locationName: textOrUndefined(location?.name),
    locationCode: textOrUndefined(location?.code),
    city: textOrUndefined(location?.city),
    stateCode: textOrUndefined(location?.state?.abbreviation),
    scheduledCutoff: failure.scheduledCutoff,
    actualArrival: failure.actualArrival,
    gracePeriodMinutes: failure.gracePeriodMinutes,
    lateMinutes: failure.lateMinutes,
    serviceFailureId: failure.id,
  };
}

export function serviceFailureStopSummaryFromEvaluation(
  summary: ServiceFailureStopSummary,
): ServiceFailureStopSummary {
  return {
    shipmentId: summary.shipmentId,
    shipmentMoveId: summary.shipmentMoveId,
    stopId: summary.stopId,
    stopSequence: summary.stopSequence ?? null,
    stopType: summary.stopType,
    locationId: summary.locationId,
    locationName: textOrUndefined(summary.locationName),
    locationCode: textOrUndefined(summary.locationCode),
    city: textOrUndefined(summary.city),
    stateCode: textOrUndefined(summary.stateCode),
    scheduledCutoff: summary.scheduledCutoff ?? null,
    actualArrival: summary.actualArrival ?? null,
    gracePeriodMinutes: summary.gracePeriodMinutes ?? null,
    lateMinutes: summary.lateMinutes ?? null,
    serviceFailureId: summary.serviceFailureId,
    reason: summary.reason,
  };
}

export function ServiceFailureStopContext({
  summary,
  variant = "card",
  trailing,
  className,
}: ServiceFailureStopContextProps) {
  const normalized = serviceFailureStopSummaryFromEvaluation(summary);

  return (
    <div
      className={cn(
        "min-w-0 text-xs",
        variant === "panel" && "bg-muted/25 rounded-md border px-3 py-2",
        variant === "row" && "px-3 py-2",
        className,
      )}
    >
      <div className="flex min-w-0 items-start justify-between gap-3">
        <div className="min-w-0">
          <div className="flex min-w-0 flex-wrap items-center gap-x-2 gap-y-0.5">
            <span className="text-foreground font-medium">{formatStopTitle(normalized)}</span>
            <span className="text-muted-foreground truncate">{formatLocation(normalized)}</span>
          </div>
          <div className="text-muted-foreground mt-1 flex min-w-0 flex-wrap items-center gap-x-3 gap-y-1">
            {renderTimestamp("Cutoff", normalized.scheduledCutoff)}
            {renderTimestamp("Arrived", normalized.actualArrival)}
            {renderMinutes("Grace", normalized.gracePeriodMinutes)}
            {renderMinutes("Late", normalized.lateMinutes)}
          </div>
        </div>
        {trailing && (
          <div className="text-muted-foreground max-w-[45%] shrink-0 text-right leading-snug break-words">
            {trailing}
          </div>
        )}
      </div>
    </div>
  );
}

function formatStopTitle(summary: ServiceFailureStopSummary) {
  const stopLabel = summary.stopSequence
    ? `Stop ${summary.stopSequence}`
    : summary.stopId
      ? "Stop"
      : "Shipment";
  const typeLabel = summary.stopType
    ? (findChoice(stopTypeChoices, summary.stopType)?.label ?? summary.stopType)
    : "";

  return typeLabel ? `${stopLabel} - ${typeLabel}` : stopLabel;
}

function formatLocation(summary: ServiceFailureStopSummary) {
  const name = textOrUndefined(summary.locationName);
  const code = textOrUndefined(summary.locationCode);
  const cityState = [textOrUndefined(summary.city), textOrUndefined(summary.stateCode)]
    .filter(Boolean)
    .join(", ");

  if (name && code && cityState) return `${name} (${code}) - ${cityState}`;
  if (name && code) return `${name} (${code})`;
  if (name && cityState) return `${name} - ${cityState}`;
  if (name) return name;
  if (cityState) return cityState;
  if (summary.stopId) return summary.stopId;
  return "No stop context";
}

function renderTimestamp(label: string, timestamp?: number | null) {
  if (!timestamp || timestamp <= 0) return null;

  return (
    <span className="inline-flex min-w-0 items-center gap-1">
      <span>{label}</span>
      <HoverCardTimestamp
        timestamp={timestamp}
        className="max-w-[132px] text-[11px]"
        side="top"
        align="start"
      />
    </span>
  );
}

function renderMinutes(label: string, value?: number | null) {
  if (value === undefined || value === null) return null;

  return (
    <span>
      {translate("{0} {1}m", label, value)}
    </span>
  );
}

function textOrUndefined(value?: string | null) {
  const text = value?.trim();
  return text ? text : undefined;
}
