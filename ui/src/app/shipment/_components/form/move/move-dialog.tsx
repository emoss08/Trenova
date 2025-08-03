/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import { InputField } from "@/components/fields/input-field";
import { SelectField } from "@/components/fields/select-field";
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
import { FormControl, FormGroup } from "@/components/ui/form";
import { Icon } from "@/components/ui/icons";
import { moveStatusChoices } from "@/lib/choices";
import { MoveStatus } from "@/lib/schemas/move-schema";
import { ShipmentSchema } from "@/lib/schemas/shipment-schema";
import { StopStatus, StopType } from "@/lib/schemas/stop-schema";
import { type TableSheetProps } from "@/types/data-table";
import { faPlus } from "@fortawesome/pro-regular-svg-icons";
import { AnimatePresence, motion } from "motion/react";
import { memo, useCallback, useEffect, useMemo, useRef, useState } from "react";
import {
  useFieldArray,
  useFormContext,
  type UseFieldArrayRemove,
  type UseFieldArrayUpdate,
} from "react-hook-form";
import { CompactStopForm } from "./move-stop-form-compact";
import { CompactStopsTable } from "./move-stops-table";

type MoveDialogProps = TableSheetProps & {
  moveIdx: number;
  isEditing: boolean;
  update: UseFieldArrayUpdate<ShipmentSchema, "moves">;
  remove: UseFieldArrayRemove;
};

