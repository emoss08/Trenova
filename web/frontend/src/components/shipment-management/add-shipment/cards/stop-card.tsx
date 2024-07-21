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

import { DecimalField } from "@/components/common/fields/decimal-input";
import { InputField } from "@/components/common/fields/input";
import { SelectInput } from "@/components/common/fields/select-input";
import { Icon } from "@/components/common/icons";
import { LocationAutoComplete } from "@/components/ui/autocomplete";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { useLocations } from "@/hooks/useQueries";
import { shipmentStatusChoices, shipmentStopChoices } from "@/lib/choices";
import { cn } from "@/lib/utils";
import { ShipmentFormValues } from "@/types/shipment";
import { faEllipsisV, faGrid } from "@fortawesome/pro-duotone-svg-icons";
import React, { useState } from "react";
import { Draggable } from "react-beautiful-dnd";
import { UseFieldArrayRemove, useFormContext } from "react-hook-form";
import { useTranslation } from "react-i18next";
import { RemoveStopDialog } from "../dialogs/stop-remove-dialog";

function StopForm({
  index,
  isDragDisabled,
}: {
  index: number;
  isDragDisabled: boolean;
}) {
  const { t } = useTranslation("shipment.addshipment");
  const { control } = useFormContext<ShipmentFormValues>();
  const { selectLocationData } = useLocations();

  return (
    <div className="flex flex-col">
      <div className="grid grid-cols-2 gap-4">
        <div className="col-span-1">
          <SelectInput
            name={`stops.${index}.status`}
            control={control}
            options={shipmentStatusChoices}
            rules={{ required: true }}
            label={t("card.stopInfo.fields.status.label")}
            placeholder={t("card.stopInfo.fields.status.placeholder")}
            description={t("card.stopInfo.fields.status.description")}
            isReadOnly
            defaultValue="N"
          />
        </div>
        <div className="col-span-1">
          <SelectInput
            name={`stops.${index}.stopType`}
            control={control}
            options={shipmentStopChoices}
            isReadOnly={isDragDisabled}
            rules={{ required: true }}
            label={t("card.stopInfo.fields.stopType.label")}
            placeholder={t("card.stopInfo.fields.stopType.placeholder")}
            description={t("card.stopInfo.fields.stopType.description")}
          />
        </div>
        <div className="col-span-full">
          <SelectInput
            name={`stops.${index}.location`}
            options={selectLocationData}
            control={control}
            rules={{ required: true }}
            label={t("card.stopInfo.fields.stopLocation.label")}
            placeholder={t("card.stopInfo.fields.stopLocation.placeholder")}
            description={t("card.stopInfo.fields.stopLocation.description")}
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
            label={t("card.stopInfo.fields.stopAddress.label")}
            placeholder={t("card.stopInfo.fields.stopAddress.placeholder")}
            description={t("card.stopInfo.fields.stopAddress.description")}
          />
        </div>
        <div className="col-span-1">
          <InputField
            name={`stops.${index}.pieces`}
            type="number"
            control={control}
            label={t("card.stopInfo.fields.pieces.label")}
            placeholder={t("card.stopInfo.fields.pieces.placeholder")}
            description={t("card.stopInfo.fields.pieces.description")}
          />
        </div>
        <div className="col-span-1">
          <DecimalField
            name={`stops.${index}.weight`}
            type="number"
            control={control}
            label={t("card.stopInfo.fields.weight.label")}
            placeholder={t("card.stopInfo.fields.weight.placeholder")}
            description={t("card.stopInfo.fields.weight.description")}
          />
        </div>
        <div className="col-span-1">
          <InputField
            control={control}
            type="datetime-local"
            rules={{ required: true }}
            name={`stops.${index}.appointmentTimeWindowStart`}
            label={t("card.stopInfo.fields.appointmentWindowStart.label")}
            description={t(
              "card.stopInfo.fields.appointmentWindowStart.description",
            )}
          />
        </div>
        <div className="col-span-1">
          <InputField
            control={control}
            type="datetime-local"
            rules={{ required: true }}
            name={`stops.${index}.appointmentTimeWindowEnd`}
            label={t("card.stopInfo.fields.appointmentWindowEnd.label")}
            description={t(
              "card.stopInfo.fields.appointmentWindowEnd.description",
            )}
          />
        </div>
      </div>
    </div>
  );
}

export function StopCard({
  field,
  index,
  remove,
  totalStops,
}: {
  field: Record<string, any>;
  index: number;
  remove: UseFieldArrayRemove;
  totalStops: number;
}) {
  const [removeStopIndex, setRemoveStopIndex] = useState<number | null>(null);

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

  const isDragDisabled = index === 0 || index === totalStops - 1;

  return (
    <Draggable
      key={field.id}
      draggableId={field.id.toString()}
      index={index}
      isDragDisabled={isDragDisabled}
    >
      {(provided, snapshot) => (
        <React.Fragment key={field.id}>
          <div
            className={cn(
              "bg-card border-border rounded-md border p-4",
              snapshot.isDragging && "opacity-50",
            )}
            ref={provided.innerRef}
            {...provided.draggableProps}
          >
            <div className="mb-5 flex justify-between border-b pb-2">
              <span {...provided.dragHandleProps}>
                {!isDragDisabled && (
                  <Icon
                    icon={faGrid}
                    className={cn(
                      "text-muted-foreground hover:text-foreground hover:cursor-pointer size-5",
                      snapshot.isDragging && "text-lime-400",
                    )}
                  />
                )}
              </span>
              <DropdownMenu>
                <DropdownMenuTrigger asChild>
                  <span>
                    <Icon
                      icon={faEllipsisV}
                      className="text-muted-foreground hover:text-foreground size-5 hover:cursor-pointer"
                    />
                  </span>
                </DropdownMenuTrigger>
                <DropdownMenuContent>
                  <DropdownMenuLabel>Actions</DropdownMenuLabel>
                  <DropdownMenuSeparator />
                  <DropdownMenuItem disabled>Add Comment</DropdownMenuItem>
                  <DropdownMenuItem
                    onClick={() => openRemoveAlert(index)}
                    className="focus:bg-destructive/10 focus:text-destructive font-semibold text-red-500"
                    disabled={isDragDisabled}
                  >
                    Remove
                  </DropdownMenuItem>
                </DropdownMenuContent>
              </DropdownMenu>
            </div>
            <StopForm index={index} isDragDisabled={isDragDisabled} />
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
}
