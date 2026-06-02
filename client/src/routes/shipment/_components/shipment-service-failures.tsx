import { HoverCardTimestamp } from "@/components/hover-card-timestamp";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { ColorOptionValue } from "@/components/fields/select-components";
import { usePermission } from "@/hooks/use-permission";
import { findChoice, serviceFailureStatusChoices, serviceFailureTypeChoices } from "@/lib/choices";
import { queries } from "@/lib/queries";
import { cn } from "@/lib/utils";
import { apiService } from "@/services/api";
import { Operation, Resource } from "@/types/permission";
import type { ServiceFailure, ServiceFailureStopSummary } from "@/types/service-failure";
import type { Shipment } from "@/types/shipment";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  AlertTriangleIcon,
  CheckCircle2Icon,
  CircleAlertIcon,
  InfoIcon,
  RefreshCwIcon,
  SendIcon,
  ShieldCheckIcon,
  XCircleIcon,
} from "lucide-react";
import { type ReactNode, useState } from "react";
import { toast } from "sonner";
import {
  ServiceFailureStopContext,
  serviceFailureStopSummaryFromEvaluation,
  serviceFailureStopSummaryFromFailure,
} from "../../service-failure/_components/service-failure-stop-context";

type ShipmentServiceFailuresProps = {
  shipment?: Shipment | null;
};

type EvaluationSummary = {
  created: number;
  updated: number;
  skipped: number;
  createdStops: ServiceFailureStopSummary[];
  updatedStops: ServiceFailureStopSummary[];
  skippedStops: ServiceFailureStopSummary[];
};

