import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { formatSplitDateTime } from "@/lib/date";
import { cn } from "@/lib/utils";
import { apiService } from "@/services/api";
import { useAuthStore } from "@/stores/auth-store";
import type { MoveStatus, Shipment, Stop, StopStatus, StopType } from "@/types/shipment";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  ArrowDownIcon,
  CheckIcon,
  EllipsisVerticalIcon,
  PencilIcon,
  PlusIcon,
  ScissorsIcon,
  TrashIcon,
  TruckIcon,
  UserIcon,
  UserXIcon,
  XIcon,
} from "lucide-react";
import { useState } from "react";
import { useFormContext, useWatch } from "react-hook-form";
import { toast } from "sonner";
import { AssignmentDialog } from "../assignment-dialog";
import { SplitMoveDialog } from "../shipment-split-move-dialog";

export function MoveCard({
  moveIndex,
  allowPersistedMoveRemoval,
  onEdit,
  onRemove,
}: {
  moveIndex: number;
  allowPersistedMoveRemoval: boolean;
  onEdit: () => void;
  onRemove: () => void;
}) {
  const {
    control,
    setValue,
    formState: { errors },
  } = useFormContext<Shipment>();
  const queryClient = useQueryClient();
  const move = useWatch({ control, name: `moves.${moveIndex}` });
  const [assignmentOpen, setAssignmentOpen] = useState(false);
  const [splitOpen, setSplitOpen] = useState(false);

  const stops = move?.stops ?? [];
  const statusConfig = moveStatusConfig[move?.status ?? "New"];
  const moveErrors = errors.moves?.[moveIndex]?.stops;
  const hasId = !!move?.id;
  const hasAssignment = !!move?.assignment?.id;
  const isTerminal = move?.status === "Completed" || move?.status === "Canceled";
  const canRemove = !hasId || allowPersistedMoveRemoval;
  const canUnassign =
    hasAssignment && move?.status === "Assigned" && move?.assignment?.status === "New";
  const canSplit =
    hasId &&
    !isTerminal &&
    move?.stops?.length === 2 &&
    (move?.status === "New" || move?.status === "Assigned");

  const unassignMutation = useMutation({
    mutationFn: () => apiService.assignmentService.unassign(move.id!),
    onSuccess: () => {
      setValue(`moves.${moveIndex}.assignment`, undefined as any);
      setValue(`moves.${moveIndex}.status`, "New");
      void queryClient.invalidateQueries({ queryKey: ["shipment-list"] });
      toast.success("Move unassigned", {
        description: "The assignment has been removed from this move.",
      });
    },
    onError: () => {
      toast.error("Failed to unassign move");
    },
  });

  return (
    <div className="rounded-lg border bg-card">
      <div className="flex items-center justify-between border-b px-4 py-2.5">
        <div className="flex items-center gap-2">
          <Badge variant={statusConfig.variant}>{statusConfig.label}</Badge>
          {move?.loaded && <Badge variant="secondary">Loaded</Badge>}
          {move?.distance ? (
            <span className="text-xs text-muted-foreground">{move.distance} mi</span>
          ) : null}
        </div>
        <DropdownMenu>
          <DropdownMenuTrigger
            render={
              <Button type="button" variant="ghost" size="icon" className="size-7">
                <EllipsisVerticalIcon className="size-3.5 text-muted-foreground" />
              </Button>
            }
          />
          <DropdownMenuContent align="end" sideOffset={4} className="w-auto">
            {hasId && !isTerminal ? (
              <DropdownMenuItem
                label={hasAssignment ? "Reassign" : "Assign"}
                title={hasAssignment ? "Reassign" : "Assign"}
                startContent={<UserIcon className="size-3.5" />}
                onClick={() => setAssignmentOpen(true)}
              />
            ) : (
              <Tooltip>
                <TooltipTrigger
                  render={
                    <div className="flex cursor-not-allowed items-center px-1.5 py-1 text-sm text-muted-foreground">
                      <UserIcon className="mr-2 size-3.5" />
                      {hasAssignment ? "Reassign" : "Assign"}
                    </div>
                  }
                />
                <TooltipContent side="left" sideOffset={10}>
                  {isTerminal
                    ? "Cannot assign a completed or canceled move"
                    : "Save the shipment first to assign workers"}
                </TooltipContent>
              </Tooltip>
            )}
            {canUnassign && (
              <DropdownMenuItem
                label="Unassign"
                title="Unassign"
                color="danger"
                startContent={<UserXIcon className="size-3.5" />}
                onClick={() => unassignMutation.mutate()}
              />
            )}
            {canSplit && (
              <DropdownMenuItem
                label="Split"
                title="Split"
                startContent={<ScissorsIcon className="size-3.5" />}
                onClick={() => setSplitOpen(true)}
              />
            )}
            <DropdownMenuSeparator />
            <DropdownMenuItem
              startContent={<PencilIcon className="size-3.5" />}
              title="Edit"
              label="Edit"
              onClick={onEdit}
            />
            <DropdownMenuItem
              startContent={<TrashIcon className="size-3.5" />}
              title="Delete"
              label="Delete"
              description={
                canRemove
                  ? "Delete this move and all associated stops"
                  : "Move removal is disabled by shipment control"
              }
              color="danger"
              disabled={!canRemove}
              onClick={onRemove}
            />
          </DropdownMenuContent>
        </DropdownMenu>
      </div>

      <ScrollArea className="max-h-62.5 px-4 py-2">
        <div className="relative space-y-2 py-2">
          {stops.map((stop, stopIdx) => {
            const isLast = stopIdx === stops.length - 1;
            const nextStop = !isLast ? stops[stopIdx + 1] : undefined;
            const showConnector =
              !isLast && stopHasInfo(stop) && (nextStop ? stopHasInfo(nextStop) : false);

            const stopErrs = moveErrors?.[stopIdx];
            const stopHasErrors = !!(stopErrs && Object.keys(stopErrs).length > 0);
            const stopErrorMessages = stopHasErrors
              ? Object.entries(stopErrs as Record<string, { message?: string }>)
                  .filter(([key]) => key !== "ref" && key !== "root")
                  .map(([field, err]) => `${field}: ${err?.message ?? "invalid"}`)
              : [];

            return (
              <StopTimelineItem
                key={stopIdx}
                stop={stop}
                isLast={isLast}
                moveStatus={move?.status ?? "New"}
                prevStopStatus={stopIdx > 0 ? stops[stopIdx - 1]?.status : undefined}
                showConnector={showConnector}
                hasErrors={stopHasErrors}
                errorMessages={stopErrorMessages}
              />
            );
          })}
        </div>
      </ScrollArea>
      <AssignmentDetails assignmentId={move?.assignment?.id} />
      {hasId && (
        <>
          <AssignmentDialog
            open={assignmentOpen}
            onOpenChange={setAssignmentOpen}
            moveId={move.id!}
            existingAssignment={move?.assignment}
            onAssigned={(assignment) => {
              setValue(`moves.${moveIndex}.assignment`, assignment);
              setValue(`moves.${moveIndex}.status`, "Assigned");
              void queryClient.invalidateQueries({
                queryKey: ["assignment", assignment.id],
              });
            }}
          />
          {canSplit && (
            <SplitMoveDialog
              open={splitOpen}
              onOpenChange={setSplitOpen}
              moveId={move.id!}
              currentMove={move}
              onSplit={() => {
                void queryClient.invalidateQueries({
                  queryKey: ["shipment-list"],
                });
                void queryClient.invalidateQueries({
                  queryKey: ["shipment", move.shipmentId],
                });
              }}
            />
          )}
        </>
      )}
    </div>
  );
}
const moveStatusConfig: Record<
  MoveStatus,
  {
    label: string;
    variant: "secondary" | "info" | "orange" | "active" | "inactive";
  }
