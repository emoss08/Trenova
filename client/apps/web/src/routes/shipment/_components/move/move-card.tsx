import { CarrierAssignmentCancelDialog } from "@/components/carrier-assignment/carrier-assignment-cancel-dialog";
import { useErrorSurface } from "@/hooks/use-error-surface";
import { useEtaLabels } from "@/hooks/use-eta-labels";
import { useIdempotencyKey } from "@/hooks/use-idempotency-key";
import { shipment } from "@/lib/queries/shipment";
import { assignmentService } from "@/services/assignment";
import { carrierAssignmentService } from "@/services/carrier-assignment";
import { shipmentService } from "@/services/shipment";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  AlertTriangleIcon,
  Building07Icon,
  DotsHorizontalIcon,
  Edit02Icon,
  RouteIcon,
  Scissors01Icon,
  Trash01Icon,
  User01Icon,
  UserX01Icon,
} from "@trenova/shared/components/icons";
import { Button } from "@trenova/shared/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@trenova/shared/components/ui/dropdown-menu";
import { toast } from "@trenova/shared/components/ui/toast";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import { formatNumber } from "@trenova/shared/i18n/format";
import { defineLabels } from "@trenova/shared/i18n/labels";
import { useT } from "@trenova/shared/i18n/use-t";
import { getProfile, getRule, RULE_KEYS } from "@trenova/shared/lib/capability";
import { STOP_TYPE_LABELS } from "@trenova/shared/lib/stop-type";
import { cn } from "@trenova/shared/lib/utils";
import type {
  MoveStatus,
  Shipment,
  ShipmentMove,
  Stop,
  StopActualAction,
  StopEta,
} from "@trenova/shared/types/shipment";
import { isActiveCarrierAssignment } from "@trenova/shared/types/shipment";
import { useState } from "react";
import { useFormContext, useFormState, useWatch } from "react-hook-form";
import { AssignmentDialog } from "../assignment-dialog";
import { SplitMoveDialog } from "../shipment-split-move-dialog";
import { MoveCoverage } from "./move-coverage";
import { focusStop, moveHeadline } from "./move-headline";
import { type MoveProblem, moveProblems, problemTextInMove } from "./move-problems";
import { MoveStops, stopDescription, useStopPlace } from "./move-stops";
import { stopEtaSlackText, VERDICT_DOT } from "./stop-eta";
import { RecordStopActualDialog } from "./record-stop-actual-dialog";

const MOVE_STATUS_LABELS: Record<MoveStatus, string> = defineLabels({
  New: "New",
  Assigned: "Assigned",
  InTransit: "In transit",
  Completed: "Completed",
  Canceled: "Canceled",
});

/** The line under the headline: how the projection to the stop in focus stands, in its tone. */
function HeadlineDetail({ eta }: { eta: StopEta | undefined }) {
  const t = useT();
  const labels = useEtaLabels();
  if (!eta || eta.arrived) return null;
  const slack = stopEtaSlackText(eta.slackMinutes, t);
  return (
    <span className="text-foreground-muted text-2xs flex items-center gap-1.5">
      <span aria-hidden className={cn("size-1.5 rounded-full", VERDICT_DOT[eta.verdict])} />
      <span className="text-foreground font-medium">{labels.verdict(eta.verdict)}</span>
      {slack ? (
        <>
          <span aria-hidden>·</span>
          <span className="tabular-nums">{slack}</span>
        </>
      ) : null}
    </span>
  );
}

/**
 * What keeps the move from saving, in place of the line under the headline: how many problems
 * there are, each named with its stop and field on hover, opening the editor at the first.
 */
function MoveProblems({
  problems,
  onFix,
}: {
  problems: readonly MoveProblem[];
  onFix: () => void;
}) {
  const t = useT();
  return (
    <Tooltip>
      <TooltipTrigger
        render={
          <button
            type="button"
            data-move-problem=""
            onClick={onFix}
            className="text-danger ui-focus-ring text-2xs flex items-center gap-1 self-start rounded-sm font-medium"
          />
        }
      >
        <AlertTriangleIcon aria-hidden className="size-3" />
        {t("{0, plural, one {# problem to fix} other {# problems to fix}}", problems.length)}
      </TooltipTrigger>
      <TooltipContent side="top" className="max-w-xs">
        <ul className="space-y-1 text-xs">
          {problems.map((problem) => (
            <li key={`${problem.path}:${problem.message}`}>{problemTextInMove(problem, t)}</li>
          ))}
        </ul>
      </TooltipContent>
    </Tooltip>
  );
}

