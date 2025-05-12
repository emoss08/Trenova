import { StopDialog } from "@/app/shipment/_components/sidebar/stop-details/stop-dialog";
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
import { http } from "@/lib/http-client";
import type { MoveSchema } from "@/lib/schemas/move-schema";
import { MoveStatus } from "@/types/move";
import { faEllipsisVertical } from "@fortawesome/pro-regular-svg-icons";
import { useQueryClient } from "@tanstack/react-query";
import { useCallback, useState } from "react";
import { useFieldArray } from "react-hook-form";
import { AssignmentDialog } from "../../assignment/assignment-dialog";

export function MoveActions({
  move,
  moveIdx,
  onEdit,
  onDelete,
}: {
  move: MoveSchema;
  moveIdx: number;
  onEdit: () => void;
  onDelete: () => void;
}) {
  // * TODO(Wolfred): we need to add a check before this is able to open, if the move is undefined.
  // * More than likely, we just need to disable the move actions if there is no move.

  const [assignmentDialogOpen, setAssignmentDialogOpen] =
    useState<boolean>(false);
  const queryClient = useQueryClient();

  // Use field array for the stops
  const { update } = useFieldArray({
    name: `moves.${moveIdx}`,
  });

  const { assignment, status } = move;

  // Move is not new, so we cannot assign equipment and workers
  const reassignEnabled = status === MoveStatus.Assigned;
  const assignEnabled = validAssignmentStatuses.includes(status);

  const handleOpenAssignmentDialog = useCallback(() => {
    setAssignmentDialogOpen(true);
  }, []);

  const handleCloseAssignmentDialog = useCallback(
    async (open: boolean) => {
      setAssignmentDialogOpen(open);

      if (!open && move.id) {
        // Invalidate queries to ensure other components have the latest data
        queryClient.invalidateQueries({
          queryKey: ["shipment", "stop", "assignment", "move"],
        });

        // Wait briefly for the server to process the assignment
        setTimeout(async () => {
          try {
            // Fetch the latest move data directly
            const response = await http.get(
              `/shipment-moves/${move.id}?expandMoveDetails=true`,
            );
            if (response.data) {
              // Update the move data in the form
              const updatedMove = { ...move, ...response.data };
              update(moveIdx, updatedMove);
            }
          } catch (error) {
            console.error("Failed to fetch updated move data:", error);
          }
        }, 300); // Small delay to ensure server has processed the assignment
      }
    },
    [move, moveIdx, queryClient, update],
  );

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
            onClick={handleOpenAssignmentDialog}
            disabled={!assignEnabled}
          />

          <StopDialogAction moveIdx={moveIdx} stopIdx={0} />
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
          onOpenChange={handleCloseAssignmentDialog}
          shipmentMoveId={move.id}
          assignmentId={assignment?.id}
        />
      )}
    </>
  );
}
// * Statuses where the worker can be assigned.
const validAssignmentStatuses = [MoveStatus.New, MoveStatus.Assigned];

function StopDialogAction({
  moveIdx,
  stopIdx,
}: {
  moveIdx: number;
  stopIdx: number;
}) {
  const [isOpen, setIsOpen] = useState<boolean>(false);

  const handleOpen = useCallback(() => {
    setIsOpen(true);
  }, []);

  const handleClose = useCallback(() => {
    setIsOpen(false);
  }, []);

  console.info("stop-dialog-action debug information", {
    moveIdx,
    isOpen,
  });

  return (
    <>
      <DropdownMenuItem
        title="Add Stop"
        description="Add a new stop to the move"
        onClick={handleOpen}
      />
      {isOpen && (
        <StopDialog
          open={isOpen}
          onOpenChange={handleClose}
          moveIdx={moveIdx}
          stopIdx={stopIdx}
        />
      )}
    </>
  );
}
