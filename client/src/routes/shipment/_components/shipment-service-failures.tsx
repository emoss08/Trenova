import { HoverCardTimestamp } from "@/components/hover-card-timestamp";
import { Button } from "@/components/ui/button";
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
  RefreshCwIcon,
  ShieldCheckIcon,
  XCircleIcon,
} from "lucide-react";
import { toast } from "sonner";

type ShipmentServiceFailuresProps = {
  shipment?: Shipment | null;
};

export default function ShipmentServiceFailures({ shipment }: ShipmentServiceFailuresProps) {
  const shipmentId = shipment?.id ?? "";
  const queryClient = useQueryClient();
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
      toast.success("Service failure evaluation complete", {
        description: `${result.createdIds.length} created, ${result.updatedIds.length} updated, ${result.skipped} skipped.`,
      });
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
                  <Button
                    type="button"
                    variant="outline"
                    size="icon-xs"
                    title="Review"
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
                  <Button
                    type="button"
                    variant="outline"
                    size="icon-xs"
                    title="Resolve"
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
                  <Button
                    type="button"
                    variant="outline"
                    size="icon-xs"
                    title="Void"
                    disabled={
                      !canArchive.allowed ||
                      failure.status === "Voided" ||
                      lifecycleMutation.isPending
                    }
                    onClick={() => lifecycleMutation.mutate({ failure, action: "void" })}
                  >
                    <XCircleIcon className="size-3.5" />
                  </Button>
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
    </div>
  );
}