> = {
  New: { label: "New", variant: "secondary" },
  Assigned: { label: "Assigned", variant: "info" },
  InTransit: { label: "In Transit", variant: "orange" },
  Completed: { label: "Completed", variant: "active" },
  Canceled: { label: "Canceled", variant: "inactive" },
};

const stopTypeLabels: Record<StopType, string> = {
  Pickup: "Pickup",
  Delivery: "Delivery",
  SplitPickup: "Split Pickup",
  SplitDelivery: "Split Delivery",
};

const stopStatusBgColor: Record<StopStatus, string> = {
  New: "bg-purple-500",
  InTransit: "bg-blue-500",
  Completed: "bg-green-500",
  Canceled: "bg-red-500",
};

const stopStatusLineColor: Record<StopStatus, string> = {
  New: "bg-purple-500",
  InTransit: "bg-blue-500",
  Completed: "bg-green-500",
  Canceled: "bg-red-500",
};

function getStatusIcon(status: StopStatus, isLast: boolean, moveStatus: MoveStatus) {
  if (isLast && moveStatus === "Completed") return CheckIcon;
  switch (status) {
    case "New":
      return PlusIcon;
    case "InTransit":
      return TruckIcon;
    case "Completed":
      return ArrowDownIcon;
    case "Canceled":
      return XIcon;
  }
}

