import { LazyComponent } from "@/components/error-boundary";
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
import { http } from "@/lib/http-client";
import { type ShipmentSchema } from "@/lib/schemas/shipment-schema";
import { MoveStatus, type ShipmentMove } from "@/types/move";
import { faEllipsisVertical } from "@fortawesome/pro-regular-svg-icons";
import { useQueryClient } from "@tanstack/react-query";
import { nanoid } from "nanoid";
import { lazy, memo, useEffect, useState } from "react";
import {
  UseFieldArrayRemove,
  UseFieldArrayUpdate,
  type FieldArrayWithId,
} from "react-hook-form";
import { AssignmentDialog } from "../../assignment/assignment-dialog";
import { AssignmentDetails } from "../move-assignment-details";

type MoveInformationProps = {
  moves: FieldArrayWithId<ShipmentSchema, "moves", "formId">[];
  update: UseFieldArrayUpdate<ShipmentSchema, "moves">;
  remove: UseFieldArrayRemove;
  onEdit: (index: number) => void;
  onDelete: (index: number) => void;
};

const StopTimeline = lazy(
  () =>
    import(
      "@/app/shipment/_components/sidebar/stop-details/stop-timeline-content"
    ),
);

export default function MoveInformation({
  moves,
  update,
  remove,
  onEdit,
  onDelete,
}: MoveInformationProps) {
  const queryClient = useQueryClient();

  // Listen for query invalidations that might affect the moves
  useEffect(() => {
    const unsubscribe = queryClient.getQueryCache().subscribe(() => {
      // When queries are invalidated or updated, we need to force a rerender
      // This ensures the component shows the latest assignment data
      // This is a lightweight way to cause a rerender without fetching data
      setTimeout(() => {
        queryClient.invalidateQueries({
          queryKey: ["force-rerender-move-information"],
          refetchType: "none",
        });
      }, 100);
    });

    return () => {
      unsubscribe();
    };
  }, [queryClient]);

  return (
    <div className="flex flex-col gap-4">
      {moves.map((move, moveIdx) => {
        return (
          <MoveRow
            key={move.id || nanoid()}
            move={move as ShipmentMove}
            moveIdx={moveIdx}
            update={update}
            remove={remove}
            onEdit={() => onEdit(moveIdx)}
            onDelete={() => onDelete(moveIdx)}
          />
        );
      })}
    </div>
  );
}

const MoveRow = memo(function MoveRow({
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
        <StatusBadge
          move={move}
          moveIdx={moveIdx}
          update={update}
          onEdit={onEdit}
          onDelete={onDelete}
        />
        <ScrollArea className="flex max-h-[250px] flex-col overflow-y-auto px-4 py-2 rounded-b-lg">
          <div className="relative py-4">
            <div className="space-y-6">
              {move.stops.map((stop, stopIdx) => {
                if (!stop) {
                  return null;
                }

                const isLastStop = stopIdx === move.stops.length - 1;
                const nextStop = !isLastStop ? move.stops[stopIdx + 1] : null;
                const prevStopStatus =
                  stopIdx > 0 ? move.stops[stopIdx - 1]?.status : undefined;

                return (
                  <LazyComponent key={stop.id || nanoid()}>
                    <StopTimeline
                      stop={stop}
                      nextStop={nextStop}
                      isLast={isLastStop}
                      moveStatus={move.status}
                      moveIdx={moveIdx}
                      stopIdx={stopIdx}
                      update={update}
                      remove={remove}
                      prevStopStatus={prevStopStatus}
                    />
                  </LazyComponent>
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
  moveIdx,
  update,
  onEdit,
  onDelete,
}: {
  move?: ShipmentMove;
  moveIdx: number;
  update: UseFieldArrayUpdate<ShipmentSchema, "moves">;
  onEdit: () => void;
  onDelete: () => void;
}) {
  if (!move) {
    return <p>No move</p>;
  }

  return (
    <div className="flex justify-between items-center p-3 border-b border-sidebar-border">
      <MoveStatusBadge status={move.status} />
      <MoveActions
        move={move}
        moveIdx={moveIdx}
        update={update}
        onEdit={onEdit}
        onDelete={onDelete}
      />
    </div>
  );
});

const AssignmentStatus = [MoveStatus.New, MoveStatus.Assigned];

function MoveActions({
  move,
  moveIdx,
  update,
  onEdit,
  onDelete,
}: {
  move: ShipmentMove;
  moveIdx: number;
  update: UseFieldArrayUpdate<ShipmentSchema, "moves">;
  onEdit: () => void;
  onDelete: () => void;
}) {
  const [assignmentDialogOpen, setAssignmentDialogOpen] =
    useState<boolean>(false);
  const queryClient = useQueryClient();

  if (!move) {
    return null;
  }

  const { assignment, status } = move;

  // Move is not new, so we cannot assign equipment and workers
  const reassignEnabled = status === MoveStatus.Assigned;
  const assignEnabled = AssignmentStatus.includes(status);

  const handleOpenAssignmentDialog = () => {
    setAssignmentDialogOpen(true);
  };

  const handleCloseAssignmentDialog = async (open: boolean) => {
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
  };

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
