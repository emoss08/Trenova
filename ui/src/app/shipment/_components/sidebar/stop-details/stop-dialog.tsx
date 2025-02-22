import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogBody,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Icon } from "@/components/ui/icons";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { ShipmentSchema } from "@/lib/schemas/shipment-schema";
import { StopSchema } from "@/lib/schemas/stop-schema";
import { type TableSheetProps } from "@/types/data-table";
import { MoveStatus } from "@/types/move";
import { StopStatus, StopType } from "@/types/stop";
import { faInfoCircle, faXmark } from "@fortawesome/pro-solid-svg-icons";
import { useLocalStorage } from "@uidotdev/usehooks";
import { useCallback } from "react";
import {
  UseFieldArrayRemove,
  UseFieldArrayUpdate,
  useFormContext,
} from "react-hook-form";
import { StopDialogForm } from "./stop-dialog-form";

type StopDialogProps = TableSheetProps & {
  stopId: string;
  isEditing: boolean;
  update: UseFieldArrayUpdate<ShipmentSchema, `moves`>;
  moveIdx: number;
  stopIdx: number;
  remove: UseFieldArrayRemove;
};

export function StopDialog({
  open,
  onOpenChange,
  isEditing,
  update,
  moveIdx,
  stopIdx,
  remove,
}: StopDialogProps) {
  const { getValues, reset } = useFormContext<ShipmentSchema>();

  const handleSave = () => {
    const formValues = getValues();
    const stop = formValues.moves?.[moveIdx]?.stops?.[stopIdx];

    if (stop?.locationId && stop?.location) {
      const updatedStop: StopSchema = {
        locationId: stop?.locationId || "",
        location: stop?.location || "",
        addressLine: stop?.addressLine || "",
        type: stop?.type || StopType.Pickup,
        status: stop?.status || StopStatus.New,
        pieces: stop?.pieces || 1,
        weight: stop?.weight || 0,
        sequence: stop?.sequence || 0,
        plannedArrival: stop?.plannedArrival || 0,
        plannedDeparture: stop?.plannedDeparture || 0,
        actualArrival: stop?.actualArrival || undefined,
        actualDeparture: stop?.actualDeparture || undefined,
        id: isEditing ? stop?.id : undefined,
        shipmentMoveId: formValues?.moves?.[moveIdx]?.id || "",
      };

      console.info("updatedStop", updatedStop);

      update(moveIdx, {
        ...formValues.moves?.[moveIdx],
        loaded: formValues.moves?.[moveIdx]?.loaded ?? false,
        sequence: formValues.moves?.[moveIdx]?.sequence ?? 0,
        stops: [
          ...(formValues.moves?.[moveIdx]?.stops || []).slice(0, stopIdx),
          updatedStop,
          ...(formValues.moves?.[moveIdx]?.stops || []).slice(stopIdx + 1),
        ],
        status: formValues.moves?.[moveIdx]?.status || MoveStatus.New,
      });

      onOpenChange(false);
    }
  };

  const handleClose = useCallback(() => {
    onOpenChange(false);

    if (!isEditing) {
      remove(stopIdx);
    } else {
      const originalValues = getValues();
      const stops = originalValues?.moves?.[moveIdx]?.stops || [];

      reset(
        {
          moves: [
            ...(originalValues?.moves || []).slice(0, moveIdx),
            {
              ...(originalValues?.moves || [])[moveIdx],
              stops: stops.slice(0, stopIdx),
            },
            ...(originalValues?.moves || []).slice(moveIdx + 1),
          ],
        },
        {
          keepValues: true,
        },
      );
    }
  }, [onOpenChange, remove, stopIdx, isEditing, reset, getValues, moveIdx]);

  return (
    <>
      <Dialog open={open} onOpenChange={onOpenChange}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{isEditing ? "Edit Stop" : "Add Stop"}</DialogTitle>
            <DialogDescription>
              {isEditing
                ? "Edit the stop details for this shipment."
                : "Add a new stop to the shipment."}
            </DialogDescription>
          </DialogHeader>
          <StopDialogNotice />
          <DialogBody>
            <StopDialogForm moveIdx={moveIdx} stopIdx={stopIdx} />
          </DialogBody>
          <DialogFooter>
            <Button type="button" variant="outline" onClick={handleClose}>
              Cancel
            </Button>
            <TooltipProvider>
              <Tooltip>
                <TooltipTrigger asChild>
                  <Button type="submit" onClick={handleSave}>
                    {isEditing ? "Update" : "Add"}
                  </Button>
                </TooltipTrigger>
                <TooltipContent className="flex items-center gap-2">
                  <kbd className="-me-1 inline-flex h-5 max-h-full items-center rounded bg-muted-foreground/60 px-1 font-[inherit] text-[0.625rem] font-medium text-foreground">
                    Ctrl
                  </kbd>
                  <kbd className="-me-1 inline-flex h-5 max-h-full items-center rounded bg-muted-foreground/60 px-1 font-[inherit] text-[0.625rem] font-medium text-foreground">
                    Enter
                  </kbd>
                  <p>to save and close the stop</p>
                </TooltipContent>
              </Tooltip>
            </TooltipProvider>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}

function StopDialogNotice() {
  const [noticeVisible, setNoticeVisible] = useLocalStorage(
    "showStopDialogNotice",
    true,
  );

  const handleClose = () => {
    setNoticeVisible(false);
  };
  return noticeVisible ? (
    <div className="bg-muted px-4 py-3 text-foreground">
      <div className="flex gap-2">
        <div className="flex grow gap-3">
          <Icon
            icon={faInfoCircle}
            className="mt-0.5 shrink-0 text-foreground"
            aria-hidden="true"
          />
          <div className="flex grow flex-col justify-between gap-2 md:flex-row">
            <span className="text-sm">
              All times are displayed in your local timezone. Please ensure
              location details are accurate for proper routing.
            </span>
          </div>
        </div>
        <Button
          variant="secondary"
          className="group -my-1.5 -me-2 size-8 shrink-0 p-0 hover:bg-transparent"
          onClick={handleClose}
          aria-label="Close banner"
        >
          <Icon
            icon={faXmark}
            className="opacity-60 transition-opacity group-hover:opacity-100"
            aria-hidden="true"
          />
        </Button>
      </div>
    </div>
  ) : null;
}
