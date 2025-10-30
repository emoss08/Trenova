import { LazyComponent } from "@/components/error-boundary";
import { MOVE_DELETE_DIALOG_KEY } from "@/constants/env";
import { ShipmentSchema } from "@/lib/schemas/shipment-schema";
import { lazy, memo, useCallback, useMemo, useState } from "react";
import { useFieldArray, useFormContext } from "react-hook-form";
import { toast } from "sonner";
import { MoveDeleteDialog } from "./move-delete-dialog";
import { MoveDialog } from "./move-dialog";
import { MoveListHeader } from "./move-list-header";

const MoveInformation = lazy(() => import("./move-information"));

const resequenceMoves = (
  moves?: ShipmentSchema["moves"],
  deletedIndex?: number,
): ShipmentSchema["moves"] => {
  if (!moves || !deletedIndex) {
    return [];
  }

  const updatedMoves = [...moves];

  const deletedSequence = moves[deletedIndex].sequence;

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

  const performDeleteMove = useCallback(
    (index: number) => {
      const currentMoves = getValues("moves");

      const updatedMoves = resequenceMoves(currentMoves, index);

      // * Update all moves with their new sequences before removal
      updatedMoves.forEach((move, idx) => {
        if (idx !== index) {
          // * Skip the one being deleted
          update(idx, move);
        }
      });

      remove(index);
    },
    [getValues, update, remove],
  );

  const handleDeleteMove = useCallback(
    (index: number) => {
      // * If there is only one move, we cannot delete it
      if (moves.length === 1) {
        toast.error("Unable to proceed", {
          description: "A shipment must have at least one move.",
        });
        return;
      }

      const showDialog =
        localStorage.getItem(MOVE_DELETE_DIALOG_KEY) !== "false";

      if (showDialog) {
        setDeletingIndex(index);
        setDeleteDialogOpen(true);
      } else {
        performDeleteMove(index);
      }
    },
    [moves.length, performDeleteMove],
  );

  const handleConfirmDelete = useCallback(
    (doNotShowAgain: boolean) => {
      if (deletingIndex !== null) {
        performDeleteMove(deletingIndex);

        if (doNotShowAgain) {
          localStorage.setItem(MOVE_DELETE_DIALOG_KEY, "false");
        }

        setDeleteDialogOpen(false);
        setDeletingIndex(null);
      }
    },
    [deletingIndex, performDeleteMove],
  );

  const handleDialogClose = useCallback(() => {
    // * If we're adding a new move and the dialog is closed without saving,
    // * remove the placeholder move
    if (
      editingIndex === moves.length - 1 &&
      !moves[editingIndex]?.stops?.length
    ) {
      remove(editingIndex);
    }

    setMoveDialogOpen(false);
    setEditingIndex(null);
  }, [editingIndex, moves, remove]);

  const isEditing = useMemo(
    () =>
      editingIndex !== null &&
      ((editingIndex < moves.length - 1 ||
        moves[editingIndex]?.stops?.length) ??
        false),
    [editingIndex, moves],
  );

  return (
    <ShipmentMovesDetailsInner className="flex flex-col gap-2 border-t border-bg-sidebar-border py-4">
      <MoveListHeader moves={moves} handleAddMove={handleAddMove} />
      <LazyComponent>
        <MoveInformation
          moves={moves}
          update={update}
          remove={remove}
          onEdit={handleEditMove}
          onDelete={handleDeleteMove}
        />
      </LazyComponent>
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
    </ShipmentMovesDetailsInner>
  );
};

export default memo(ShipmentMovesDetailsComponent);

function ShipmentMovesDetailsInner({
  children,
  className,
}: {
  children: React.ReactNode;
  className?: string;
}) {
  return <div className={className}>{children}</div>;
}
