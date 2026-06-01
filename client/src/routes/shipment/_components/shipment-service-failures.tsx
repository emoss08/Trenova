import { HoverCardTimestamp } from "@/components/hover-card-timestamp";
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
import type { ServiceFailure } from "@/types/service-failure";
import type { Shipment } from "@/types/shipment";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  AlertTriangleIcon,
  CheckCircle2Icon,
  InfoIcon,
  RefreshCwIcon,
  ShieldCheckIcon,
  XCircleIcon,
} from "lucide-react";
import { type ReactNode, useState } from "react";
import { toast } from "sonner";

type ShipmentServiceFailuresProps = {
  shipment?: Shipment | null;
};

type EvaluationSummary = {
  created: number;
  updated: number;
  skipped: number;
  skippedStops: Array<{
    shipmentId?: string;
    stopId?: string;
    stopSequence?: number | null;
    stopType?: string;
    reason?: string;
  }>;
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
        skippedStops: result.skippedStops,
      };
      if (result.skipped > 0) {
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

              <div className="mt-2 grid grid-cols-2 gap-2 text-xs">
                <div>
                  <span className="text-muted-foreground">Stop</span>
                  <p>{failure.stopType}</p>
                </div>
                <div>
                  <span className="text-muted-foreground">Reason</span>
                  <p>{failure.reasonCode?.label ?? "Unassigned"}</p>
                </div>
              </div>
              {failure.notes && (
                <p className="mt-2 rounded bg-muted/40 px-2 py-1.5 text-xs">{failure.notes}</p>
              )}
            </div>
          );
        })}
      </div>

      <Dialog
        open={!!evaluationSummary}
        onOpenChange={(open) => !open && setEvaluationSummary(null)}
      >
        <DialogContent className="sm:max-w-lg">
          <DialogHeader>
            <DialogTitle>Service Failure Evaluation</DialogTitle>
            <DialogDescription>
              {evaluationSummary?.created ?? 0} created, {evaluationSummary?.updated ?? 0} updated,{" "}
              {evaluationSummary?.skipped ?? 0} skipped.
            </DialogDescription>
          </DialogHeader>

          <div className="rounded-md border bg-muted/25">
            <div className="flex items-center gap-2 border-b px-3 py-2 text-sm font-medium">
              <InfoIcon className="size-4 text-amber-500" />
              Skipped Stops
            </div>
            <div className="max-h-72 overflow-y-auto">
              {evaluationSummary?.skippedStops.length ? (
                <div className="divide-y">
                  {evaluationSummary.skippedStops.map((item, index) => (
                    <div
                      key={`${item.stopId ?? item.shipmentId ?? "shipment"}-${index}`}
                      className="grid grid-cols-[minmax(0,1fr)_minmax(0,1.4fr)] gap-3 px-3 py-2 text-sm"
                    >
                      <div className="min-w-0">
                        <p className="font-medium">
                          {item.stopSequence ? `Stop ${item.stopSequence}` : "Shipment"}
                        </p>
                        <p className="truncate text-xs text-muted-foreground">
                          {item.stopType ?? item.stopId ?? item.shipmentId ?? "No stop context"}
                        </p>
                      </div>
                      <p className="text-muted-foreground">{formatSkippedReason(item.reason)}</p>
                    </div>
                  ))}
                </div>
              ) : (
                <p className="px-3 py-6 text-center text-sm text-muted-foreground">
                  No skipped stop details were returned.
                </p>
              )}
            </div>
          </div>

          <DialogFooter showCloseButton />
        </DialogContent>
      </Dialog>
    </div>
  );
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