export default function ShipmentServiceFailures({ shipment }: ShipmentServiceFailuresProps) {
  const shipmentId = shipment?.id ?? "";
  const queryClient = useQueryClient();
  const [evaluationSummary, setEvaluationSummary] = useState<EvaluationSummary | null>(null);
  const canCreate = usePermission(Resource.ServiceFailure, Operation.Create);
  const canUpdate = usePermission(Resource.ServiceFailure, Operation.Update);
  const canApprove = usePermission(Resource.ServiceFailure, Operation.Approve);
  const canArchive = usePermission(Resource.ServiceFailure, Operation.Archive);

  const failuresQuery = useQuery({
    ...queries.serviceFailure.listByShipment(shipmentId),
    enabled: !!shipmentId,
  });

  const evaluateMutation = useMutation({
    mutationFn: () => apiService.serviceFailureService.evaluateShipment(shipmentId),
    onSuccess: (result) => {
      const summary = {
        created: result.createdIds.length,
        updated: result.updatedIds.length,
        skipped: result.skipped,
        createdStops: result.createdStops.map(serviceFailureStopSummaryFromEvaluation),
        updatedStops: result.updatedStops.map(serviceFailureStopSummaryFromEvaluation),
        skippedStops: result.skippedStops,
      };
      const hasStopRows =
        summary.createdStops.length > 0 ||
        summary.updatedStops.length > 0 ||
        summary.skippedStops.length > 0;
      if (hasStopRows || result.skipped > 0) {
        setEvaluationSummary(summary);
      } else {
        toast.success("Service failure evaluation complete", {
          description: `${summary.created} created, ${summary.updated} updated, 0 skipped.`,
        });
      }
      void queryClient.invalidateQueries(queries.serviceFailure.listByShipment(shipmentId));
      void queryClient.invalidateQueries({ queryKey: ["service-failure-list"] });
      void queryClient.invalidateQueries({ queryKey: ["shipment-list"] });
    },
  });

  const lifecycleMutation = useMutation({
    mutationFn: ({
      failure,
      action,
    }: {
      failure: ServiceFailure;
      action: "review" | "resolve" | "void";
    }) => {
      const payload = {
        shipmentId: failure.shipmentId,
        reasonCodeId: failure.reasonCodeId ?? undefined,
        version: failure.version ?? 0,
      };
      if (action === "review") {
        return apiService.serviceFailureService.review(failure.id ?? "", payload);
      }
      if (action === "resolve") {
        return apiService.serviceFailureService.resolve(failure.id ?? "", payload);
      }
      const notes = window.prompt("Enter a void reason");
      if (!notes?.trim()) return Promise.resolve(failure);
      return apiService.serviceFailureService.void(failure.id ?? "", {
        ...payload,
        notes: notes.trim(),
      });
    },
    onSuccess: () => {
      toast.success("Service failure updated");
      void queryClient.invalidateQueries({ queryKey: ["serviceFailure"] });
      void queryClient.invalidateQueries(queries.serviceFailure.listByShipment(shipmentId));
      void queryClient.invalidateQueries({ queryKey: ["service-failure-list"] });
      void queryClient.invalidateQueries({ queryKey: ["shipment-list"] });
    },
  });

  const failures = failuresQuery.data?.results ?? [];

  return (
    <div className="flex flex-col gap-3">
      <div className="flex items-center justify-between gap-2">
        <div className="flex items-center gap-2 text-sm font-medium">
          <AlertTriangleIcon className="size-4 text-amber-500" />
          {failures.length} service failure{failures.length === 1 ? "" : "s"}
        </div>
        <div className="flex items-center gap-1">
          <ActionTooltip content={evaluationTooltip(canCreate.allowed, shipmentId)}>
            <Button
              type="button"
              variant="outline"
              size="xs"
              onClick={() => evaluateMutation.mutate()}
              disabled={!canCreate.allowed || !shipmentId}
              isLoading={evaluateMutation.isPending}
              loadingText="Evaluating..."
            >
              <RefreshCwIcon className="size-3.5" />
              Evaluate
            </Button>
          </ActionTooltip>
        </div>
      </div>

      <div className="flex flex-col gap-2">
        {failures.length === 0 && (
          <div className="rounded-md border border-dashed py-8 text-center text-sm text-muted-foreground">
            No service failures recorded for this shipment.
          </div>
        )}
        {failures.map((failure) => {
          const status = findChoice(serviceFailureStatusChoices, failure.status);
          const failureType = findChoice(serviceFailureTypeChoices, failure.type);
          const terminal = failure.status === "Resolved" || failure.status === "Voided";
          const stopSummary = serviceFailureStopSummaryFromFailure(failure);

          return (
            <div
              key={failure.id}
              className={cn("rounded-md border bg-card p-3", terminal && "bg-muted/25")}
            >
              <div className="flex items-start justify-between gap-3">
                <div className="min-w-0">
                  <div className="flex flex-wrap items-center gap-2">
                    <span className="font-medium">{failure.number}</span>
                    {status && <ColorOptionValue color={status.color} value={status.label} />}
                    {failureType && (
                      <ColorOptionValue color={failureType.color} value={failureType.label} />
                    )}
                  </div>
                  <div className="mt-1 text-xs text-muted-foreground">
                    {failure.lateMinutes} minute(s) late after {failure.gracePeriodMinutes} minute
                    grace · <HoverCardTimestamp timestamp={failure.detectedAt} />
                  </div>
                </div>
                <div className="flex shrink-0 items-center gap-1">
                  <ActionTooltip
                    content={reviewTooltip(canApprove.allowed, failure, lifecycleMutation.isPending)}
                  >
                    <Button
                      type="button"
                      variant="outline"
                      size="icon-xs"
                      aria-label="Review service failure"
                      disabled={
                        !canApprove.allowed ||
                        failure.status !== "Open" ||
                        !failure.reasonCodeId ||
                        lifecycleMutation.isPending
                      }
                      onClick={() => lifecycleMutation.mutate({ failure, action: "review" })}
                    >
                      <ShieldCheckIcon className="size-3.5" />
                    </Button>
                  </ActionTooltip>
                  <ActionTooltip
                    content={resolveTooltip(
                      canUpdate.allowed,
                      failure,
                      terminal,
                      lifecycleMutation.isPending,
                    )}
                  >
                    <Button
                      type="button"
                      variant="outline"
                      size="icon-xs"
                      aria-label="Resolve service failure"
                      disabled={
                        !canUpdate.allowed ||
                        terminal ||
                        !failure.reasonCodeId ||
                        lifecycleMutation.isPending
                      }
                      onClick={() => lifecycleMutation.mutate({ failure, action: "resolve" })}
                    >
                      <CheckCircle2Icon className="size-3.5" />
                    </Button>
                  </ActionTooltip>
                  <ActionTooltip
                    content={voidTooltip(canArchive.allowed, failure, lifecycleMutation.isPending)}
                  >
                    <Button
                      type="button"
                      variant="outline"
                      size="icon-xs"
                      aria-label="Void service failure"
                      disabled={
                        !canArchive.allowed ||
                        failure.status === "Voided" ||
                        lifecycleMutation.isPending
                      }
                      onClick={() => lifecycleMutation.mutate({ failure, action: "void" })}
                    >
                      <XCircleIcon className="size-3.5" />
                    </Button>
                  </ActionTooltip>
                </div>
              </div>

              <div className="mt-2 grid gap-2 text-xs sm:grid-cols-[minmax(0,1fr)_180px]">
                <ServiceFailureStopContext summary={stopSummary} />
                <div>
                  <span className="text-muted-foreground">Reason</span>
                  <p>{failure.reasonCode?.label ?? "Unassigned"}</p>
                </div>
              </div>
              {failure.notes && (
                <p className="mt-2 rounded bg-muted/40 px-2 py-1.5 text-xs">{failure.notes}</p>
              )}
              <ServiceFailureEDI214Readiness failure={failure} />
            </div>
          );
        })}
      </div>

      <Dialog
        open={!!evaluationSummary}
        onOpenChange={(open) => !open && setEvaluationSummary(null)}
      >
        <DialogContent className="sm:max-w-2xl">
          <DialogHeader>
            <DialogTitle>Service Failure Evaluation</DialogTitle>
            <DialogDescription>
              {evaluationSummary?.created ?? 0} created, {evaluationSummary?.updated ?? 0} updated,{" "}
              {evaluationSummary?.skipped ?? 0} skipped.
            </DialogDescription>
          </DialogHeader>

          <div className="max-h-[28rem] overflow-y-auto rounded-md border bg-muted/20">
            <div className="flex items-center gap-2 border-b px-3 py-2 text-sm font-medium">
              <InfoIcon className="size-4 text-amber-500" />
              Stop Results
            </div>
            <EvaluationStopGroup
              label="Created"
              count={evaluationSummary?.created ?? 0}
              stops={evaluationSummary?.createdStops ?? []}
            />
            <EvaluationStopGroup
              label="Updated"
              count={evaluationSummary?.updated ?? 0}
              stops={evaluationSummary?.updatedStops ?? []}
            />
            <EvaluationStopGroup
              label="Skipped"
              count={evaluationSummary?.skipped ?? 0}
              stops={evaluationSummary?.skippedStops ?? []}
              renderTrailing={(item) => formatSkippedReason(item.reason)}
            />
          </div>

          <DialogFooter showCloseButton />
        </DialogContent>
      </Dialog>
    </div>
  );
}

