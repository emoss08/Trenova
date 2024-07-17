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



import { Shipment, ShipmentStatus } from "@/types/shipment";
import { Droppable } from "react-beautiful-dnd";
import { ShipmentInfo } from "./shipment-info";

export function ShipmentList({
  shipments,
  finalStatuses,
  progressStatuses,
}: {
  shipments: Shipment[];
  finalStatuses: ShipmentStatus[];
  progressStatuses: ShipmentStatus[];
}) {
  return (
    <Droppable droppableId="shipmentList">
      {(provided) => (
        <div ref={provided.innerRef} {...provided.droppableProps}>
          {shipments.length > 0 ? (
            shipments.map((shipment) => (
              <Droppable key={shipment.id} droppableId={shipment.id}>
                {(provided, snapshot) => (
                  <div
                    ref={provided.innerRef}
                    {...provided.droppableProps}
                    className={`mb-2 transition-colors duration-200 ${
                      snapshot.isDraggingOver ? "bg-green-500/50" : ""
                    }`}
                  >
                    <ShipmentInfo
                      shipment={shipment}
                      finalStatuses={finalStatuses}
                      progressStatuses={progressStatuses}
                    />
                    <div style={{ display: "none" }}>
                      {provided.placeholder}
                    </div>
                  </div>
                )}
              </Droppable>
            ))
          ) : (
            <div className="text-muted-foreground py-8 text-center">
              No shipments found for the given criteria.
            </div>
          )}
          {provided.placeholder}
        </div>
      )}
    </Droppable>
  );
}