const transitionGradients: Record<string, string> = {
  "New-InTransit": "bg-linear-to-b from-purple-500 to-blue-500",
  "New-Completed": "bg-linear-to-b from-purple-500 to-green-500",
  "New-Canceled": "bg-linear-to-b from-purple-500 to-red-500",
  "InTransit-New": "bg-linear-to-b from-blue-500 to-purple-500",
  "InTransit-Completed": "bg-linear-to-b from-blue-500 to-green-500",
  "InTransit-Canceled": "bg-linear-to-b from-blue-500 to-red-500",
  "Completed-New": "bg-linear-to-b from-green-500 to-purple-500",
  "Completed-InTransit": "bg-linear-to-b from-green-500 to-blue-500",
  "Completed-Canceled": "bg-linear-to-b from-green-500 to-red-500",
  "Canceled-New": "bg-linear-to-b from-red-500 to-purple-500",
  "Canceled-InTransit": "bg-linear-to-b from-red-500 to-blue-500",
  "Canceled-Completed": "bg-linear-to-b from-red-500 to-green-500",
};

function getConnectorLineClasses(status: StopStatus, prevStatus?: StopStatus): string {
  if (status === "InTransit") {
    return "bg-linear-to-b from-blue-500 to-transparent bg-[length:100%_200%] animate-flow-down";
  }
  if (prevStatus && prevStatus !== status) {
    const key = `${prevStatus}-${status}`;
    return transitionGradients[key] ?? stopStatusLineColor[status];
  }
  return stopStatusLineColor[status];
}

function LocationDisplay({ locationId, stopType }: { locationId: string; stopType: StopType }) {
  const { data: location } = useQuery({
    queryKey: ["location", "selectOption", locationId],
    queryFn: () => apiService.locationService.getOption(locationId),
    enabled: !!locationId,
    staleTime: 5 * 60 * 1000,
  });

  if (!location) return null;

  return (
    <>
      <div className="flex items-center gap-1.5">
        {location.addressLine1 && <span className="truncate text-xs">{location.addressLine1}</span>}
        <span className="text-xs whitespace-nowrap text-muted-foreground">
          ({stopTypeLabels[stopType]})
        </span>
      </div>
      <p className="truncate text-xs text-muted-foreground">
        {location.city}
        {location.state?.abbreviation && `, ${location.state.abbreviation}`} {location.postalCode}
      </p>
    </>
  );
}

function stopHasInfo(stop: Stop): boolean {
  return !!(
    stop.locationId ||
    stop.addressLine ||
    (stop.scheduledWindowStart && stop.scheduledWindowStart > 0)
  );
}

