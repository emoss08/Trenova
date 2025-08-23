/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import { Button, FormSaveButton } from "@/components/ui/button";
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
import { useLocalStorage } from "@/hooks/use-local-storage";
import type { ShipmentSchema } from "@/lib/schemas/shipment-schema";
import {
  stopSchema,
  StopStatus,
  StopType,
  type StopSchema,
} from "@/lib/schemas/stop-schema";
import { type TableSheetProps } from "@/types/data-table";
import { faInfoCircle, faXmark } from "@fortawesome/pro-solid-svg-icons";
import { memo, useCallback, useEffect, useState } from "react";
import { useFieldArray, useFormContext } from "react-hook-form";
import { toast } from "sonner";
import { StopDialogForm } from "./stop-dialog-form";

type StopDialogProps = TableSheetProps & {
  moveIdx: number;
  stopIdx: number;
};

export function StopDialog({
  open,
  onOpenChange,
  moveIdx,
  stopIdx,
}: StopDialogProps) {
  const { getValues, setValue, trigger } = useFormContext<ShipmentSchema>();
  const [isSubmitting, setIsSubmitting] = useState(false);

  // Use field array for reactive updates
  const { append } = useFieldArray({
    name: `moves.${moveIdx}.stops`,
  });

  // Check if we're editing an existing stop or adding a new one
  const currentStops = getValues(`moves.${moveIdx}.stops`) || [];
  const isEditing = stopIdx < currentStops.length;

  // Initialize default values for new stops
  useEffect(() => {
    if (open && !isEditing) {
      const now = Math.floor(Date.now() / 1000);
      const oneHour = 3600;

      // Initialize new stop with default values directly in the parent form
      const newStop: StopSchema = {
        status: StopStatus.enum.New,
        actualArrival: undefined,
        actualDeparture: undefined,
        pieces: 0,
        weight: 0,
        type: StopType.enum.Pickup,
        sequence: stopIdx,
        locationId: "",
        addressLine: "",
        plannedArrival: now,
        plannedDeparture: now + oneHour,
        organizationId: getValues().moves?.[moveIdx]?.organizationId || "",
        businessUnitId: getValues().moves?.[moveIdx]?.businessUnitId || "",
        location: null, // Required field
      };

      // Temporarily add the stop to the form for editing
      append(newStop);
    }
  }, [open, isEditing, getValues, moveIdx, stopIdx, append]);

  const handleClose = useCallback(() => {
    // Trigger validation for the stop to ensure errors are shown in timeline
    const stopPath = `moves.${moveIdx}.stops.${stopIdx}` as const;
    trigger(stopPath);

    // If we were adding a new stop and canceling, remove the temporary stop
    if (!isEditing) {
      const stops = getValues(`moves.${moveIdx}.stops`) || [];
      if (stops.length > stopIdx) {
        // Remove the temporarily added stop
        const newStops = [...stops];
        newStops.splice(stopIdx, 1);
        setValue(`moves.${moveIdx}.stops`, newStops);
      }
    }

    onOpenChange(false);
  }, [onOpenChange, moveIdx, stopIdx, isEditing, getValues, setValue, trigger]);

  const handleSave = useCallback(async () => {
    setIsSubmitting(true);

    try {
      // Validate only the stop fields (field-level validation)
      const stopPath = `moves.${moveIdx}.stops.${stopIdx}` as const;
      const isValid = await trigger(stopPath);

      if (!isValid) {
        toast.error("Please fix the validation errors", {
          description: "Check the form for errors and try again",
        });
        return;
      }

      // Get the validated stop data
      const stopData = getValues(stopPath);

      if (!stopData) {
        toast.error("Stop data not found");
        return;
      }

      // Additional validation with zod schema
      const validationResult = stopSchema.safeParse(stopData);

      if (!validationResult.success) {
        toast.error("Please fix the validation errors", {
          description:
            validationResult.error.issues[0]?.message ||
            "Check the form for errors",
        });
        return;
      }

      // If we're adding a new stop, it's already been appended
      // If we're editing, the changes are already in the form
      // Just close the dialog
      onOpenChange(false);

      // Success message
      toast.success(
        isEditing ? "Stop updated successfully" : "Stop added successfully",
        {
          description: isEditing
            ? "The stop details have been updated"
            : "The new stop has been added to the shipment",
        },
      );
    } catch (error) {
      console.error("Error saving stop:", error);
      toast.error("Failed to save stop", {
        description: "An error occurred while saving the stop details",
      });
    } finally {
      setIsSubmitting(false);
    }
  }, [moveIdx, stopIdx, trigger, getValues, isEditing, onOpenChange]);

  return (
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
          <FormSaveButton
            type="button"
            onClick={handleSave}
            isSubmitting={isSubmitting}
            title={isEditing ? "Update Stop" : "Add Stop"}
            text={isEditing ? "Update" : "Add"}
          />
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
