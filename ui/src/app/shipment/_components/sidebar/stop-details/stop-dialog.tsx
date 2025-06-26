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
import type { ShipmentSchema } from "@/lib/schemas/shipment-schema";
import { stopSchema, type StopSchema } from "@/lib/schemas/stop-schema";
import { type TableSheetProps } from "@/types/data-table";
import { StopStatus, StopType } from "@/types/stop";
import { faInfoCircle, faXmark } from "@fortawesome/pro-solid-svg-icons";
import { useLocalStorage } from "@uidotdev/usehooks";
import { memo, useCallback, useEffect, useState } from "react";
import {
  FormProvider,
  useFieldArray,
  useForm,
  useFormContext,
} from "react-hook-form";
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
  const {
    getValues,
    clearErrors,
    formState: { errors },
  } = useFormContext<ShipmentSchema>();
  const [isSubmitting, setIsSubmitting] = useState(false);

  // Use field array for reactive updates
  const { append, update } = useFieldArray({
    name: `moves.${moveIdx}.stops`,
  });

  // Check if we're editing an existing stop or adding a new one
  const currentStops = getValues(`moves.${moveIdx}.stops`) || [];
  const isEditing = stopIdx < currentStops.length;

  // Create a local form for the stop dialog
  const localForm = useForm<{ stop: StopSchema }>({
    defaultValues: {
      stop: {
        status: StopStatus.New,
        actualArrival: undefined,
        actualDeparture: undefined,
        pieces: 0,
        weight: 0,
        type: StopType.Pickup,
        sequence: stopIdx,
        locationId: "",
        addressLine: "",
        plannedArrival: Math.floor(Date.now() / 1000),
        plannedDeparture: Math.floor(Date.now() / 1000) + 3600,
        organizationId: "",
        businessUnitId: "",
      },
    },
  });

  // Initialize the local form with existing data when editing or default values when adding
  useEffect(() => {
    if (open) {
      const now = Math.floor(Date.now() / 1000);
      const oneHour = 3600;

      if (isEditing) {
        // Load existing stop data
        const existingStop = getValues(`moves.${moveIdx}.stops.${stopIdx}`);
        if (existingStop) {
          localForm.reset({ stop: existingStop });
        }
      } else {
        // Initialize with default values for new stop
        localForm.reset({
          stop: {
            status: StopStatus.New,
            actualArrival: undefined,
            actualDeparture: undefined,
            pieces: 0,
            weight: 0,
            type: StopType.Pickup,
            sequence: stopIdx,
            locationId: "",
            addressLine: "",
            plannedArrival: now,
            plannedDeparture: now + oneHour,
            organizationId: getValues().moves?.[moveIdx]?.organizationId || "",
            businessUnitId: getValues().moves?.[moveIdx]?.businessUnitId || "",
          },
        });
      }
    }
  }, [open, isEditing, getValues, moveIdx, stopIdx, localForm]);

  // Sync server validation errors to local form
  useEffect(() => {
    if (open) {
      const stopErrors = errors.moves?.[moveIdx]?.stops?.[stopIdx];
      if (stopErrors) {
        // Clear existing errors first
        localForm.clearErrors();

        // Map server errors to local form
        Object.entries(stopErrors).forEach(([fieldName, error]) => {
          if (error && typeof error === "object" && "message" in error) {
            const localFieldPath = `stop.${fieldName}` as any;
            localForm.setError(localFieldPath, {
              type: "server",
              message: error.message as string,
            });
          }
        });
      }
    }
  }, [open, errors, moveIdx, stopIdx, localForm]);

  const handleClose = useCallback(() => {
    onOpenChange(false);
    // Clear main form errors for this stop
    clearErrors(`moves.${moveIdx}.stops.${stopIdx}`);
    localForm.reset(); // Reset local form on close
  }, [onOpenChange, clearErrors, localForm, moveIdx, stopIdx]);

  const handleSave = useCallback(async () => {
    setIsSubmitting(true);

    try {
      // Get the stop data from the local form
      const stopData = localForm.getValues("stop");

      if (!stopData) {
        toast.error("Stop data not found");
        return;
      }

      // Validate the stop data using zod schema
      const validationResult = stopSchema.safeParse(stopData);

      if (!validationResult.success) {
        // Set form errors on the local form
        const errors = validationResult.error.issues;

        console.log("stop dialog validation errors", errors);

        errors.forEach((error) => {
          const fieldPath = `stop.${error.path.join(".")}` as any;
          localForm.setError(fieldPath, {
            type: "manual",
            message: error.message,
          });
        });

        toast.error("Please fix the validation errors", {
          description: "Check the form for errors and try again",
        });
        return;
      }

      // Trigger validation on the local form
      const isValid = await localForm.trigger("stop");

      if (!isValid) {
        toast.error("Please fix the validation errors", {
          description: "Check the form for errors and try again",
        });
        return;
      }

      // If validation passed, update the main shipment form
      if (isEditing) {
        // Update existing stop
        update(stopIdx, stopData);
      } else {
        // Add new stop
        append(stopData);
      }

      // Close the dialog first
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
  }, [localForm, stopIdx, isEditing, onOpenChange, update, append]);

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
          <FormProvider {...localForm}>
            <StopDialogForm moveIdx={0} stopIdx={0} stopFieldName="stop" />
          </FormProvider>
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
