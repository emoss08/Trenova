import { ServiceFailureReasonCodeAutocompleteField } from "@/components/autocomplete-fields";
import { HoverCardTimestamp } from "@/components/hover-card-timestamp";
import { Button } from "@/components/ui/button";
import { Form, FormControl, FormGroup } from "@/components/ui/form";
import { SelectField } from "@/components/fields/select-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { ColorOptionValue } from "@/components/fields/select-components";
import { usePermission } from "@/hooks/use-permission";
import { findChoice, serviceFailureStatusChoices, serviceFailureTypeChoices } from "@/lib/choices";
import { queries } from "@/lib/queries";
import { cn } from "@/lib/utils";
import { apiService } from "@/services/api";
import { Operation, Resource } from "@/types/permission";
import type { ServiceFailure, ServiceFailureManualCreate } from "@/types/service-failure";
import { serviceFailureManualCreateSchema } from "@/types/service-failure";
import type { Shipment, StopType } from "@/types/shipment";
import { zodResolver } from "@hookform/resolvers/zod";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  AlertTriangleIcon,
  CheckCircle2Icon,
  PlusIcon,
  RefreshCwIcon,
  ShieldCheckIcon,
  XCircleIcon,
} from "lucide-react";
import { useEffect, useState } from "react";
import { FormProvider, type Resolver, useForm } from "react-hook-form";
import { toast } from "sonner";

type ShipmentServiceFailuresProps = {
  shipment?: Shipment | null;
};

type StopContext = {
  stopId: string;
  shipmentMoveId: string;
  label: string;
  type: StopType;
  failureType: ServiceFailureManualCreate["type"];
};

const defaultCreateValues: ServiceFailureManualCreate = {
  shipmentId: "",
  shipmentMoveId: "",
  stopId: "",
  reasonCodeId: "",
  type: "LateDelivery",
  notes: "",
  internalNotes: "",
  x12StatusCodeOverride: "",
  x12ReasonCodeOverride: "",
  x12ExceptionCode: "",
};

function failureTypeForStop(stopType: StopType): ServiceFailureManualCreate["type"] {
  return stopType === "Pickup" || stopType === "SplitPickup" ? "LatePickup" : "LateDelivery";
}

function buildStopContexts(shipment?: Shipment | null): StopContext[] {
  if (!shipment?.moves) return [];

  return shipment.moves.flatMap((move, moveIndex) => {
    const stops: StopContext[] = [];
    if (!move.id) return stops;

    for (const stop of move.stops ?? []) {
      if (!stop.id) continue;

      const location = stop.location?.name ?? stop.locationId ?? "Unknown location";
      stops.push({
        stopId: stop.id,
        shipmentMoveId: move.id,
        type: stop.type,
        failureType: failureTypeForStop(stop.type),
        label: `Move ${moveIndex + 1} · ${stop.type} #${stop.sequence + 1} · ${location}`,
      });
    }

    return stops;
  });
}

