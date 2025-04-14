import { LazyComponent } from "@/components/error-boundary";
import { Button } from "@/components/ui/button";
import { Icon } from "@/components/ui/icons";
import { MOVE_DELETE_DIALOG_KEY } from "@/constants/env";
import { MoveSchema } from "@/lib/schemas/move-schema";
import { ShipmentSchema } from "@/lib/schemas/shipment-schema";
import { faPlus } from "@fortawesome/pro-regular-svg-icons";
import { lazy, memo, useCallback, useMemo, useState } from "react";
import { useFieldArray, useFormContext } from "react-hook-form";
import { toast } from "sonner";
import { MoveDeleteDialog } from "./move-delete-dialog";
import { MoveDialog } from "./move-dialog";

// Lazy loaded components
const MoveInformation = lazy(() => import("./move-information"));

// Utility function to resequence moves after deletion
const resequenceMoves = (
  moves?: MoveSchema[],
  deletedIndex?: number,
): MoveSchema[] => {
  if (!moves || !deletedIndex) {
    return [];
  }

  // Create a copy of the moves array to avoid mutating the original
  const updatedMoves = [...moves];

  // Get the sequence number of the deleted move
  const deletedSequence = moves[deletedIndex].sequence;

  // Adjust sequence numbers for all moves after the deleted one
  for (let i = 0; i < updatedMoves.length; i++) {
    if (i !== deletedIndex && updatedMoves[i].sequence > deletedSequence) {
      updatedMoves[i] = {
        ...updatedMoves[i],
        sequence: updatedMoves[i].sequence - 1,
      };
    }
  }

  return updatedMoves;
};

const ShipmentMovesDetailsComponent = () => {
  const { control, getValues } = useFormContext<ShipmentSchema>();
  const [moveDialogOpen, setMoveDialogOpen] = useState(false);
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const [editingIndex, setEditingIndex] = useState<number | null>(null);
  const [deletingIndex, setDeletingIndex] = useState<number | null>(null);

  const {
    fields: moves,
    update,
    remove,
  } = useFieldArray({
    control,
    name: "moves",
    keyName: "formId",
  });

  const handleAddMove = useCallback(() => {
    setMoveDialogOpen(true);
  }, []);

  const handleEditMove = useCallback((index: number) => {
    setEditingIndex(index);
    setMoveDialogOpen(true);
  }, []);

  // Modified handleDeleteMove function for the ShipmentMovesDetails component
  const handleDeleteMove = useCallback((index: number) => {
    // If there is only one move, we cannot delete it
    if (moves.length === 1) {
      toast.error("Unable to proceed", {
        description: "A shipment must have at least one move.",
      });
      return;
    }

    // Always check localStorage directly
    const showDialog = localStorage.getItem(MOVE_DELETE_DIALOG_KEY) !== "false";

    if (showDialog) {
      setDeletingIndex(index);
      setDeleteDialogOpen(true);
    } else {
      performDeleteMove(index);
    }
  }, [moves.length]);

  // Function to perform the actual deletion with resequencing
  const performDeleteMove = useCallback((index: number) => {
    // Get the current moves data
    const currentMoves = getValues("moves");

    // Resequence the moves
    const updatedMoves = resequenceMoves(currentMoves, index);

    // Update all moves with their new sequences before removal
    updatedMoves.forEach((move, idx) => {
      if (idx !== index) {
        // Skip the one being deleted
        update(idx, move);
      }
    });

    // Now remove the move at the specified index
    remove(index);
  }, [getValues, update, remove]);

  // Modified handleConfirmDelete function
  const handleConfirmDelete = useCallback((doNotShowAgain: boolean) => {
    if (deletingIndex !== null) {
      performDeleteMove(deletingIndex);

      if (doNotShowAgain) {
        localStorage.setItem(MOVE_DELETE_DIALOG_KEY, "false");
      }

      setDeleteDialogOpen(false);
      setDeletingIndex(null);
    }
  }, [deletingIndex, performDeleteMove]);

  const handleDialogClose = useCallback(() => {
    // If we're adding a new move and the dialog is closed without saving,
    // remove the placeholder move
    if (
      editingIndex === moves.length - 1 &&
      !moves[editingIndex]?.stops?.length
    ) {
      remove(editingIndex);
    }

    setMoveDialogOpen(false);
    setEditingIndex(null);
  }, [editingIndex, moves, remove]);

  const isEditing = useMemo(() => 
    editingIndex !== null &&
    ((editingIndex < moves.length - 1 || moves[editingIndex]?.stops?.length) ??
      false)
  , [editingIndex, moves]);

  return (
    <>
      <div className="flex flex-col gap-2 border-t border-bg-sidebar-border py-4">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-1">
            <h3 className="text-sm font-medium">Moves</h3>
            <span className="text-2xs text-muted-foreground">
              ({moves?.length ?? 0})
            </span>
          </div>
          <AddMoveButton onClick={handleAddMove} />
        </div>
        <LazyComponent>
          <MoveInformation
            moves={moves}
            update={update}
            remove={remove}
            onEdit={handleEditMove}
            onDelete={handleDeleteMove}
          />
        </LazyComponent>
      </div>
      {moveDialogOpen && (
        <MoveDialog
          open={moveDialogOpen}
          onOpenChange={handleDialogClose}
          isEditing={!!isEditing}
          moveIdx={editingIndex ?? moves.length}
          update={update}
          remove={remove}
        />
      )}
      {deleteDialogOpen && (
        <MoveDeleteDialog
          open={deleteDialogOpen}
          onOpenChange={(open) => {
            setDeleteDialogOpen(open);
            if (!open) {
              setDeletingIndex(null);
            }
          }}
          handleDelete={handleConfirmDelete}
        />
      )}
    </>
  );
};

export default memo(ShipmentMovesDetailsComponent);

const AddMoveButton = memo(function AddMoveButton({
  onClick,
}: {
  onClick: () => void;
}) {
  return (
    <Button type="button" variant="outline" size="xs" onClick={onClick}>
      <Icon icon={faPlus} className="size-4" />
      Add Move
    </Button>
  );
});
