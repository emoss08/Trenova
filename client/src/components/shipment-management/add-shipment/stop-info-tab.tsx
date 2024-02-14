/*
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

import { InputField, TimeField } from "@/components/common/fields/input";
import {
  AsyncSelectInput,
  SelectInput,
} from "@/components/common/fields/select-input";
import { LocationAutoComplete } from "@/components/ui/autocomplete";
import { Button } from "@/components/ui/button";
import { useLocations } from "@/hooks/useQueries";
import { shipmentStatusChoices, shipmentStopChoices } from "@/lib/choices";
import { cn } from "@/lib/utils";
import { Location } from "@/types/location";
import { ShipmentFormValues } from "@/types/order";
import { faGrid } from "@fortawesome/pro-duotone-svg-icons";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import React, { useEffect, useState } from "react";
import {
  DragDropContext,
  Draggable,
  DropResult,
  Droppable,
} from "react-beautiful-dnd";
import { useFieldArray, useFormContext } from "react-hook-form";
import { useTranslation } from "react-i18next";
import { RemoveStopDialog } from "./dialogs/stop-remove-dialog";

export default function StopInfoTab() {
  const { control, watch, setValue } = useFormContext<ShipmentFormValues>();
  const [removeStopIndex, setRemoveStopIndex] = useState<number | null>(null);

  console.info("stops", watch("stops"));

  const { fields, append, remove, move } = useFieldArray({
    control,
    name: "stops",
    keyName: "id",
  });

  const { t } = useTranslation("shipment.addshipment");
  const { locations } = useLocations();

  const addNewStop = () => {
    append({
      status: "N",
      stopType: "P",
      location: null,
      addressLine: "",
      pieces: 0,
      weight: "0.00",
      appointmentTimeWindowStart: "",
      appointmentTimeWindowEnd: "",
      sequence: fields.length + 1,
    });
  };

  const openRemoveAlert = (index: number) => {
    setRemoveStopIndex(index);
  };

  const closeRemoveAlert = () => {
    setRemoveStopIndex(null);
  };

  const removeSelectedStop = () => {
    if (removeStopIndex !== null) {
      remove(removeStopIndex);
      closeRemoveAlert();
    }
  };

  const handleDrag = ({ source, destination }: DropResult) => {
    if (destination) {
      move(source.index, destination.index);
      // after moving update the sequence number for each stop to where it was moved in the grid.
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
            const newAddressLine = `${location.addressLine1}, ${location.city}, ${location.state} ${location.zipCode}`;
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
                  "mt-4 size-[100%]",
                  snapshot.isUsingPlaceholder && "overflow-hidden",
                )}
              >
                {fields.length > 0 && (
                  <div className="grid grid-cols-1 gap-6 lg:grid-cols-2 xl:grid-cols-3">
                    {fields.map((field, index) => {
                      return (
                        <Draggable
                          key={field.id}
                          draggableId={field.id.toString()}
                          index={index}
                        >
                          {(provided, snapshot) => (
                            <React.Fragment key={field.id}>
                              <div
                                className={cn(
                                  "bg-card border-border rounded-md border p-4",
                                  snapshot.isDragging && "opacity-30",
                                )}
                                ref={provided.innerRef}
                                {...provided.draggableProps}
                              >
                                <div className="mb-5 flex justify-between border-b pb-2">
                                  <span {...provided.dragHandleProps}>
                                    <FontAwesomeIcon
                                      icon={faGrid}
                                      className={cn(
                                        "text-muted-foreground hover:text-foreground hover:cursor-pointer",
                                        snapshot.isDragging && "text-lime-400",
                                      )}
                                    />
                                  </span>
                                  <Button
                                    type="button"
                                    size="xs"
                                    variant="link"
                                    onClick={() => openRemoveAlert(index)}
                                  >
                                    Remove
                                  </Button>
                                </div>
                                <div className="flex flex-col">
                                  <div className="grid grid-cols-2 gap-4">
                                    <div className="col-span-1">
                                      <SelectInput
                                        name={`stops.${index}.status`}
                                        control={control}
                                        options={shipmentStatusChoices}
                                        rules={{ required: true }}
                                        label={t(
                                          "card.stopInfo.fields.status.label",
                                        )}
                                        placeholder={t(
                                          "card.stopInfo.fields.status.placeholder",
                                        )}
                                        description={t(
                                          "card.stopInfo.fields.status.description",
                                        )}
                                        isReadOnly
                                        defaultValue="N"
                                      />
                                    </div>
                                    <div className="col-span-1">
                                      <SelectInput
                                        name={`stops.${index}.stopType`}
                                        control={control}
                                        options={shipmentStopChoices}
                                        rules={{ required: true }}
                                        label={t(
                                          "card.stopInfo.fields.stopType.label",
                                        )}
                                        placeholder={t(
                                          "card.stopInfo.fields.stopType.placeholder",
                                        )}
                                        description={t(
                                          "card.stopInfo.fields.stopType.description",
                                        )}
                                      />
                                    </div>
                                    <div className="col-span-full">
                                      <AsyncSelectInput
                                        name={`stops.${index}.location`}
                                        link="/locations/"
                                        control={control}
                                        rules={{ required: true }}
                                        label={t(
                                          "card.stopInfo.fields.stopLocation.label",
                                        )}
                                        placeholder={t(
                                          "card.stopInfo.fields.stopLocation.placeholder",
                                        )}
                                        description={t(
                                          "card.stopInfo.fields.stopLocation.description",
                                        )}
                                        hasPopoutWindow
                                        popoutLink="/dispatch/locations/"
                                        isClearable
                                        popoutLinkLabel="Location"
                                      />
                                    </div>
                                    <div className="col-span-full">
                                      <LocationAutoComplete
                                        name={`stops.${index}.addressLine`}
                                        control={control}
                                        rules={{ required: true }}
                                        autoCapitalize="none"
                                        autoCorrect="off"
                                        type="text"
                                        label={t(
                                          "card.stopInfo.fields.stopAddress.label",
                                        )}
                                        placeholder={t(
                                          "card.stopInfo.fields.stopAddress.placeholder",
                                        )}
                                        description={t(
                                          "card.stopInfo.fields.stopAddress.description",
                                        )}
                                      />
                                    </div>
                                    <div className="col-span-1">
                                      <InputField
                                        name={`stops.${index}.pieces`}
                                        type="number"
                                        control={control}
                                        rules={{ required: true }}
                                        label={t(
                                          "card.stopInfo.fields.pieces.label",
                                        )}
                                        placeholder={t(
                                          "card.stopInfo.fields.pieces.placeholder",
                                        )}
                                        description={t(
                                          "card.stopInfo.fields.pieces.description",
                                        )}
                                      />
                                    </div>
                                    <div className="col-span-1">
                                      <InputField
                                        name={`stops.${index}.weight`}
                                        type="number"
                                        control={control}
                                        rules={{ required: true }}
                                        label={t(
                                          "card.stopInfo.fields.weight.label",
                                        )}
                                        placeholder={t(
                                          "card.stopInfo.fields.weight.placeholder",
                                        )}
                                        description={t(
                                          "card.stopInfo.fields.weight.description",
                                        )}
                                      />
                                    </div>
                                    <div className="col-span-1">
                                      <TimeField
                                        control={control}
                                        rules={{ required: true }}
                                        name={`stops.${index}.appointmentTimeWindowStart`}
                                        label={t(
                                          "card.stopInfo.fields.appointmentWindowStart.label",
                                        )}
                                        placeholder={t(
                                          "card.stopInfo.fields.appointmentWindowStart.placeholder",
                                        )}
                                        description={t(
                                          "card.stopInfo.fields.appointmentWindowStart.description",
                                        )}
                                      />
                                    </div>
                                    <div className="col-span-1">
                                      <TimeField
                                        control={control}
                                        rules={{ required: true }}
                                        name={`stops.${index}.appointmentTimeWindowEnd`}
                                        label={t(
                                          "card.stopInfo.fields.appointmentWindowEnd.label",
                                        )}
                                        placeholder={t(
                                          "card.stopInfo.fields.appointmentWindowEnd.placeholder",
                                        )}
                                        description={t(
                                          "card.stopInfo.fields.appointmentWindowEnd.description",
                                        )}
                                      />
                                    </div>
                                  </div>
                                </div>
                              </div>
                              {removeStopIndex === index && (
                                <RemoveStopDialog
                                  open={removeStopIndex === index}
                                  onClose={closeRemoveAlert}
                                  removeStop={removeSelectedStop}
                                />
                              )}
                            </React.Fragment>
                          )}
                        </Draggable>
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
