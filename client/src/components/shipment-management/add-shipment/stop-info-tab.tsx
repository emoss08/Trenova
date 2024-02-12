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
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { useLocations } from "@/hooks/useQueries";
import { shipmentStopChoices } from "@/lib/choices";
import { Location } from "@/types/location";
import { ShipmentFormValues } from "@/types/order";
import { useEffect } from "react";
import { useFieldArray, useFormContext } from "react-hook-form";
import { useTranslation } from "react-i18next";

function StopCard({ field, index }: { field: any; index: number }) {
  return (
    <div className="bg-card hover:bg-muted/20 relative mb-2 flex select-none flex-col rounded-lg border shadow-sm hover:cursor-pointer">
      <div className="flex justify-between border-b p-2 text-sm">
        <span className="font-semibold">{field[index].status}</span>
        <Badge variant="active">{field[index].status}</Badge>
      </div>
      <div className="flex flex-row">
        <div className="flex grow flex-col space-y-2 p-4">
          <span
            className="text-muted-foreground text-sm"
            aria-label="Location Code"
          >
            {field[index].location}
          </span>
          <span
            className="text-foreground text-lg font-semibold"
            aria-label="Location Address"
          >
            {field[index].addressLine}
          </span>
          <div className="flex justify-between text-sm">
            <span className="font-semibold">Appointment Window</span>
            <span className="text-muted-foreground">
              {field[index].appointmentTimeWindowStart} -{" "}
              {field[index].appointmentTimeWindowEnd}
            </span>
          </div>
          <div className="flex justify-between border-t pt-2 text-sm">
            <span className="font-semibold">Summary</span>
            <span className="text-muted-foreground">
              {field[index].pieces} Pieces, {field[index].weight} lbs
            </span>
          </div>
        </div>
      </div>
    </div>
  );
}

export default function StopInfoTab() {
  const { control, watch, setValue, getValues } =
    useFormContext<ShipmentFormValues>();
  const { t } = useTranslation("shipment.addshipment");
  const { locations } = useLocations();

  const { fields, append } = useFieldArray({
    control,
    name: "stops",
    keyName: "id",
  });

  const handleAddStop = () => {
    append({
      status: "N",
      stopType: "",
      location: "",
      addressLine: "",
      pieces: 0,
      weight: "0.00",
      appointmentTimeWindowStart: "",
      appointmentTimeWindowEnd: "",
    });
  };

  // Watch the stops field array
  const stopArray = watch("stops", []);
  const stopFields = [...(stopArray || [])];

  // This function finds the address for a given location ID
  const findLocationAddress = (locationId: string) => {
    const location = (locations as Location[]).find(
      (loc) => loc.id === locationId,
    );
    if (location) {
      return `${location.addressLine1}, ${location.city}, ${location.state} ${location.zipCode}`;
    }
    return "";
  };

  useEffect(() => {
    stopFields.forEach((stop, index) => {
      if (stop.location) {
        const newAddressLine = findLocationAddress(stop.location);
        // Get the current address value for comparison
        const currentAddressLine = getValues(`stops.${index}.addressLine`);

        // Only update if the address has actually changed
        if (newAddressLine !== currentAddressLine) {
          setValue(`stops.${index}.addressLine`, newAddressLine, {
            shouldDirty: true,
          });
        }
      }
    });
  }, [stopFields, locations, setValue]);

  return (
    <>
      {/* Render StopCards for each stop if any */}
      {fields.length > 0 && (
        <div className="space-y-4">
          {fields.map((field, index) => (
            <div
              key={field.id}
              className="border-border bg-card rounded-md border p-4"
            >
              <div className="flex flex-col">
                <div className="grid grid-cols-2 gap-4">
                  <div className="col-span-1">
                    <SelectInput
                      name={`stops.${index}.stopType`}
                      control={control}
                      options={shipmentStopChoices}
                      rules={{ required: true }}
                      label={t("card.stopInfo.fields.stopType.label")}
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
                      label={t("card.stopInfo.fields.stopLocation.label")}
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
                      label={t("card.stopInfo.fields.stopAddress.label")}
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
                      label={t("card.stopInfo.fields.pieces.label")}
                      placeholder={t("card.stopInfo.fields.pieces.placeholder")}
                      description={t("card.stopInfo.fields.pieces.description")}
                    />
                  </div>
                  <div className="col-span-1">
                    <InputField
                      name={`stops.${index}.weight`}
                      type="number"
                      control={control}
                      rules={{ required: true }}
                      label={t("card.stopInfo.fields.weight.label")}
                      placeholder={t("card.stopInfo.fields.weight.placeholder")}
                      description={t("card.stopInfo.fields.weight.description")}
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
          ))}
        </div>
      )}

      {/* Always display the Add Stop button */}
      <div className="mt-4 flex justify-center">
        <Button
          type="button"
          size="sm"
          variant="outline"
          onClick={handleAddStop}
        >
          Add Stop
        </Button>
      </div>
    </>
  );
}
