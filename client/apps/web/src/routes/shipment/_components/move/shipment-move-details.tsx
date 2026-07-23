import { Button } from "@/components/ui/button";
import { FormSection } from "@/components/ui/form";
import { TruckIcon } from "@/components/ui/truck";
import { queries } from "@/lib/queries";
import type { Shipment } from "@/types/shipment";
import { useQuery } from "@tanstack/react-query";
import { PlusIcon } from "lucide-react";
import { useState } from "react";
import { useFieldArray, useFormContext, useWatch } from "react-hook-form";
import { MoveEditDialog } from "../shipment-move-edit-dialog";
import { MoveCard } from "./move-card";

type MoveDialogState = { open: false } | { open: true; moveIndex: number; isNew: boolean };

export default function ShipmentMoveDetails() {
  const { control } = useFormContext<Shipment>();
  const shipmentID = useWatch({ control, name: "id" });
  const {
    fields: moveFields,
    append: appendMove,
    remove: removeMove,
  } = useFieldArray({
    control,
    name: "moves",
    keyName: "fieldId",
  });

  const [moveDialog, setMoveDialog] = useState<MoveDialogState>({
    open: false,
  });
  const { data: shipmentControl } = useQuery({
    ...queries.shipment.uiPolicy(),
  });
  const allowPersistedMoveRemoval = !shipmentID || shipmentControl?.allowMoveRemovals === true;

  function handleDialogClose() {
    setMoveDialog({ open: false });
  }

  function handleDialogCancel() {
    if (moveDialog.open && moveDialog.isNew) {
      removeMove(moveDialog.moveIndex);
    }
    setMoveDialog({ open: false });
  }
  function handleAddMove() {
    const newIndex = moveFields.length;
    appendMove({
      status: "New",
      loaded: true,
      sequence: newIndex,
      distance: 0,
      stops: [
        {
          status: "New",
          type: "Pickup",
          scheduleType: "Open",
          locationId: "",
          sequence: 0,
          scheduledWindowStart: 0,
          scheduledWindowEnd: null,
        },
        {
          status: "New",
          type: "Delivery",
          scheduleType: "Open",
          locationId: "",
          sequence: 1,
          scheduledWindowStart: 0,
          scheduledWindowEnd: null,
        },
      ],
    });
    setMoveDialog({ open: true, moveIndex: newIndex, isNew: true });
  }

  return (
    <>
      <FormSection
        title="Move Details"
        description="Execution legs and stop sequences for this shipment"
        className="border-t border-border pt-4"
        action={
          <Button type="button" variant="outline" size="xxs" onClick={handleAddMove}>
            <PlusIcon className="size-3" />
            Add Move
          </Button>
        }
      >
        {moveFields.map((field, moveIndex) => (
          <MoveCard
            key={field.fieldId}
            moveIndex={moveIndex}
            allowPersistedMoveRemoval={allowPersistedMoveRemoval}
            onEdit={() => setMoveDialog({ open: true, moveIndex, isNew: false })}
            onRemove={() => {
              if (field.id && !allowPersistedMoveRemoval) {
                return;
              }
              removeMove(moveIndex);
            }}
          />
        ))}
        {moveFields.length === 0 && (
          <div className="flex flex-col items-center justify-center rounded-lg border border-dashed py-8 text-center">
            <TruckIcon className="mb-2 size-6 text-muted-foreground/40" />
            <p className="text-sm font-medium text-muted-foreground">No moves yet</p>
            <p className="mt-0.5 text-xs text-muted-foreground/70">
              Add a move to define the route for this shipment
            </p>
            <Button
              type="button"
              variant="outline"
              size="xs"
              className="mt-3"
              onClick={handleAddMove}
            >
              <PlusIcon className="size-3" />
              Add First Move
            </Button>
          </div>
        )}
      </FormSection>
      <MoveEditDialog
        state={moveDialog}
        onClose={handleDialogClose}
        onCancel={handleDialogCancel}
      />
    </>
  );
}
