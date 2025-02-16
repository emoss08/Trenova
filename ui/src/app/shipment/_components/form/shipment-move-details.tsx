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
import { ShipmentSchema } from "@/lib/schemas/shipment-schema";
import { MoveStatus, type ShipmentMove } from "@/types/move";
import { faEllipsisVertical, faPlus } from "@fortawesome/pro-regular-svg-icons";
import { memo, useState } from "react";
import { useFieldArray, useFormContext } from "react-hook-form";
import { AssignmentDialog } from "../assignment/assignment-dialog";
import { StopTimeline } from "../sidebar/stop-details/stop-timeline-content";
import { AssignmentDetails } from "./move-assignment-details";

export function ShipmentMovesDetails() {
  const { control } = useFormContext<ShipmentSchema>();

  const {
    fields: moves,
    append,
    remove,
  } = useFieldArray({
    control,
    name: "moves",
  });

  return (
    <div className="flex flex-col gap-2 border-t border-bg-sidebar-border py-4">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-1">
          <h3 className="text-sm font-medium">Moves</h3>
          <span className="text-2xs text-muted-foreground">
            ({moves?.length ?? 0})
          </span>
        </div>
        <Button variant="outline" size="xs">
          <Icon icon={faPlus} className="size-4" />
          Add Move
        </Button>
      </div>
      <div className="flex flex-col gap-4">
        {moves.map((move) => (
          <MoveInformation key={move.id} move={move as ShipmentMove} />
        ))}
      </div>
    </div>
  );
}

const MoveInformation = memo(function MoveInformation({
  move,
}: {
  move?: ShipmentMove;
}) {
  if (!move) {
    return <p>No move</p>;
  }

  return (
    <div
      className="bg-card rounded-lg border border-bg-sidebar-border space-y-2"
      key={move.id}
    >
      <StatusBadge move={move} />
      <ScrollArea className="flex max-h-[250px] flex-col overflow-y-auto px-4 py-2 rounded-b-lg">
        <div className="relative py-4">
          <div className="space-y-6">
            {move.stops.map((stop, index) => {
              const isLastStop = index === move.stops.length - 1;

              return (
                <StopTimeline
                  key={stop.id}
                  stop={stop}
                  isLast={isLastStop}
                  moveStatus={move.status}
                />
              );
            })}
          </div>
        </div>
        <div className="pointer-events-none absolute bottom-0 left-0 right-0 h-8 bg-gradient-to-t from-sidebar to-transparent z-50" />
      </ScrollArea>
      <AssignmentDetails move={move} />
    </div>
  );
});

const StatusBadge = memo(function StatusBadge({
  move,
}: {
  move?: ShipmentMove;
}) {
  if (!move) {
    return <p>No move</p>;
  }

  return (
    <div className="flex justify-between items-center p-3 border-b border-sidebar-border">
      <MoveStatusBadge status={move.status} />
      <MoveActions move={move} />
    </div>
  );
});

const AssignmentStatus = [MoveStatus.New, MoveStatus.Assigned];

function MoveActions({ move }: { move: ShipmentMove }) {
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
        <DropdownMenuContent align="start">
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
            description="Divide this move into multiple parts."
          />
          <DropdownMenuItem
            title="Edit Move"
            description="Modify move details."
          />
          <DropdownMenuItem
            title="View Audit Log"
            description="View the audit log for the move."
          />
        </DropdownMenuContent>
      </DropdownMenu>
      <AssignmentDialog
        open={assignmentDialogOpen}
        onOpenChange={setAssignmentDialogOpen}
        shipmentMoveId={move.id}
        assignmentId={assignment?.id}
      />
    </>
  );
}
