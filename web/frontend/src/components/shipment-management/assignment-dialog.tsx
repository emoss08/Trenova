/**
 * COPYRIGHT(c) 2024 Trenova
 *
 * This file is part of Trenova.
 *
 * The Trenova software is licensed under the Business Source License 1.1. You are granted the right
 * to copy, modify, and redistribute the software, but only for non-production use or with a total
 * of less than three server instances. Starting from the Change Date (November 16, 2026), the
 * software will be made available under version 2 or later of the GNU General Public License.
 * If you use the software in violation of this license, your rights under the license will be
 * terminated automatically. The software is provided "as is," and the Licensor disclaims all
 * warranties and conditions. If you use this license's text or the "Business Source License" name
 * and trademark, you must comply with the Licensor's covenants, which include specifying the
 * Change License as the GPL Version 2.0 or a compatible license, specifying an Additional Use
 * Grant, and not modifying the license in any other way.
 */



import { cn } from "@/lib/utils";
import { getActiveAssignmentsForTractor } from "@/services/EquipmentRequestService";
import type {
  NewAssignment,
  Tractor,
  TractorAssignmentFormValues,
} from "@/types/equipment";
import { useQuery } from "@tanstack/react-query";
import { GripIcon, XIcon } from "lucide-react";
import { useEffect } from "react";
import {
  DragDropContext,
  Draggable,
  DraggableProvided,
  DraggableStateSnapshot,
  Droppable,
  DropResult,
} from "react-beautiful-dnd";
import ReactDOM from "react-dom";
import { useFieldArray, useForm } from "react-hook-form";
import { Input } from "../common/fields/input";
import { Button } from "../ui/button";
import {
  Credenza,
  CredenzaBody,
  CredenzaContent,
  CredenzaDescription,
  CredenzaHeader,
  CredenzaTitle,
} from "../ui/credenza";
import { DialogFooter } from "../ui/dialog";

// Create a portal container
const portalContainer = document.createElement("div");
portalContainer.setAttribute("id", "drag-drop-portal");
document.body.appendChild(portalContainer);

const PortalAwareItem = ({
  provided,
  snapshot,
  field,
  onRemove,
}: {
  provided: DraggableProvided;
  snapshot: DraggableStateSnapshot;
  field: TractorAssignmentFormValues["assignments"]["0"];
  onRemove: (id: any) => void;
}) => {
  const child = (
    <li
      ref={provided.innerRef}
      {...provided.draggableProps}
      className={cn(
        "border-border hover:bg-muted flex items-center space-x-2 rounded-md border p-2",
        snapshot.isDragging && "opacity-60 shadow-lg bg-muted",
      )}
    >
      <div
        {...provided.dragHandleProps}
        className="rounded p-1 hover:cursor-move"
      >
        <GripIcon className="text-foreground size-5" />
      </div>
      <Input value={field.shipmentProNumber} readOnly className="grow" />
      <Button
        className="size-8"
        size="icon"
        variant="ghost"
        onClick={() => onRemove(field.id)}
        type="button"
      >
        <XIcon className="size-4" />
      </Button>
    </li>
  );

  if (snapshot.isDragging) {
    return ReactDOM.createPortal(child, portalContainer);
  }

  return child;
};
export function AssignmentDialog({
  open,
  onOpenChange,
  handleAssignTractor,
  selectedTractor,
  newAssignment,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  handleAssignTractor: (
    assignments: TractorAssignmentFormValues["assignments"],
  ) => void;
  selectedTractor: Tractor;
  newAssignment: NewAssignment;
}) {
  const { data: activeAssignments, isLoading } = useQuery({
    queryKey: ["activeAssignments", selectedTractor.id],
    queryFn: async () => getActiveAssignmentsForTractor(selectedTractor.id),
    enabled: open,
  });

  const { control, handleSubmit } = useForm<TractorAssignmentFormValues>({
    defaultValues: {
      assignments: newAssignment
        ? [
            {
              id: "new",
              ...newAssignment,
              sequence: 1,
            },
          ]
        : [],
    },
  });

  const { fields, replace, move, remove } = useFieldArray({
    control,
    name: "assignments",
  });

  useEffect(() => {
    if (activeAssignments) {
      const currentAssignments = activeAssignments.map((assignment) => ({
        id: assignment.id,
        shipmentId: assignment.shipmentId,
        shipmentMoveId: assignment.shipmentMoveId,
        sequence: assignment.sequence,
        shipmentProNumber: assignment.shipment.proNumber,
        assignedById: assignment.assignedById,
      }));

      if (
        newAssignment &&
        !currentAssignments.some(
          (a) => a.shipmentId === newAssignment.shipmentId,
        )
      ) {
        currentAssignments.push({
          id: "new",
          ...newAssignment,
          sequence: currentAssignments.length + 1,
        });
      }

      replace(currentAssignments);
    }
  }, [activeAssignments, newAssignment, replace]);

  const onSubmit = (data: TractorAssignmentFormValues) => {
    handleAssignTractor(data.assignments);
    onOpenChange(false);
  };

  const handleDragEnd = (result: DropResult) => {
    if (!result.destination) return;
    move(result.source.index, result.destination.index);
  };

  if (isLoading && fields.length === 0) {
    return <div>Loading assignments...</div>;
  }

  return (
    <Credenza open={open} onOpenChange={onOpenChange}>
      <CredenzaContent className="max-w-md">
        <CredenzaHeader>
          <CredenzaTitle>Assignments: {selectedTractor.code}</CredenzaTitle>
        </CredenzaHeader>
        <CredenzaDescription>
          Drag and drop to reorder assignments.
        </CredenzaDescription>
        <CredenzaBody>
          <form onSubmit={handleSubmit(onSubmit)}>
            <DragDropContext onDragEnd={handleDragEnd}>
              <Droppable droppableId="assignments">
                {(provided) => (
                  <ul
                    {...provided.droppableProps}
                    ref={provided.innerRef}
                    className="max-h-[50vh] space-y-2 overflow-y-auto"
                  >
                    {fields.map((field, index) => (
                      <Draggable
                        key={field.id}
                        draggableId={field.id}
                        index={index}
                      >
                        {(provided, snapshot) => (
                          <PortalAwareItem
                            provided={provided}
                            snapshot={snapshot}
                            field={field}
                            onRemove={remove}
                          />
                        )}
                      </Draggable>
                    ))}
                    {provided.placeholder}
                  </ul>
                )}
              </Droppable>
            </DragDropContext>
            <DialogFooter className="mt-4">
              <Button
                type="button"
                variant="outline"
                onClick={() => onOpenChange(false)}
              >
                Cancel
              </Button>
              <Button type="submit">Confirm Assignments</Button>
            </DialogFooter>
          </form>
        </CredenzaBody>
      </CredenzaContent>
    </Credenza>
  );
}
