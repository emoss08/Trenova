import { getActiveAssignmentsForTractor } from "@/services/EquipmentRequestService";
import type {
  NewAssignment,
  Tractor,
  TractorAssignmentFormValues,
} from "@/types/equipment";
import { useQuery } from "@tanstack/react-query";
import { useEffect } from "react";
import { DragDropContext, Draggable, Droppable } from "react-beautiful-dnd";
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

interface AssignmentDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  handleAssignTractor: (
    assignments: TractorAssignmentFormValues["assignments"],
  ) => void;
  selectedTractor: Tractor;
  newAssignment: NewAssignment;
}

export function AssignmentDialog({
  open,
  onOpenChange,
  handleAssignTractor,
  selectedTractor,
  newAssignment,
}: AssignmentDialogProps) {
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

  const handleDragEnd = (result: any) => {
    if (!result.destination) return;
    move(result.source.index, result.destination.index);
  };

  if (isLoading && fields.length === 0) {
    return <div>Loading assignments...</div>;
  }

  return (
    <Credenza open={open} onOpenChange={onOpenChange}>
      <CredenzaContent>
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
                  <ul {...provided.droppableProps} ref={provided.innerRef}>
                    {fields.map((field, index) => (
                      <Draggable
                        key={field.id}
                        draggableId={field.id}
                        index={index}
                      >
                        {(provided) => (
                          <li
                            ref={provided.innerRef}
                            {...provided.draggableProps}
                            {...provided.dragHandleProps}
                            className="mb-2 flex items-center space-x-2"
                          >
                            <p></p>
                            <span>{index + 1}.</span>
                            <Input value={field.shipmentProNumber} readOnly />
                            <Button
                              size="xs"
                              variant="link"
                              onClick={() => remove(index)}
                            >
                              Remove
                            </Button>
                          </li>
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