const MoveDialogComponent = ({
  open,
  onOpenChange,
  moveIdx,
  isEditing,
  update,
  remove,
}: MoveDialogProps) => {
  const { control, getValues, setValue, reset } =
    useFormContext<ShipmentSchema>();
  const [editingStopIdx, setEditingStopIdx] = useState<number | null>(null);
  const hasSavedRef = useRef(false);

  // Use field array for stops
  const {
    fields,
    remove: removeStop,
    insert,
  } = useFieldArray({
    control,
    name: `moves.${moveIdx}.stops`,
  });

  // Reset saved state when dialog opens
  useEffect(() => {
    if (open) {
      hasSavedRef.current = false;
    }
  }, [open]);

  // Helper to update stop sequences - memoize to prevent recreation
  const updateStopSequences = useCallback(() => {
    const currentMoveValues = getValues(`moves.${moveIdx}`);

    if (currentMoveValues.stops) {
      // Manually update sequences of all stops
      const updatedStops = currentMoveValues.stops.map((stop, idx) => ({
        ...stop,
        sequence: idx,
      }));

      // Update the entire move with properly sequenced stops
      update(moveIdx, {
        ...currentMoveValues,
        stops: updatedStops,
      });
    }
  }, [getValues, moveIdx, update]);

  // Initialize a new move with default values - memoize this function
  const initializeNewMove = useCallback(() => {
    if (!open || isEditing) return;

    // Set default values for a new move
    setValue(`moves.${moveIdx}.status`, MoveStatus.enum.New);
    setValue(`moves.${moveIdx}.distance`, 0);
    setValue(`moves.${moveIdx}.loaded`, true);
    setValue(`moves.${moveIdx}.sequence`, moveIdx);

    // Set the current time for defaults
    const now = Math.floor(Date.now() / 1000);
    const oneHour = 3600;

    // Append two new stops to the move
    // TODO(Wolfred): Add placeholder data
    setValue(`moves.${moveIdx}.stops`, [
      {
        sequence: 0,
        status: StopStatus.enum.New,
        type: StopType.enum.Pickup,
        locationId: "",
        plannedArrival: now,
        plannedDeparture: now + oneHour / 2,
        addressLine: "",
        location: null,
      },
      {
        sequence: 1,
        status: StopStatus.enum.New,
        type: StopType.enum.Delivery,
        locationId: "",
        plannedArrival: now + oneHour,
        plannedDeparture: now + oneHour + oneHour / 2,
        addressLine: "",
        location: null,
      },
    ]);
  }, [open, isEditing, moveIdx, setValue]);

  // Call the initialization function
  useEffect(() => {
    initializeNewMove();
  }, [initializeNewMove]);

  // Handle dialog close
  const handleClose = useCallback(() => {
    // If we're adding a new move and haven't explicitly saved it, remove it
    if (!isEditing && !hasSavedRef.current) {
      remove(moveIdx);
    } else if (isEditing && !hasSavedRef.current) {
      // If we're editing an existing move but haven't saved changes, reset to original values
      const originalValues = getValues();
      const moves = originalValues?.moves || [];

      reset(
        {
          moves: [
            ...moves.slice(0, moveIdx),
            moves[moveIdx],
            ...moves.slice(moveIdx + 1),
          ],
        },
        { keepValues: true },
      );
    }

    // Close the dialog
    onOpenChange(false);
  }, [onOpenChange, getValues, moveIdx, remove, isEditing, reset]);

  // Add a handler for dialog's escape key or outside click to ensure we remove unsaved moves
  const handleOpenChange = useCallback(
    (newOpenState: boolean) => {
      if (!newOpenState && !hasSavedRef.current) {
        handleClose();
      } else {
        onOpenChange(newOpenState);
      }
    },
    [handleClose, onOpenChange, hasSavedRef],
  );

  // Handle save move
  const handleSave = useCallback(() => {
    const move = getValues().moves?.[moveIdx];

    if (move) {
      // Ensure all required fields have values and preserve location data in stops
      const updatedMove = {
        ...move,
        status: move.status || MoveStatus.enum.New,
        distance: move.distance ?? 0,
        loaded: move.loaded ?? true,
        // Make sure we preserve all stop data
        stops: move.stops || [],
        sequence: move.sequence ?? moveIdx,
      };

      update(moveIdx, updatedMove);
      hasSavedRef.current = true;
      onOpenChange(false);
    }
  }, [moveIdx, update, onOpenChange, getValues]);

  // Handle keyboard shortcut (Ctrl+Enter) to save
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.ctrlKey && e.key === "Enter" && open) {
        handleSave();
      }
    };

    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, [open, handleSave]);

  // Memoize this function to prevent recreation on every render
  const handleAddStop = useCallback(() => {
    // Insert after the first stop but before the last stop
    const insertPosition = fields.length > 1 ? fields.length - 1 : 1;

    // Get the previous stop's departure time
    const prevStop = fields[insertPosition - 1];

    // Calculate times that make sense between previous and next stops
    const prevDepartureTime =
      prevStop?.plannedDeparture || Math.floor(Date.now() / 1000);

    const oneHour = 3600;
    const plannedArrival = Math.floor(prevDepartureTime + oneHour / 2);
    const plannedDeparture = Math.floor(prevDepartureTime + oneHour);

    // Alternate pickup/delivery for intermediate stops
    const isEvenPosition = insertPosition % 2 === 0;
    const stopType = isEvenPosition
      ? StopType.enum.Pickup
      : StopType.enum.Delivery;

    // Insert the new stop at position just before the last stop
    // TODO(Wolfred): Add placeholder data
    insert(insertPosition, {
      sequence: insertPosition,
      status: StopStatus.enum.New,
      type: stopType,
      locationId: "",
      plannedArrival: plannedArrival,
      plannedDeparture: plannedDeparture,
      addressLine: "",
      location: null,
    });

    // Update sequences of all stops
    updateStopSequences();

    // Start editing the new stop
    setEditingStopIdx(insertPosition);
  }, [fields, insert, updateStopSequences, setEditingStopIdx]);

  const handleEditStop = useCallback((stopIdx: number) => {
    setEditingStopIdx(stopIdx);
  }, []);

  const handleStopEditCancel = useCallback(() => {
    setEditingStopIdx(null);
  }, []);

  const handleStopEditSave = useCallback(() => {
    // Get the current form values for the stop being edited
    const currentValues = getValues();
    const editedStop = currentValues.moves?.[moveIdx]?.stops?.[editingStopIdx!];

    if (editedStop) {
      // Get the current move data
      const currentMoveValues = getValues(`moves.${moveIdx}`);

      // Update the specific stop in the stops array
      if (currentMoveValues.stops) {
        const updatedStops = [...currentMoveValues.stops];
        updatedStops[editingStopIdx!] = editedStop;

        // Update the entire move with the edited stop
        update(moveIdx, {
          ...currentMoveValues,
          stops: updatedStops,
        });
      }
    }

    // Close the edit form
    setEditingStopIdx(null);
  }, [editingStopIdx, getValues, moveIdx, update]);

  const handleDeleteStop = useCallback(
    (stopIdx: number) => {
      // Prevent deletion of first pickup or last delivery
      if (stopIdx === 0 || stopIdx === fields.length - 1) {
        return;
      }

      // Remove the stop
      removeStop(stopIdx);

      // Update sequences of all remaining stops
      updateStopSequences();

      // Handle the editing state if relevant
      if (editingStopIdx !== null) {
        if (editingStopIdx === stopIdx) {
          setEditingStopIdx(null);
        } else if (editingStopIdx > stopIdx) {
          setEditingStopIdx(editingStopIdx - 1);
        }
      }
    },
    [fields.length, removeStop, updateStopSequences, editingStopIdx],
  );

  // Memoize the dialog title and description
  const dialogInfo = useMemo(
    () => ({
      title: isEditing ? "Edit Move" : "Add Move",
      description: isEditing
        ? "Edit the move details for this shipment."
        : "Add a new move to the shipment.",
    }),
    [isEditing],
  );

  // Memoize the save button text
  const saveButtonText = useMemo(
    () => (isEditing ? "Update" : "Add"),
    [isEditing],
  );

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent className="max-w-3xl">
        <DialogHeader>
          <DialogTitle>{dialogInfo.title}</DialogTitle>
          <DialogDescription>{dialogInfo.description}</DialogDescription>
        </DialogHeader>
        <DialogBody>
          {/* Move Basic Information */}
          <div className="space-y-6">
            <FormGroup cols={2} className="gap-4">
              <FormControl>
                <SelectField
                  control={control}
                  name={`moves.${moveIdx}.status`}
                  label="Status"
                  placeholder="Select status"
                  isReadOnly
                  description="Indicates the current operational status of this move."
                  options={moveStatusChoices}
                />
              </FormControl>
              <FormControl>
                <InputField
                  name={`moves.${moveIdx}.distance`}
                  control={control}
                  label="Distance"
                  placeholder="Enter distance"
                  type="text"
                  description="Specifies the total distance of this move."
                  sideText="mi"
                  readOnly
                />
              </FormControl>
            </FormGroup>
          </div>

          {/* Stops Section */}
          <div className="space-y-2">
            <div className="flex justify-between items-center">
              <div className="flex items-center gap-1">
                <h3 className="text-sm font-medium">Stops</h3>
                <span className="text-2xs text-muted-foreground">
                  ({fields.length})
                </span>
              </div>
              <Button
                size="xs"
                variant="outline"
                onClick={handleAddStop}
                className="flex items-center gap-1"
                disabled={editingStopIdx !== null}
              >
                <Icon icon={faPlus} className="size-3.5" />
                Add Stop
              </Button>
            </div>

            <AnimatePresence mode="wait">
              {editingStopIdx !== null ? (
                <motion.div
                  key="edit-form"
                  initial={{ opacity: 0, y: 10 }}
                  animate={{ opacity: 1, y: 0 }}
                  exit={{ opacity: 0, y: -10 }}
                  transition={{ duration: 0.2 }}
                >
                  <CompactStopForm
                    moveIdx={moveIdx}
                    stopIdx={editingStopIdx}
                    onCancel={handleStopEditCancel}
                    onSave={handleStopEditSave}
                    isFirstOrLastStop={
                      editingStopIdx === 0 ||
                      editingStopIdx === fields.length - 1
                    }
                  />
                </motion.div>
              ) : (
                <motion.div
                  key="stops-table"
                  initial={{ opacity: 0 }}
                  animate={{ opacity: 1 }}
                  exit={{ opacity: 0 }}
                  transition={{ duration: 0.2 }}
                >
                  <CompactStopsTable
                    stops={fields}
                    onEdit={handleEditStop}
                    onDelete={handleDeleteStop}
                  />
                </motion.div>
              )}
            </AnimatePresence>
          </div>
        </DialogBody>
        <DialogFooter>
          <Button type="button" variant="outline" onClick={handleClose}>
            Cancel
          </Button>
          <FormSaveButton
            title="move"
            type="button"
            onClick={handleSave}
            text={saveButtonText}
          />
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
};

export const MoveDialog = memo(MoveDialogComponent);
