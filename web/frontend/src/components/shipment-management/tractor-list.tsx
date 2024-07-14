import type { Tractor, TractorFilterForm } from "@/types/equipment";
import { Draggable, Droppable } from "react-beautiful-dnd";
import { type UseFormReturn } from "react-hook-form";
import { Input } from "../common/fields/input";
import { Select } from "../common/fields/select";

interface TractorListProps {
  tractors: Tractor[];
  form: UseFormReturn<TractorFilterForm>;
}

export function TractorList({ tractors, form }: TractorListProps) {
  const { register } = form;

  return (
    <div>
      <div className="mb-4 space-y-2">
        <Input {...register("searchQuery")} placeholder="Search tractors..." />
        <Select {...register("status")}>
          <option value="Available">Available</option>
          <option value="InUse">In Use</option>
          <option value="Maintenance">Maintenance</option>
        </Select>
        <Input {...register("fleetCodeId")} placeholder="Fleet Code ID" />
      </div>
      <Droppable droppableId="tractorList">
        {(provided) => (
          <ul
            {...provided.droppableProps}
            ref={provided.innerRef}
            className="space-y-2"
          >
            {tractors.map((tractor, index) => (
              <Draggable
                key={tractor.id}
                draggableId={tractor.id}
                index={index}
              >
                {(provided) => (
                  <li
                    ref={provided.innerRef}
                    {...provided.draggableProps}
                    {...provided.dragHandleProps}
                    className="bg-background rounded p-2 shadow"
                  >
                    {tractor.code}
                  </li>
                )}
              </Draggable>
            ))}
            {provided.placeholder}
          </ul>
        )}
      </Droppable>
    </div>
  );
}
