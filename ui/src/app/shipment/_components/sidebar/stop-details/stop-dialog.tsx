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
import { STOP_DIALOG_NOTICE_KEY } from "@/constants/env";
import { ShipmentSchema } from "@/lib/schemas/shipment-schema";
import { stopSchema } from "@/lib/schemas/stop-schema";
import { type TableSheetProps } from "@/types/data-table";
import { MoveStatus } from "@/types/move";
import { StopStatus, StopType, type Stop } from "@/types/stop";
import { faInfoCircle, faXmark } from "@fortawesome/pro-solid-svg-icons";
import { useLocalStorage } from "@uidotdev/usehooks";
import { useCallback, useEffect } from "react";
import {
  UseFieldArrayRemove,
  UseFieldArrayUpdate,
  useFormContext,
  useWatch,
} from "react-hook-form";
import { ValidationError } from "yup";
import { useLocationData } from "./queries";
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
  const { getValues, reset, setValue, control, setError, clearErrors } =
    useFormContext<ShipmentSchema>();

  const locationId = useWatch({
    control,
    name: `moves.${moveIdx}.stops.${stopIdx}.locationId`,
  });

  const { data: locationData, isLoading: isLoadingLocation } =
    useLocationData(locationId);

  const validateStop = async () => {
    // Clear existing errors only for this stop
    clearErrors(`moves.${moveIdx}.stops.${stopIdx}`);

    try {
      const formValues = getValues();
      const moveWithStop = formValues.moves?.[moveIdx];
      const stop = moveWithStop?.stops?.[stopIdx];

      if (!moveWithStop || !stop) return false;

      // Create a simplified object with only the fields we want to validate
      // This prevents location schema fields from being validated
      const stopToValidate = {
        organizationId: stop.organizationId,
        businessUnitId: stop.businessUnitId,
        locationId: stop.locationId,
        status: stop.status,
        type: stop.type,
        sequence: stop.sequence || 0,
        pieces: stop.pieces,
        weight: stop.weight,
        plannedArrival: stop.plannedArrival,
        plannedDeparture: stop.plannedDeparture,
        actualArrival: stop.actualArrival,
        actualDeparture: stop.actualDeparture,
        addressLine: stop.addressLine,
      };

      // Validate against the stopSchema directly instead of using moveSchema.validateAt
      await stopSchema.validate(stopToValidate, {
        abortEarly: false,
      });

      return true;
    } catch (error) {
      if (error instanceof ValidationError) {
        error.inner.forEach((err) => {
          const fieldPath = err.path;
          if (fieldPath) {
            // Just use the direct field path without splitting
            const fullPath = `moves.${moveIdx}.stops.${stopIdx}.${fieldPath}`;
            console.info("Setting error", fullPath, err.message);
            setError(fullPath as any, {
              type: "manual",
              message: err.message,
            });
          }
        });

        // Force a re-render to show the errors
        setValue(
          // @ts-expect-error // This is a temporary field to force a re-render
          `moves.${moveIdx}.stops.${stopIdx}._lastValidated`,
          Date.now(),
          {
            shouldValidate: false,
            shouldDirty: false,
          },
        );
      }
      return false;
    }
  };

  const handleSave = async () => {
    const isValid = await validateStop();

    if (isValid) {
      const formValues = getValues();
      const stop = formValues.moves?.[moveIdx]?.stops?.[stopIdx] as Stop;

      if (stop) {
        const updatedStop: Stop = {
          organizationId: stop?.organizationId,
          businessUnitId: stop?.businessUnitId,
          locationId: stop?.locationId,
          location: stop?.location || null,
          addressLine: stop.addressLine,
          type: stop.type || StopType.Pickup,
          status: stop.status || StopStatus.New,
          pieces: stop.pieces,
          weight: stop.weight,
          sequence: stop.sequence,
          plannedArrival: stop?.plannedArrival,
          plannedDeparture: stop?.plannedDeparture,
          actualArrival: stop?.actualArrival,
          actualDeparture: stop?.actualDeparture,
          id: isEditing ? stop.id : undefined,
          shipmentMoveId: formValues?.moves?.[moveIdx]?.id || "",
        };

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
    }
  };

  // Set the Location ID and Location
  // When the location ID is set, set the location
  useEffect(() => {
    if (!isLoadingLocation && locationId && locationData) {
      // @ts-expect-error // Location information is not required, but exists
      setValue(`moves.${moveIdx}.stops.${stopIdx}.location`, locationData);
    }
  }, [isLoadingLocation, locationId, locationData, moveIdx, setValue, stopIdx]);

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
                  <Button type="button" onClick={handleSave}>
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
    STOP_DIALOG_NOTICE_KEY,
    true,
  );

  const handleClose = () => {
    setNoticeVisible(false);
  };
  return noticeVisible ? (
    <div className="bg-blue-500/20 px-4 py-3 text-blue-500">
      <div className="flex gap-2">
        <div className="flex grow gap-3">
          <Icon
            icon={faInfoCircle}
            className="mt-0.5 shrink-0 text-blue-500"
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
          variant="ghost"
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
