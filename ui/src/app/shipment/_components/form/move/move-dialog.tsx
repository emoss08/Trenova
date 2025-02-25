import { InputField } from "@/components/fields/input-field";
import { SelectField } from "@/components/fields/select-field";
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
import { FormControl, FormGroup } from "@/components/ui/form";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { moveStatusChoices } from "@/lib/choices";
import { ShipmentSchema } from "@/lib/schemas/shipment-schema";
import { type TableSheetProps } from "@/types/data-table";
import { MoveStatus, type ShipmentMove } from "@/types/move";
import { StopStatus, StopType } from "@/types/stop";
import { useCallback, useEffect } from "react";
import {
  useFormContext,
  type UseFieldArrayRemove,
  type UseFieldArrayUpdate,
} from "react-hook-form";

type MoveDialogProps = TableSheetProps & {
  moveIdx: number;
  isEditing: boolean;
  update: UseFieldArrayUpdate<ShipmentSchema, "moves">;
  remove: UseFieldArrayRemove;
  initialData?: ShipmentMove;
};

export function MoveDialog({
  open,
  onOpenChange,
  moveIdx,
  isEditing,
  update,
  remove,
  initialData,
}: MoveDialogProps) {
  const { getValues, setValue } = useFormContext<ShipmentSchema>();

  // Set default values when the dialog opens and it's a new move
  useEffect(() => {
    if (open && !isEditing) {
      // Set default values for a new move
      setValue(`moves.${moveIdx}.status`, MoveStatus.New);
      setValue(`moves.${moveIdx}.distance`, 0);
      setValue(`moves.${moveIdx}.loaded`, true);
      // Append two new stops to the move
      setValue(`moves.${moveIdx}.stops`, [
        {
          sequence: 0,
          status: StopStatus.New,
          type: StopType.Pickup,
          locationId: "",
          // @ts-expect-error - This is a temporary fix
          plannedDeparture: undefined,
          // @ts-expect-error - This is a temporary fix
          plannedArrival: undefined,
          addressLine: "",
        },
        {
          sequence: 1,
          status: StopStatus.New,
          type: StopType.Delivery,
          locationId: "",
          // @ts-expect-error - This is a temporary fix
          plannedDeparture: undefined,
          // @ts-expect-error - This is a temporary fix
          plannedArrival: undefined,
          addressLine: "",
        },
      ]);
      setValue(`moves.${moveIdx}.sequence`, moveIdx);
    }
  }, [open, isEditing, moveIdx, setValue]);

  const handleClose = useCallback(() => {
    onOpenChange(false);
  }, [onOpenChange]);

  const handleSave = useCallback(() => {
    const formValues = getValues();
    const move = formValues.moves?.[moveIdx];

    if (move) {
      // Ensure all required fields have values
      const updatedMove = {
        ...move,
        status: move.status || MoveStatus.New,
        distance: move.distance ?? 0,
        loaded: move.loaded ?? true,
        stops: move.stops || [],
        sequence: move.sequence ?? moveIdx,
      };

      update(moveIdx, updatedMove);
      onOpenChange(false);
    } else {
      console.error("No move data found at index", moveIdx);
    }
  }, [getValues, moveIdx, update, onOpenChange]);

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

  return (
    <>
      <Dialog open={open} onOpenChange={onOpenChange}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{isEditing ? "Edit Move" : "Add Move"}</DialogTitle>
            <DialogDescription>
              {isEditing
                ? "Edit the move details for this shipment."
                : "Add a new move to the shipment."}
            </DialogDescription>
          </DialogHeader>
          <DialogBody>
            <MoveDialogForm moveIdx={moveIdx} />
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

function MoveDialogForm({ moveIdx }: { moveIdx: number }) {
  const { control } = useFormContext<ShipmentSchema>();

  return (
    <div className="space-y-2">
      <div>
        <div className="flex items-center gap-2 mb-1">
          <h3 className="text-sm font-semibold text-foreground">
            Basic Information
          </h3>
        </div>
        <p className="text-2xs text-muted-foreground mb-3">
          Define the fundamental details and current status of this move.
        </p>
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
    </div>
  );
}
