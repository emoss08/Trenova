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
import { STOP_DIALOG_NOTICE_KEY } from "@/constants/env";
import { ShipmentSchema } from "@/lib/schemas/shipment-schema";
import { stopSchema } from "@/lib/schemas/stop-schema";
import { type TableSheetProps } from "@/types/data-table";
import { MoveStatus } from "@/types/move";
import { StopStatus, StopType, type Stop } from "@/types/stop";
import { faInfoCircle, faXmark } from "@fortawesome/pro-solid-svg-icons";
import { useLocalStorage } from "@uidotdev/usehooks";
import { memo, useCallback, useEffect } from "react";
import {
  UseFieldArrayRemove,
  UseFieldArrayUpdate,
  useFormContext,
} from "react-hook-form";
import { z } from "zod";
import { StopDialogForm } from "./stop-dialog-form";

type StopDialogProps = TableSheetProps & {
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
  const { getValues, setValue, setError, clearErrors } =
    useFormContext<ShipmentSchema>();

  // Initialize a new stop with empty values when adding a new stop - only runs when dialog opens
  useEffect(() => {
    if (open && !isEditing) {
      // Initialize with default empty values
      const now = Math.floor(Date.now() / 1000);
      const oneHour = 3600;

      const currentValues = getValues(`moves.${moveIdx}.stops.${stopIdx}`);
      // Only set values if they're missing or this is a new stop
      if (!currentValues || !currentValues.status) {
        setValue(`moves.${moveIdx}.stops.${stopIdx}`, {
          status: StopStatus.New,
          // Provide a default type that users can change
          type: StopType.Pickup,
          sequence: stopIdx,
          locationId: "",
          addressLine: "",
          plannedArrival: now,
          plannedDeparture: now + oneHour,
          // Copy organization and business unit from the move
          organizationId: getValues().moves?.[moveIdx]?.organizationId,
          businessUnitId: getValues().moves?.[moveIdx]?.businessUnitId,
        });
      }
    }
  }, [open, isEditing, moveIdx, stopIdx, setValue, getValues]);

  const validateStop = useCallback(async () => {
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
      await stopSchema.safeParseAsync(stopToValidate);

      return true;
    } catch (error) {
      console.log("Error Type", typeof error);
      if (error instanceof z.ZodError) {
        error.errors.forEach((err) => {
          const fieldPath = err.path.join(".");
          if (fieldPath) {
            // Just use the direct field path without splitting
            const fullPath = `moves.${moveIdx}.stops.${stopIdx}.${fieldPath}`;
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
  }, [clearErrors, getValues, moveIdx, setError, setValue, stopIdx]);

  const handleSave = useCallback(async () => {
    const isValid = await validateStop();

    if (isValid) {
      const formValues = getValues();
      const stop = formValues.moves?.[moveIdx]?.stops?.[stopIdx] as Stop;

      if (stop) {
        const updatedStop: Stop = {
          organizationId: stop?.organizationId,
          businessUnitId: stop?.businessUnitId,
          locationId: stop?.locationId,
          location: stop?.location || undefined,
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

        // Always use the update function to update the move with the new stop data
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

        // Close the dialog after successful save
        onOpenChange(false);
      }
    }
  }, [
    validateStop,
    getValues,
    moveIdx,
    stopIdx,
    update,
    onOpenChange,
    isEditing,
  ]);

  const handleClose = useCallback(() => {
    onOpenChange(false);

    if (!isEditing) {
      // When adding a new stop and canceling, remove it
      remove(stopIdx);
    } else {
      // When editing an existing stop and canceling, reset to original values
      // but don't remove anything
      const originalValues = getValues();
      const moves = originalValues?.moves || [];
      const stops = moves[moveIdx]?.stops || [];

      // Only reset if we have the original stop data
      if (moves.length > moveIdx && stops.length > stopIdx) {
        const originalStop = stops[stopIdx];

        if (originalStop) {
          // Reset only this specific stop's values
          setValue(`moves.${moveIdx}.stops.${stopIdx}`, originalStop, {
            shouldValidate: false,
          });
        }
      }
    }
  }, [onOpenChange, remove, stopIdx, isEditing, getValues, moveIdx, setValue]);

  return (
    <Dialog open={open} onOpenChange={onOpenChange} modal={true}>
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
          <Button type="button" onClick={handleSave}>
            {isEditing ? "Update" : "Add"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

const StopDialogNotice = memo(function StopDialogNotice() {
  const [noticeVisible, setNoticeVisible] = useLocalStorage(
    STOP_DIALOG_NOTICE_KEY,
    true,
  );

  const handleClose = useCallback(() => {
    setNoticeVisible(false);
  }, [setNoticeVisible]);

  if (!noticeVisible) return null;

  return (
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
  );
});