/**
 * One move of a shipment, compact: its number, status and distance with every action in one
 * menu; its stops as a route; and who covers it, in a line that opens to the carrier's details.
 * Check calls, assignment, cancelling a carrier and splitting the move happen here and write the
 * result back into the form, so the page reflects them without a reload. Anything that keeps the
 * move from saving is flagged on the card and on the stop it belongs to, and opens the editor at
 * the field to fix.
 */
export function MoveCard({
  moveIndex,
  allowPersistedMoveRemoval,
  onEdit,
  onRemove,
}: {
  moveIndex: number;
  allowPersistedMoveRemoval: boolean;
  /** Opens the move editor, on the field at the form path given when there is one. */
  onEdit: (focusPath?: string) => void;
  onRemove: () => void;
}) {
  const t = useT();

  const { control, setValue } = useFormContext<Shipment>();
  // Only this move's errors, so an error elsewhere in the shipment does not redraw it.
  const { errors } = useFormState({ control, name: `moves.${moveIndex}` });
  // The card says every problem of the move and its stops, so a server refusal on a stop field
  // lands on that stop even while the move editor, where the field is drawn, is closed.
  useErrorSurface(control, `moves.${moveIndex}`);
  const queryClient = useQueryClient();
  const { data: shipmentUIPolicy } = useQuery({ ...shipment.uiPolicy() });
  const move = useWatch({ control, name: `moves.${moveIndex}` });
  const shipmentId = useWatch({ control, name: "id" });
  const { data: shipmentEta } = useQuery({
    ...shipment.eta(shipmentId ?? ""),
    enabled: !!shipmentId,
  });
  const [assignmentOpen, setAssignmentOpen] = useState(false);
  const [cancelCarrierOpen, setCancelCarrierOpen] = useState(false);
  const [splitOpen, setSplitOpen] = useState(false);
  const [checkCall, setCheckCall] = useState<{
    stopId: string;
    action: StopActualAction;
    description: string;
  } | null>(null);

  const stops = move?.stops ?? [];
  const status: MoveStatus = move?.status ?? "New";
  const moveId = move?.id;
  const problems = moveProblems(moveIndex, errors.moves?.[moveIndex]);
  const invalid = problems.length > 0;
  const etas = new Map((shipmentEta?.stops ?? []).map((eta) => [eta.stopId, eta]));

  const carrierAssignment = isActiveCarrierAssignment(move?.carrierAssignment)
    ? (move?.carrierAssignment ?? null)
    : null;
  const hasDriverAssignment = !!move?.assignment?.id;
  const hasAssignment = hasDriverAssignment || !!carrierAssignment;
  const isTerminal = status === "Completed" || status === "Canceled";
  const assignBlockedReason = !moveId
    ? t("Save the shipment to assign a driver or carrier")
    : isTerminal
      ? t("A completed or canceled move cannot be assigned")
      : null;
  const canRemove = !moveId || allowPersistedMoveRemoval;
  const moveRemovalRule = getRule(getProfile(shipmentUIPolicy), RULE_KEYS.moveRemoval);
  const removalBlockedReason = moveRemovalRule
    ? t("Move removal is blocked by the {0} profile", moveRemovalRule.provenance.profileName)
    : t("Move removal is disabled by shipment control");
  const canUnassign =
    hasDriverAssignment && status === "Assigned" && move?.assignment?.status === "New";
  const canCancelCarrier = !!carrierAssignment && status === "Assigned";
  const canSplit = !!moveId && stops.length === 2 && (status === "New" || status === "Assigned");
  const canRecordActuals =
    !!moveId && !!carrierAssignment && (status === "Assigned" || status === "InTransit");

  const refreshShipmentLists = () => {
    void queryClient.invalidateQueries({ queryKey: ["shipment-list"] });
  };

  const unassignMutation = useMutation({
    mutationFn: () => assignmentService.unassign(moveId!),
    onSuccess: () => {
      setValue(`moves.${moveIndex}.assignment`, null);
      setValue(`moves.${moveIndex}.coverageType`, "unassigned");
      setValue(`moves.${moveIndex}.status`, "New");
      refreshShipmentLists();
    },
    onError: () => toast.error(t("Failed to unassign move")),
  });

  const cancelCarrierMutation = useMutation({
    mutationFn: (reason: string) => carrierAssignmentService.cancel(moveId!, reason),
    onSuccess: () => {
      setValue(`moves.${moveIndex}.carrierAssignment`, null);
      setValue(`moves.${moveIndex}.coverageType`, "unassigned");
      setValue(`moves.${moveIndex}.status`, "New");
      setCancelCarrierOpen(false);
      refreshShipmentLists();
    },
    onError: (error: Error) =>
      toast.error(t("Failed to cancel carrier assignment"), { description: error.message }),
  });

  const withIdempotencyKey = useIdempotencyKey();
  const checkCallMutation = useMutation({
    mutationFn: ({
      stopId,
      action,
      occurredAt,
    }: {
      stopId: string;
      action: StopActualAction;
      occurredAt?: number;
    }) => {
      const payload = { action, ...(occurredAt !== undefined ? { occurredAt } : {}) };
      return withIdempotencyKey({ moveId, stopId, ...payload }, (idempotencyKey) =>
        assignmentService.recordStopActual(moveId!, stopId, payload, {
          idempotencyKey,
        }),
      );
    },
    onSuccess: (updatedMove: ShipmentMove) => {
      setValue(`moves.${moveIndex}.status`, updatedMove.status);
      setValue(`moves.${moveIndex}.stops`, updatedMove.stops);
      setCheckCall(null);
      refreshShipmentLists();
      if (move?.shipmentId) {
        void queryClient.invalidateQueries({ queryKey: ["shipment", move.shipmentId] });
      }
    },
    onError: (error: Error) =>
      toast.error(t("Failed to record stop actual"), { description: error.message }),
  });

  const recalculateDistanceMutation = useMutation({
    mutationFn: () => shipmentService.recalculateDistance(shipmentId!),
    onSuccess: () => {
      refreshShipmentLists();
      toast.success(t("Distance recalculated"));
    },
    onError: () => toast.error(t("Failed to recalculate distance")),
  });

  const distance = move?.distance ? t("{0} mi", formatNumber(move.distance)) : null;
  const focus = focusStop(stops);
  const focusPlace = useStopPlace(focus);
  const headline = moveHeadline({
    status,
    stops,
    etas,
    placeOf: (stop) =>
      (stop === focus ? focusPlace.city || focusPlace.name : stop.location?.city) ||
      STOP_TYPE_LABELS[stop.type],
    t,
  });
  const focusEta = headline.focus?.id ? etas.get(headline.focus.id) : undefined;

  return (
    <div
      data-move-card=""
      data-invalid={invalid || undefined}
      className={cn("bg-card overflow-hidden rounded-lg border", invalid && "border-danger")}
    >
      <div className="flex items-start gap-2 pt-3 pr-2 pb-2 pl-4">
        <div className="flex min-w-0 flex-1 flex-col gap-0.5">
          <div className="text-foreground-muted text-2xs flex flex-wrap items-center gap-x-1.5">
            <span className="font-medium">{t("Move {0}", moveIndex + 1)}</span>
            <span aria-hidden>·</span>
            <span>{MOVE_STATUS_LABELS[status]}</span>
            {distance && (
              <>
                <span aria-hidden>·</span>
                {move?.distanceSource ? (
                  <Tooltip>
                    <TooltipTrigger
                      render={
                        <span
                          tabIndex={0}
                          className="ui-focus-ring cursor-help rounded-sm tabular-nums"
                        />
                      }
                    >
                      {distance}
                    </TooltipTrigger>
                    <TooltipContent side="top">
                      {t("Distance from {0}", move.distanceSource)}
                    </TooltipContent>
                  </Tooltip>
                ) : (
                  <span className="tabular-nums">{distance}</span>
                )}
              </>
            )}
            {move && !move.loaded && (
              <>
                <span aria-hidden>·</span>
                <span>{t("Empty")}</span>
              </>
            )}
          </div>
          <h3
            // A new key when the move's story changes, so the new sentence settles in.
            key={headline.title}
            className={cn(
              "animate-rise truncate text-sm font-semibold",
              headline.tone === "success" && "text-success",
              headline.tone === "muted" && "text-foreground-muted",
            )}
            title={headline.title}
          >
            {headline.title}
          </h3>
          {invalid ? (
            <MoveProblems problems={problems} onFix={() => onEdit(problems[0].path)} />
          ) : (
            <HeadlineDetail eta={focusEta} />
          )}
        </div>
        <DropdownMenu>
          <DropdownMenuTrigger
            render={
              <Button
                type="button"
                variant="quiet"
                size="icon-sm"
                aria-label={t("Move {0} actions", moveIndex + 1)}
              >
                <DotsHorizontalIcon className="size-4" />
              </Button>
            }
          />
          <DropdownMenuContent align="end" sideOffset={4} className="min-w-56">
            <DropdownMenuItem
              title={hasAssignment ? t("Reassign") : t("Assign")}
              startContent={<User01Icon className="size-3.5" />}
              description={assignBlockedReason ?? undefined}
              disabled={!!assignBlockedReason}
              onClick={() => setAssignmentOpen(true)}
            />
            {canUnassign && (
              <DropdownMenuItem
                title={t("Unassign")}
                color="danger"
                startContent={<UserX01Icon className="size-3.5" />}
                onClick={() => unassignMutation.mutate()}
              />
            )}
            {canCancelCarrier && (
              <DropdownMenuItem
                title={t("Cancel carrier assignment")}
                color="danger"
                startContent={<Building07Icon className="size-3.5" />}
                onClick={() => setCancelCarrierOpen(true)}
              />
            )}
            {canSplit && (
              <DropdownMenuItem
                title={t("Split")}
                startContent={<Scissors01Icon className="size-3.5" />}
                onClick={() => setSplitOpen(true)}
              />
            )}
            {shipmentId && (
              <DropdownMenuItem
                title={t("Recalculate distance")}
                startContent={<RouteIcon className="size-3.5" />}
                disabled={recalculateDistanceMutation.isPending}
                onClick={() => recalculateDistanceMutation.mutate()}
              />
            )}
            <DropdownMenuSeparator />
            <DropdownMenuItem
              title={t("Edit")}
              startContent={<Edit02Icon className="size-3.5" />}
              onClick={() => onEdit()}
            />
            <DropdownMenuItem
              title={t("Delete")}
              startContent={<Trash01Icon className="size-3.5" />}
              description={
                canRemove ? t("Delete this move and all associated stops") : removalBlockedReason
              }
              color="danger"
              disabled={!canRemove}
              onClick={onRemove}
            />
          </DropdownMenuContent>
        </DropdownMenu>
      </div>

      <MoveStops
        stops={stops}
        problems={problems}
        etas={etas}
        distance={distance}
        canRecordActuals={canRecordActuals}
        onEdit={(stopIndex, path) =>
          onEdit(path ?? `moves.${moveIndex}.stops.${stopIndex}.locationId`)
        }
        onCheckCall={(stop: Stop, action) => {
          if (!stop.id) return;
          setCheckCall({ stopId: stop.id, action, description: stopDescription(stop, t) });
        }}
      />

      <MoveCoverage
        key={carrierAssignment?.id ?? move?.assignment?.id ?? "uncovered"}
        carrierAssignment={carrierAssignment}
        assignment={move?.assignment}
        moveId={moveId}
        canTrack={!!moveId && !isTerminal}
        assignBlockedReason={assignBlockedReason}
        onAssign={() => setAssignmentOpen(true)}
      />

      {moveId && (
        <>
          <AssignmentDialog
            open={assignmentOpen}
            onOpenChange={setAssignmentOpen}
            moveId={moveId}
            shipmentId={shipmentId}
            existingAssignment={move?.assignment}
            existingCarrierAssignment={move?.carrierAssignment ?? null}
            onAssigned={(assignment) => {
              setValue(`moves.${moveIndex}.assignment`, assignment);
              setValue(`moves.${moveIndex}.coverageType`, "driver");
              setValue(`moves.${moveIndex}.status`, "Assigned");
              void queryClient.invalidateQueries({ queryKey: ["assignment", assignment.id] });
            }}
            onCarrierAssigned={(nextCarrierAssignment) => {
              setValue(`moves.${moveIndex}.carrierAssignment`, nextCarrierAssignment);
              setValue(`moves.${moveIndex}.coverageType`, "carrier");
              setValue(`moves.${moveIndex}.status`, "Assigned");
              refreshShipmentLists();
            }}
          />
          <CarrierAssignmentCancelDialog
            open={cancelCarrierOpen}
            onOpenChange={setCancelCarrierOpen}
            carrierName={carrierAssignment?.carrier?.name}
            isSubmitting={cancelCarrierMutation.isPending}
            onConfirm={(reason) => cancelCarrierMutation.mutate(reason)}
          />
          <RecordStopActualDialog
            open={!!checkCall}
            onOpenChange={(open) => {
              if (!open) setCheckCall(null);
            }}
            action={checkCall?.action ?? "Arrive"}
            stopDescription={checkCall?.description}
            isSubmitting={checkCallMutation.isPending}
            onConfirm={(occurredAt) => {
              if (!checkCall) return;
              checkCallMutation.mutate({
                stopId: checkCall.stopId,
                action: checkCall.action,
                occurredAt,
              });
            }}
          />
          {canSplit && (
            <SplitMoveDialog
              open={splitOpen}
              onOpenChange={setSplitOpen}
              moveId={moveId}
              currentMove={move}
              onSplit={() => {
                refreshShipmentLists();
                void queryClient.invalidateQueries({ queryKey: ["shipment", move.shipmentId] });
              }}
            />
          )}
        </>
      )}
    </div>
  );
}
