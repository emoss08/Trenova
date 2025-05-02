import { MoveStatusBadge } from "@/components/status-badge";
import { type ShipmentSchema } from "@/lib/schemas/shipment-schema";
import { type ShipmentMove } from "@/types/move";
import { useQueryClient } from "@tanstack/react-query";
import { nanoid } from "nanoid";
import React, { memo, useCallback, useEffect, useMemo } from "react";
import {
  UseFieldArrayRemove,
  UseFieldArrayUpdate,
  type FieldArrayWithId,
} from "react-hook-form";
import { AssignmentDetails } from "../move-assignment-details";
import { MoveActions } from "./move-actions";
import { MoveInformationHeader } from "./move-header";
import { MoveList } from "./move-list";

type MoveInformationProps = {
  moves: FieldArrayWithId<ShipmentSchema, "moves", "formId">[];
  update: UseFieldArrayUpdate<ShipmentSchema, "moves">;
  remove: UseFieldArrayRemove;
  onEdit: (index: number) => void;
  onDelete: (index: number) => void;
};

// Use memo to prevent unnecessary re-renders of the entire component
const MoveInformation = memo(function MoveInformation({
  moves,
  update,
  remove,
  onEdit,
  onDelete,
}: MoveInformationProps) {
  const queryClient = useQueryClient();

  // * TODO(wolfred): Figure out if we can get rid of this.
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

  // Create memoized event handlers
  const handleEdit = useCallback(
    (moveIdx: number) => {
      onEdit(moveIdx);
    },
    [onEdit],
  );

  const handleDelete = useCallback(
    (moveIdx: number) => {
      onDelete(moveIdx);
    },
    [onDelete],
  );

  // Use memo for the moves list to prevent unnecessary re-renders
  const movesList = useMemo(() => {
    return moves.map((move, moveIdx) => (
      <MoveRow
        key={move.id || nanoid()}
        move={move as ShipmentMove}
        moveIdx={moveIdx}
        update={update}
        remove={remove}
        onEdit={() => handleEdit(moveIdx)}
        onDelete={() => handleDelete(moveIdx)}
      />
    ));
  }, [moves, update, remove, handleEdit, handleDelete]);

  return <div className="flex flex-col gap-4">{movesList}</div>;
});

// Export the memoized component as default
export default MoveInformation;

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
    <MoveRowInner>
      <MoveInformationHeader>
        <MoveStatusBadge status={move.status} />
        <MoveActions
          move={move}
          moveIdx={moveIdx}
          onEdit={onEdit}
          onDelete={onDelete}
        />
      </MoveInformationHeader>
      <MoveList move={move} moveIdx={moveIdx} update={update} remove={remove} />
      <AssignmentDetails move={move} />
    </MoveRowInner>
  );
});

function MoveRowInner({ children }: { children: React.ReactNode }) {
  return (
    <div className="bg-background rounded-lg border border-bg-sidebar-border space-y-2">
      {children}
    </div>
  );
}
