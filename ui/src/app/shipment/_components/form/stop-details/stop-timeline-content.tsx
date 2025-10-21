import { StopDialog } from "@/app/shipment/_components/form/stop-details/stop-dialog";
import {
  ContextMenu,
  ContextMenuContent,
  ContextMenuItem,
  ContextMenuTrigger,
} from "@/components/ui/context-menu";
import type { MoveSchema } from "@/lib/schemas/move-schema";
import { type ShipmentSchema } from "@/lib/schemas/shipment-schema";
import type { StopSchema } from "@/lib/schemas/stop-schema";
import { useCallback, useState } from "react";
import { useFormContext } from "react-hook-form";
import { toast } from "sonner";
import { StopTimelineItem } from "./stop-timeline-item";

export function StopTimeline({
  stop,
  nextStop,
  isLast,
  moveStatus,
  moveIdx,
  stopIdx,
  prevStopStatus,
}: {
  stop: StopSchema;
  nextStop: StopSchema | null;
  isLast: boolean;
  moveStatus: MoveSchema["status"];
  moveIdx: number;
  stopIdx: number;
  prevStopStatus?: StopSchema["status"];
}) {
  const {
    setValue,
    getValues,
    formState: { errors },
    watch,
  } = useFormContext<ShipmentSchema>();

  const [isDialogOpen, setIsDialogOpen] = useState(false);

  const watchedStop = watch(`moves.${moveIdx}.stops.${stopIdx}`);
  const watchedNextStop = watch(`moves.${moveIdx}.stops.${stopIdx + 1}`);

  const currentStop = watchedStop || stop;
  const currentNextStop = watchedNextStop || nextStop;

  const stopErrors = errors.moves?.[moveIdx]?.stops?.[stopIdx];
  const hasErrors = !!(stopErrors && Object.keys(stopErrors).length > 0);

  const errorMessages: string[] = hasErrors
    ? Object.entries(stopErrors)
        .map(([field, error]) => {
          if (error && typeof error === "object" && "message" in error) {
            return `${field}: ${error.message}`;
          }
          return null;
        })
        .filter((msg): msg is string => msg !== null)
    : [];

  const hasStopInfo =
    currentStop.location?.addressLine1 || currentStop.plannedArrival;

  const hasActualDates =
    currentStop.actualArrival || currentStop.actualDeparture;

  const nextStopHasInfo =
    currentNextStop?.location?.addressLine1 || currentNextStop?.plannedArrival;

  const openEditDialog = useCallback(() => {
    setIsDialogOpen(true);
  }, []);

  const handleDialogChange = useCallback((open: boolean) => {
    setIsDialogOpen(open);
  }, []);

  const handleRevert = useCallback(() => {
    const actualArrival = getValues(
      `moves.${moveIdx}.stops.${stopIdx}.actualArrival`,
    );
    const actualDeparture = getValues(
      `moves.${moveIdx}.stops.${stopIdx}.actualDeparture`,
    );

    const hasActualDates = actualArrival || actualDeparture;

    if (!hasActualDates) {
      toast.error(
        "Cannot revert stop with no actual arrival or departure dates",
      );
      return;
    }

    setValue(`moves.${moveIdx}.stops.${stopIdx}.actualArrival`, undefined, {
      shouldDirty: true,
    });
    setValue(`moves.${moveIdx}.stops.${stopIdx}.actualDeparture`, undefined, {
      shouldDirty: true,
    });
  }, [getValues, moveIdx, setValue, stopIdx]);

  return (
    <>
      <ContextMenu>
        <ContextMenuTrigger asChild>
          <StopTimelineItem
            stop={currentStop}
            hasErrors={hasErrors}
            errorMessages={errorMessages}
            nextStopHasInfo={nextStopHasInfo}
            isLast={isLast}
            moveStatus={moveStatus}
            prevStopStatus={prevStopStatus}
          />
        </ContextMenuTrigger>
        <ContextMenuContent>
          <ContextMenuItem onClick={openEditDialog}>
            <StopContextMenuItem
              title="Edit"
              description="Edit the stop information"
            />
          </ContextMenuItem>
          {hasActualDates && (
            <ContextMenuItem onClick={handleRevert}>
              <StopContextMenuItem
                title="Revert"
                description="Revert and clear the stop arrival date and times"
              />
            </ContextMenuItem>
          )}
          {hasActualDates && hasStopInfo && (
            <ContextMenuItem>
              <StopContextMenuItem
                title="Cancel"
                description="Cancel the stop and clear the stop arrival date and times"
              />
            </ContextMenuItem>
          )}
          <ContextMenuItem variant="destructive">
            <StopContextMenuItem
              title="Remove"
              description="Remove the stop from the movement"
            />
          </ContextMenuItem>
        </ContextMenuContent>
      </ContextMenu>
      {isDialogOpen && (
        <StopDialog
          open={isDialogOpen}
          onOpenChange={handleDialogChange}
          moveIdx={moveIdx}
          stopIdx={stopIdx}
        />
      )}
    </>
  );
}

function StopContextMenuItem({
  title,
  description,
}: {
  title: string;
  description: string;
}) {
  return (
    <div className="flex flex-col">
      <div className="flex items-center gap-2">
        <span>{title}</span>
      </div>
      <p className="text-xs text-muted-foreground">{description}</p>
    </div>
  );
}
