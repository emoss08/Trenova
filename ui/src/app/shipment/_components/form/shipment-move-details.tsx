import { MoveStatusBadge } from "@/components/status-badge";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Icon } from "@/components/ui/icons";
import { ScrollArea } from "@/components/ui/scroll-area";
import { MOVE_DELETE_DIALOG_KEY } from "@/constants/env";
import { MoveSchema } from "@/lib/schemas/move-schema";
import { ShipmentSchema } from "@/lib/schemas/shipment-schema";
import { MoveStatus, type ShipmentMove } from "@/types/move";
import { faEllipsisVertical, faPlus } from "@fortawesome/pro-regular-svg-icons";
import { memo, useState } from "react";
import {
  useFieldArray,
  UseFieldArrayRemove,
  UseFieldArrayUpdate,
  useFormContext,
} from "react-hook-form";
import { toast } from "sonner";
import { AssignmentDialog } from "../assignment/assignment-dialog";
import { StopTimeline } from "../sidebar/stop-details/stop-timeline-content";
import { AssignmentDetails } from "./move-assignment-details";
import { MoveDeleteDialog } from "./move/move-delete-dialog";
import { MoveDialog } from "./move/move-dialog";

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

export function ShipmentMovesDetails() {
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
  });

  const handleAddMove = () => {
    setMoveDialogOpen(true);
  };

  const handleEditMove = (index: number) => {
    setEditingIndex(index);
    setMoveDialogOpen(true);
  };

  // Modified handleDeleteMove function for the ShipmentMovesDetails component
  const handleDeleteMove = (index: number) => {
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
  };

  // Function to perform the actual deletion with resequencing
  const performDeleteMove = (index: number) => {
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
  };

  // Modified handleConfirmDelete function
  const handleConfirmDelete = (doNotShowAgain: boolean) => {
    if (deletingIndex !== null) {
      performDeleteMove(deletingIndex);

      if (doNotShowAgain) {
        localStorage.setItem(MOVE_DELETE_DIALOG_KEY, "false");
      }

      setDeleteDialogOpen(false);
      setDeletingIndex(null);
    }
  };

  const handleDialogClose = () => {
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
  };

  const isEditing =
    editingIndex !== null &&
    ((editingIndex < moves.length - 1 || moves[editingIndex]?.stops?.length) ??
      0 > 0);

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
          <Button
            type="button"
            variant="outline"
            size="xs"
            onClick={handleAddMove}
          >
            <Icon icon={faPlus} className="size-4" />
            Add Move
          </Button>
        </div>
        <div className="flex flex-col gap-4">
          {moves.map((move, moveIdx) => {
            return (
              <MoveInformation
                key={move.id}
                move={move as ShipmentMove}
                moveIdx={moveIdx}
                update={update}
                remove={remove}
                onEdit={() => handleEditMove(moveIdx)}
                onDelete={() => handleDeleteMove(moveIdx)}
              />
            );
          })}
        </div>
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
}

const MoveInformation = memo(function MoveInformation({
  move,
  moveIdx,
  update,
  remove,
  onEdit,
  onDelete,
}: {
  move: ShipmentMove;
  moveIdx: number;
  update: UseFieldArrayUpdate<ShipmentSchema, "moves">;
  remove: UseFieldArrayRemove;
  onEdit: () => void;
  onDelete: () => void;
}) {
  if (!move) {
    return <p>No move</p>;
  }

  if (!move.stops) {
    return null;
  }

  return (
    <>
      <div
        className="bg-card rounded-lg border border-bg-sidebar-border space-y-2"
        key={move.id}
      >
        <StatusBadge move={move} onEdit={onEdit} onDelete={onDelete} />
        <ScrollArea className="flex max-h-[250px] flex-col overflow-y-auto px-4 py-2 rounded-b-lg">
          <div className="relative py-4">
            <div className="space-y-6">
              {move.stops.map((stop, stopIdx) => {
                if (!stop) {
                  return null;
                }

                const isLastStop = stopIdx === move.stops.length - 1;

                return (
                  <StopTimeline
                    key={stop.id}
                    stop={stop}
                    isLast={isLastStop}
                    moveStatus={move.status}
                    moveIdx={moveIdx}
                    stopIdx={stopIdx}
                    update={update}
                    remove={remove}
                  />
                );
              })}
            </div>
          </div>
          <div className="pointer-events-none absolute bottom-0 left-0 right-0 h-8 bg-gradient-to-t from-sidebar to-transparent z-50" />
        </ScrollArea>
        <AssignmentDetails move={move} />
      </div>
    </>
  );
});

const StatusBadge = memo(function StatusBadge({
  move,
  onEdit,
  onDelete,
}: {
  move?: ShipmentMove;
  onEdit: () => void;
  onDelete: () => void;
}) {
  if (!move) {
    return <p>No move</p>;
  }

  return (
    <div className="flex justify-between items-center p-3 border-b border-sidebar-border">
      <MoveStatusBadge status={move.status} />
      <MoveActions move={move} onEdit={onEdit} onDelete={onDelete} />
    </div>
  );
});

const AssignmentStatus = [MoveStatus.New, MoveStatus.Assigned];

function MoveActions({
  move,
  onEdit,
  onDelete,
}: {
  move: ShipmentMove;
  onEdit: () => void;
  onDelete: () => void;
}) {
  const [assignmentDialogOpen, setAssignmentDialogOpen] =
    useState<boolean>(false);

  if (!move) {
    return null;
  }

  const { assignment, status } = move;

  // Move is not new, so we cannot assign equipment and workers
  const reassignEnabled = status === MoveStatus.Assigned;
  const assignDisabled = !AssignmentStatus.includes(status);

  return (
    <>
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button variant="outline" size="sm" className="p-2">
            <Icon icon={faEllipsisVertical} className="size-4" />
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent side="left" align="start">
          <DropdownMenuLabel>Move Actions</DropdownMenuLabel>
          <DropdownMenuSeparator />
          <DropdownMenuItem
            title={reassignEnabled ? "Reassign" : "Assign"}
            description="Assign equipment and worker(s) to the move"
            onClick={() => setAssignmentDialogOpen(!assignmentDialogOpen)}
            disabled={assignDisabled}
          />
          <DropdownMenuItem
            title="Split Move"
            description="Divide this move into multiple parts"
          />
          <DropdownMenuItem
            title="Edit Move"
            description="Modify move details"
            onClick={onEdit}
          />
          <DropdownMenuItem
            title="View Audit Log"
            description="View the audit log for the move"
          />

          <DropdownMenuItem
            title="Delete Move"
            color="danger"
            description="Remove this move from the shipment"
            onClick={onDelete}
          />
        </DropdownMenuContent>
      </DropdownMenu>
      {assignmentDialogOpen && (
        <AssignmentDialog
          open={assignmentDialogOpen}
          onOpenChange={setAssignmentDialogOpen}
          shipmentMoveId={move.id}
          assignmentId={assignment?.id}
        />
      )}
    </>
  );
}