function EvaluationStopGroup({
  label,
  count,
  stops,
  renderTrailing,
}: {
  label: string;
  count: number;
  stops: ServiceFailureStopSummary[];
  renderTrailing?: (item: ServiceFailureStopSummary) => ReactNode;
}) {
  if (count === 0 && stops.length === 0) {
    return null;
  }

  return (
    <div className="border-b last:border-b-0">
      <div className="flex items-center justify-between px-3 py-2 text-xs font-medium tracking-normal text-muted-foreground uppercase">
        <span>{label}</span>
        <span>{count}</span>
      </div>
      {stops.length ? (
        <div className="divide-y bg-background/70">
          {stops.map((item, index) => (
            <ServiceFailureStopContext
              key={`${label}-${item.serviceFailureId ?? item.stopId ?? item.shipmentId ?? index}`}
              summary={item}
              variant="row"
              trailing={renderTrailing?.(item)}
            />
          ))}
        </div>
      ) : (
        <p className="bg-background/70 px-3 py-3 text-xs text-muted-foreground">
          No stop details were returned.
        </p>
      )}
    </div>
  );
}

function ServiceFailureEDI214Readiness({ failure }: { failure: ServiceFailure }) {
  const trigger = ediReadinessTrigger(failure);
  const readinessQuery = useQuery({
    ...queries.serviceFailure.edi214Readiness(failure.id ?? "", trigger),
    enabled: !!failure.id && !!trigger,
    staleTime: 30_000,
  });

  if (!trigger || failure.status === "Voided") {
    return null;
  }

  const readiness = readinessQuery.data;
  if (readinessQuery.isLoading) {
    return (
      <div className="mt-2 flex items-center gap-2 text-xs text-muted-foreground">
        <SendIcon className="size-3.5" />
        Checking EDI 214 readiness
      </div>
    );
  }
  if (!readiness) return null;

  const blocked = readiness.action === "blocked";
  const available = readiness.action === "generated" || readiness.action === "duplicate";
  const ready = readiness.action === "skipped" && readiness.skippedReason === "ready";
  const label = ediReadinessLabel(readiness.action, readiness.skippedReason);
  const diagnostic = diagnosticMessage(readiness.diagnostics[0]);

  return (
    <div
      className={cn(
        "mt-2 flex flex-wrap items-center gap-2 rounded border px-2 py-1.5 text-xs",
        blocked
          ? "border-red-200 bg-red-50 text-red-700"
          : available || ready
            ? "border-emerald-200 bg-emerald-50 text-emerald-700"
            : "border-muted bg-muted/30 text-muted-foreground",
      )}
    >
      {blocked ? <CircleAlertIcon className="size-3.5" /> : <SendIcon className="size-3.5" />}
      <span className="font-medium">EDI 214 {trigger}</span>
      <Badge variant={blocked ? "inactive" : available || ready ? "active" : "secondary"}>
        {label}
      </Badge>
      {readiness.mandatory && <Badge variant="outline">Mandatory</Badge>}
      {readiness.messageId && (
        <span className="font-mono text-[11px]">Message {readiness.messageId}</span>
      )}
      {diagnostic && <span className="min-w-0 flex-1 truncate">{diagnostic}</span>}
    </div>
  );
}

