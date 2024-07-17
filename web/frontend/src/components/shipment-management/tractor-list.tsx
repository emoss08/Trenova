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
