import { Button } from "@/components/ui/button";
import { useLocations } from "@/hooks/useQueries";
import { cn } from "@/lib/utils";
import { Location } from "@/types/location";
import { ShipmentFormValues } from "@/types/shipment";
import { useEffect } from "react";
import { DragDropContext, Droppable, DropResult } from "react-beautiful-dnd";
import { useFieldArray, useFormContext } from "react-hook-form";
import { StopCard } from "./cards/stop-card";

export default function StopInfoTab() {
  const { control, watch, setValue } = useFormContext<ShipmentFormValues>();

  const { fields, remove, move, insert } = useFieldArray({
    control,
    name: "stops",
    keyName: "id",
  });

  const { locations } = useLocations();

  const addNewStop = () => {
    insert(fields.length - 1, {
      status: "N",
      stopType: "P",
      location: undefined,
      addressLine: "",
      pieces: undefined,
      weight: "",
      appointmentTimeWindowStart: "",
      appointmentTimeWindowEnd: "",
      sequence: fields.length - 1,
    });
  };

  const handleDrag = ({ source, destination }: DropResult) => {
    if (destination) {
      move(source.index, destination.index);
      // After moving update the sequence number for each stop to where it was moved in the grid.
      fields.forEach((_, index) => {
        setValue(`stops.${index}.sequence`, index + 1, {
          shouldDirty: true,
        });
      });
    }
  };

  useEffect(() => {
    const subscription = watch((value, { name }) => {
      if (name?.startsWith("stops") && name.endsWith("location")) {
        const stopIndex = Number(name.split(".")[1]); // Explicitly specify the type of stopIndex as a number
        const selectedLocationId = value.stops?.[stopIndex]?.location;

        if (selectedLocationId) {
          const location = (locations as Location[]).find(
            (loc) => loc.id === selectedLocationId,
          );
          if (location) {
            const newAddressLine = `${location.addressLine1}, ${location.city}, ${location.stateId} ${location.postalCode}`;
            setValue(`stops.${stopIndex}.addressLine`, newAddressLine, {
              shouldDirty: true,
            });
          }
        }
      }
    });

    return () => subscription.unsubscribe();
  }, [locations, setValue, watch]);

  return (
    <>
      <DragDropContext onDragEnd={handleDrag}>
        <ul>
          <Droppable droppableId="stops" direction="horizontal">
            {(provided, snapshot) => (
              <div
                {...provided.droppableProps}
                ref={provided.innerRef}
                className={cn(
                  "size-[100%]",
                  snapshot.isUsingPlaceholder && "overflow-hidden",
                )}
              >
                {fields.length > 0 && (
                  <div className="grid grid-cols-1 gap-6 lg:grid-cols-2 xl:grid-cols-3">
                    {fields.map((field, index) => {
                      return (
                        <StopCard
                          key={field.id}
                          index={index}
                          field={field}
                          remove={remove}
                          totalStops={fields.length}
                        />
                      );
                    })}
                    {provided.placeholder}
                  </div>
                )}
              </div>
            )}
          </Droppable>
        </ul>
      </DragDropContext>
      <div className="mt-4 flex justify-center">
        <Button type="button" size="sm" variant="outline" onClick={addNewStop}>
          Add Stop
        </Button>
      </div>
    </>
  );
}