function ediReadinessTrigger(failure: ServiceFailure): "Reviewed" | "Resolved" | undefined {
  if (failure.status === "Open") return "Reviewed";
  if (failure.status === "Reviewed" || failure.status === "Resolved") return "Resolved";
  return undefined;
}

function ediReadinessLabel(action: string, reason?: string) {
  if (action === "generated") return "Generated";
  if (action === "duplicate") return "Generated";
  if (action === "blocked") return "Blocked";
  if (reason === "ready") return "Ready";
  if (reason === "service failure 214 trigger disabled") return "Not configured";
  if (reason === "no outbound EDI partner for shipment customer") return "No partner";
  if (reason === "shipment status capability disabled") return "Capability off";
  if (reason === "ambiguous service failure 214 partner document profile") return "Ambiguous";
  return "Skipped";
}

function diagnosticMessage(value: unknown) {
  if (!value || typeof value !== "object" || !("message" in value)) {
    return undefined;
  }
  const message = (value as { message?: unknown }).message;
  return typeof message === "string" ? message : undefined;
}

function ActionTooltip({
  content,
  children,
}: {
  content: ReactNode;
  children: ReactNode;
}) {
  return (
    <Tooltip>
      <TooltipTrigger render={<span className="inline-flex" />}>{children}</TooltipTrigger>
      <TooltipContent side="top" sideOffset={8}>
        {content}
      </TooltipContent>
    </Tooltip>
  );
}

function evaluationTooltip(canEvaluate: boolean, shipmentId: string) {
  if (!shipmentId) {
    return "Open a shipment before evaluating service failures.";
  }
  if (!canEvaluate) {
    return "You do not have permission to evaluate service failures.";
  }
  return "Evaluate this shipment for late pickup or delivery service failures.";
}

function reviewTooltip(canReview: boolean, failure: ServiceFailure, pending: boolean) {
  if (pending) {
    return "A service failure action is in progress.";
  }
  if (!canReview) {
    return "You do not have permission to review service failures.";
  }
  if (failure.status !== "Open") {
    return "Only open service failures can be reviewed.";
  }
  if (!failure.reasonCodeId) {
    return "Assign a reason code before reviewing.";
  }
  return "Mark this service failure as reviewed.";
}

function resolveTooltip(
  canResolve: boolean,
  failure: ServiceFailure,
  terminal: boolean,
  pending: boolean,
) {
  if (pending) {
    return "A service failure action is in progress.";
  }
  if (!canResolve) {
    return "You do not have permission to resolve service failures.";
  }
  if (terminal) {
    return "Resolved or voided service failures cannot be resolved again.";
  }
  if (!failure.reasonCodeId) {
    return "Assign a reason code before resolving.";
  }
  return "Mark this service failure as resolved.";
}

function voidTooltip(canVoid: boolean, failure: ServiceFailure, pending: boolean) {
  if (pending) {
    return "A service failure action is in progress.";
  }
  if (!canVoid) {
    return "You do not have permission to void service failures.";
  }
  if (failure.status === "Voided") {
    return "This service failure is already voided.";
  }
  return "Void this service failure with a required reason.";
}

function formatSkippedReason(reason?: string) {
  switch (reason) {
    case "shipment canceled":
      return "The shipment is canceled.";
    case "stop canceled":
      return "The stop is canceled.";
    case "missing actual arrival":
      return "The stop does not have an actual arrival time.";
    case "missing scheduled cutoff":
      return "The stop does not have a scheduled cutoff.";
    case "count late override disabled":
      return "Count late is explicitly disabled for this stop.";
    case "policy skipped":
      return "Dispatch control policy does not evaluate this stop.";
    case "not late after grace":
      return "Actual arrival is not later than the scheduled cutoff plus grace period.";
    case "missing shipment ID":
      return "The evaluation request did not include a shipment ID.";
    default:
      return reason || "The stop did not qualify for service failure creation.";
  }
}
