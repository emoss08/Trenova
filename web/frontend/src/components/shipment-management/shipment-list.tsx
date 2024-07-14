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