function StopTimelineItem({
  stop,
  isLast,
  moveStatus,
  prevStopStatus,
  showConnector,
  hasErrors,
  errorMessages,
}: {
  stop: Stop;
  isLast: boolean;
  moveStatus: MoveStatus;
  prevStopStatus?: StopStatus;
  showConnector: boolean;
  hasErrors?: boolean;
  errorMessages?: string[];
}) {
  const user = useAuthStore((state) => state.user);
  const userTimezone = user?.timezone || "auto";
  const userTimeFormat = user?.timeFormat || "12-hour";
  const status = stop.status ?? "New";
  const Icon = getStatusIcon(status, isLast, moveStatus);
  const hasInfo = stopHasInfo(stop);
  const scheduled =
    stop.scheduledWindowStart && stop.scheduledWindowStart > 0
      ? formatSplitDateTime(stop.scheduledWindowStart, userTimeFormat, userTimezone)
      : null;

  return (
    <div
      className={cn(
        "relative flex h-15 items-start gap-4 rounded-lg px-3 pt-2",
        hasErrors ? "border border-destructive bg-destructive/10" : "bg-muted",
      )}
    >
      {showConnector && (
        <div
          className={`absolute top-5 left-33.75 z-10 w-0.5 ${getConnectorLineClasses(status, prevStopStatus)}`}
          style={{ height: "80px" }}
        />
      )}

      <div className="flex w-24 shrink-0 flex-col items-end pt-0.5">
        {scheduled ? (
          <>
            <span className="text-xs font-medium text-primary">{scheduled.date}</span>
            <span className="text-xs text-muted-foreground">{scheduled.time}</span>
          </>
        ) : (
          <span className="text-xs text-muted-foreground">--</span>
        )}
      </div>

      <div className="relative z-10">
        <div
          className={`mt-0.5 flex size-6 shrink-0 items-center justify-center rounded-full ${stopStatusBgColor[status]}`}
        >
          <Icon className="size-3 text-white" />
        </div>
        {hasErrors && errorMessages && errorMessages.length > 0 && (
          <Tooltip>
            <TooltipTrigger
              render={
                <div className="absolute -top-1 -right-1 flex size-3 cursor-help items-center justify-center rounded-full bg-destructive">
                  <span className="text-[8px] font-bold text-red-200">!</span>
                </div>
              }
            />
            <TooltipContent side="top" className="max-w-xs">
              <div className="space-y-1">
                <p className="text-xs font-semibold">Validation Errors:</p>
                {errorMessages.map((msg, idx) => (
                  <p key={idx} className="text-xs">
                    • {msg}
                  </p>
                ))}
              </div>
            </TooltipContent>
          </Tooltip>
        )}
      </div>

      <div className="min-w-0 flex-1 pt-0.5">
        {hasInfo ? (
          <>
            {stop.locationId ? (
              <LocationDisplay locationId={stop.locationId} stopType={stop.type} />
            ) : stop.addressLine ? (
              <>
                <div className="flex items-center gap-1.5">
                  <span className="truncate text-xs">{stop.addressLine}</span>
                  <span className="text-xs whitespace-nowrap text-muted-foreground">
                    ({stopTypeLabels[stop.type]})
                  </span>
                </div>
              </>
            ) : (
              <span className="text-xs whitespace-nowrap text-muted-foreground">
                ({stopTypeLabels[stop.type]})
              </span>
            )}
          </>
        ) : hasErrors ? (
          <div className="flex flex-col gap-0.5">
            <span className="text-xs text-destructive">
              Error in {stopTypeLabels[stop.type]} stop
            </span>
            <span className="text-xs text-muted-foreground">Click to edit and fix errors</span>
          </div>
        ) : (
          <span className="text-xs text-muted-foreground">
            Enter {stopTypeLabels[stop.type]} Information
          </span>
        )}
      </div>
    </div>
  );
}

function AssignmentDetails({ assignmentId }: { assignmentId?: string }) {
  const { data: assignment, isLoading } = useQuery({
    queryKey: ["assignment", assignmentId],
    queryFn: () => apiService.assignmentService.get(assignmentId!),
    enabled: !!assignmentId,
    staleTime: 5 * 60 * 1000,
  });

  if (!assignmentId) return null;

  if (isLoading) {
    return (
      <div className="rounded-b-md border-t bg-muted p-3">
        <p className="text-2xs text-muted-foreground">Loading assignment...</p>
      </div>
    );
  }

  if (!assignment) return null;

  const { tractor, trailer, primaryWorker, secondaryWorker } = assignment;

  if (!tractor && !trailer && !primaryWorker && !secondaryWorker) return null;

  return (
    <div className="grid grid-cols-2 gap-x-6 gap-y-2 rounded-b-md border-t bg-muted p-3">
      {tractor && (
        <div>
          <p className="text-2xs text-muted-foreground">Tractor</p>
          <p className="text-xs font-medium">{tractor.code}</p>
        </div>
      )}
      {trailer && (
        <div>
          <p className="text-2xs text-muted-foreground">Trailer</p>
          <p className="text-xs font-medium">{trailer.code}</p>
        </div>
      )}
      {primaryWorker && (
        <div>
          <p className="text-2xs text-muted-foreground">Primary Worker</p>
          <p className="text-xs font-medium">
            {`${primaryWorker.firstName} ${primaryWorker.lastName}`}
          </p>
        </div>
      )}
      {secondaryWorker && (
        <div>
          <p className="text-2xs text-muted-foreground">Secondary Worker</p>
          <p className="text-xs font-medium">
            {`${secondaryWorker.firstName} ${secondaryWorker.lastName}`}
          </p>
        </div>
      )}
    </div>
  );
}