export default function ShipmentServiceFailures({ shipment }: ShipmentServiceFailuresProps) {
  const shipmentId = shipment?.id ?? "";
  const queryClient = useQueryClient();
  const canCreate = usePermission(Resource.ServiceFailure, Operation.Create);
  const canUpdate = usePermission(Resource.ServiceFailure, Operation.Update);
  const canApprove = usePermission(Resource.ServiceFailure, Operation.Approve);
  const canArchive = usePermission(Resource.ServiceFailure, Operation.Archive);
  const [showManualForm, setShowManualForm] = useState(false);
  const stopContexts = buildStopContexts(shipment);
  const stopOptions = stopContexts.map((stop) => ({
    label: stop.label,
    value: stop.stopId,
  }));

  const failuresQuery = useQuery({
    ...queries.serviceFailure.listByShipment(shipmentId),
    enabled: !!shipmentId,
  });

  const form = useForm<ServiceFailureManualCreate>({
    resolver: zodResolver(serviceFailureManualCreateSchema) as Resolver<ServiceFailureManualCreate>,
    defaultValues: defaultCreateValues,
  });
  const { control, handleSubmit, reset, setValue } = form;

  useEffect(() => {
    if (!shipmentId) return;
    const firstStop = stopContexts[0];
    reset({
      ...defaultCreateValues,
      shipmentId,
      shipmentMoveId: firstStop?.shipmentMoveId ?? "",
      stopId: firstStop?.stopId ?? "",
      type: firstStop?.failureType ?? "LateDelivery",
    });
  }, [reset, shipmentId, stopContexts]);

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

  const createMutation = useMutation({
    mutationFn: (values: ServiceFailureManualCreate) =>
      apiService.serviceFailureService.createManual(values),
    onSuccess: () => {
      toast.success("Manual service failure recorded");
      setShowManualForm(false);
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
        version: failure.version ?? 0,
      };
      if (action === "review") {
        return apiService.serviceFailureService.review(failure.id ?? "", payload);
      }
      if (action === "resolve") {
        return apiService.serviceFailureService.resolve(failure.id ?? "", payload);
      }
      return apiService.serviceFailureService.void(failure.id ?? "", payload);
    },
    onSuccess: () => {
      toast.success("Service failure updated");
      void queryClient.invalidateQueries(queries.serviceFailure.listByShipment(shipmentId));
      void queryClient.invalidateQueries({ queryKey: ["service-failure-list"] });
      void queryClient.invalidateQueries({ queryKey: ["shipment-list"] });
    },
  });

  const onStopChange = (stopId: string) => {
    const stop = stopContexts.find((candidate) => candidate.stopId === stopId);
    if (!stop) return;
    setValue("shipmentMoveId", stop.shipmentMoveId, { shouldDirty: true });
    setValue("type", stop.failureType, { shouldDirty: true });
  };

  const onSubmit = (values: ServiceFailureManualCreate) => {
    const stop = stopContexts.find((candidate) => candidate.stopId === values.stopId);
    if (!stop) return;
    createMutation.mutate({
      ...values,
      shipmentId,
      shipmentMoveId: stop.shipmentMoveId,
      type: stop.failureType,
    });
  };

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
          <Button
            type="button"
            size="xs"
            onClick={() => setShowManualForm((value) => !value)}
            disabled={!canCreate.allowed || stopContexts.length === 0}
          >
            <PlusIcon className="size-3.5" />
            Manual
          </Button>
        </div>
      </div>

      {showManualForm && (
        <div className="rounded-md border bg-muted/20 p-3">
          <FormProvider {...form}>
            <Form onSubmit={handleSubmit(onSubmit)}>
              <FormGroup cols={2}>
                <FormControl cols="full">
                  <SelectField
                    control={control}
                    name="stopId"
                    label="Stop"
                    placeholder="Select Stop"
                    options={stopOptions}
                    onValueChange={onStopChange}
                    rules={{ required: true }}
                  />
                </FormControl>
                <FormControl cols="full">
                  <ServiceFailureReasonCodeAutocompleteField
                    control={control}
                    name="reasonCodeId"
                    label="Reason Code"
                    placeholder="Select Reason Code"
                    rules={{ required: true }}
                  />
                </FormControl>
                <FormControl cols="full">
                  <TextareaField
                    control={control}
                    name="notes"
                    label="Operations Notes"
                    placeholder="Add operational context"
                  />
                </FormControl>
              </FormGroup>
              <div className="mt-3 flex justify-end gap-2">
                <Button
                  type="button"
                  variant="outline"
                  size="xs"
                  onClick={() => setShowManualForm(false)}
                >
                  Cancel
                </Button>
                <Button
                  type="submit"
                  size="xs"
                  isLoading={createMutation.isPending}
                  loadingText="Recording..."
                >
                  Record Failure
                </Button>
              </div>
            </Form>
          </FormProvider>
        </div>
      )}

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
                    disabled={!canUpdate.allowed || terminal || lifecycleMutation.isPending}
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
