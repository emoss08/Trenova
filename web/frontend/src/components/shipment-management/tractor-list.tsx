/**
 * Copyright (c) 2024 Trenova Technologies, LLC
 *
 * Licensed under the Business Source License 1.1 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     https://trenova.app/pricing/
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *
 * Key Terms:
 * - Non-production use only
 * - Change Date: 2026-11-16
 * - Change License: GNU General Public License v2 or later
 *
 * For full license text, see the LICENSE file in the root directory.
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
