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

import { InputField } from "@/components/common/fields/input";
import { SelectInput } from "@/components/common/fields/select-input";
import { LocationAutoComplete } from "@/components/ui/autocomplete";
import { Skeleton } from "@/components/ui/skeleton";
import { TitleWithTooltip } from "@/components/ui/title-with-tooltip";
import { useLocations } from "@/hooks/useQueries";
import { Location } from "@/types/location";
import { ShipmentControl, ShipmentFormValues } from "@/types/shipment";
import { StopFormValues } from "@/types/stop";
import { useEffect } from "react";
import { Control, useFieldArray, useFormContext } from "react-hook-form";
import { useTranslation } from "react-i18next";

function LocationSection({
  section,
  control,
  selectLocationData,
  isLocationError,
  isLocationsLoading,
}: {
  section: "origin" | "destination";
  control: Control<ShipmentFormValues>;
  selectLocationData: Array<{ value: string; label: string }>;
  isLocationError: boolean;
  isLocationsLoading: boolean;
}) {
  const { t } = useTranslation("shipment.addshipment");

  return (
    <div className="flex-1">
      <div className="flex flex-col">
        <div className="border-border rounded-md border">
          <div className="border-border bg-background flex justify-center rounded-t-md border-b p-2">
            <TitleWithTooltip
              title={t(`card.${section}.label`)}
              tooltip={t(`card.${section}.description`)}
            />
          </div>
          <div className="bg-card grid grid-cols-1 gap-y-4 p-4">
            <div className="col-span-3">
              <SelectInput
                name={`${section}Location`}
                control={control}
                options={selectLocationData}
                isFetchError={isLocationError}
                isLoading={isLocationsLoading}
                label={t(`card.${section}.fields.${section}Location.label`)}
                placeholder={t(
                  `card.${section}.fields.${section}Location.placeholder`,
                )}
                description={t(
                  `card.${section}.fields.${section}Location.description`,
                )}
                hasPopoutWindow
                popoutLink="/dispatch/locations/"
                isClearable
                popoutLinkLabel="Location"
              />
            </div>
            <div className="col-span-3">
              <LocationAutoComplete
                control={control}
                name={`${section}Address`}
                rules={{ required: true }}
                autoCapitalize="none"
                autoCorrect="off"
                type="text"
                label={t(`card.${section}.fields.${section}Address.label`)}
                placeholder={t(
                  `card.${section}.fields.${section}Address.placeholder`,
                )}
                description={t(
                  `card.${section}.fields.${section}Address.description`,
                )}
              />
            </div>
            <div className="grid grid-cols-2 gap-x-4">
              <div className="col-span-1">
                <InputField
                  control={control}
                  rules={{ required: true }}
                  name={`${section}AppointmentWindowStart`}
                  type="datetime-local"
                  label={t(
                    `card.${section}.fields.${section}AppointmentWindowStart.label`,
                  )}
                  description={t(
                    `card.${section}.fields.${section}AppointmentWindowStart.description`,
                  )}
                />
              </div>
              <div className="col-span-1">
                <InputField
                  control={control}
                  rules={{ required: true }}
                  type="datetime-local"
                  name={`${section}AppointmentWindowEnd`}
                  label={t(
                    `card.${section}.fields.${section}AppointmentWindowEnd.label`,
                  )}
                  description={t(
                    `card.${section}.fields.${section}AppointmentWindowEnd.description`,
                  )}
                />
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}

export default function LocationInformation({
  shipmentControlData,
  isShipmentControlLoading,
}: {
  shipmentControlData: ShipmentControl;
  isShipmentControlLoading: boolean;
}) {
  const {
    locations,
    selectLocationData,
    isLoading: isLocationsLoading,
    isError: isLocationError,
  } = useLocations();
  const { control, setValue, watch, setError } =
    useFormContext<ShipmentFormValues>();

  console.info("stops", watch("stops"));

  const { fields, append } = useFieldArray({
    control,
    name: "stops",
    keyName: "id",
  });

  useEffect(() => {
    const subscription = watch((value, { name }) => {
      // Set the origin address based on the selected origin location
      if (name === "originLocation" && locations && value.originLocation) {
        const selectedOriginLocation = (locations as Location[]).find(
          (location) => location.id === value.originLocation,
        );

        if (selectedOriginLocation) {
          setValue(
            "originAddress",
            `${selectedOriginLocation.addressLine1}, ${selectedOriginLocation.city}, ${selectedOriginLocation.state} ${selectedOriginLocation.zipCode}`,
          );
        }
      }

      // Set the destination address based on the selected destination location
      if (
        name === "destinationLocation" &&
        locations &&
        value.destinationLocation
      ) {
        const selectedDestinationLocation = (locations as Location[]).find(
          (location) => location.id === value.destinationLocation,
        );

        console.info(
          "selectedDestinationLocation",
          selectedDestinationLocation,
        );

        if (selectedDestinationLocation) {
          setValue(
            "destinationAddress",
            `${selectedDestinationLocation.addressLine1}, ${selectedDestinationLocation.city}, ${selectedDestinationLocation.state} ${selectedDestinationLocation.zipCode}`,
          );
        }
      }

      if (
        shipmentControlData &&
        shipmentControlData?.enforceOriginDestination &&
        value.originLocation &&
        value.destinationLocation &&
        value.originLocation === value.destinationLocation
      ) {
        setError("originLocation", {
          type: "manual",
          message: "Origin and Destination locations cannot be the same.",
        });
        setError("destinationLocation", {
          type: "manual",
          message: "Origin and Destination locations cannot be the same.",
        });
      }
    });

    return () => subscription.unsubscribe();
  }, [watch, locations, setValue, shipmentControlData, setError]);

  useEffect(() => {
    const subscription = watch((value, { name }) => {
      // Logic for origin stop
      if (name?.includes("origin")) {
        const originStopIndex = fields.findIndex(
          (field) => field.stopType === "P",
        );
        const isOriginComplete =
          value.originLocation &&
          value.originAddress &&
          value.originAppointmentWindowStart &&
          value.originAppointmentWindowEnd;

        const originStop: StopFormValues = {
          addressLine: value?.originAddress,
          location: value?.originLocation,
          appointmentTimeWindowStart: value.originAppointmentWindowStart || "",
          appointmentTimeWindowEnd: value.originAppointmentWindowEnd || "",
          pieces: undefined,
          status: "N",
          weight: "",
          sequence: 1,
          stopType: "P",
        };

        if (isOriginComplete) {
          if (originStopIndex === -1) {
            append(originStop);
          } else {
            setValue(`stops.${originStopIndex}`, originStop);
          }
        }
      }

      if (name?.includes("destination")) {
        const destinationStopIndex = fields.findIndex(
          (field) => field.stopType === "D",
        );
        const isDestinationComplete =
          value.destinationLocation &&
          value.destinationAddress &&
          value.destinationAppointmentWindowStart &&
          value.destinationAppointmentWindowEnd;

        const destinationStop: StopFormValues = {
          addressLine: value.destinationAddress,
          location: value.destinationLocation,
          appointmentTimeWindowStart:
            value.destinationAppointmentWindowStart || "",
          appointmentTimeWindowEnd: value.destinationAppointmentWindowEnd || "",
          pieces: undefined,
          status: "N",
          weight: "",
          sequence: fields.length + 1,
          stopType: "D",
        };

        if (isDestinationComplete) {
          if (destinationStopIndex === -1) {
            append(destinationStop);
          } else {
            setValue(`stops.${destinationStopIndex}`, destinationStop);
          }
        }
      }
    });

    return () => subscription.unsubscribe();
  }, [watch, append, fields, setValue]);

  if (isShipmentControlLoading) {
    return <Skeleton className="h-[40vh]" />;
  }

  return (
    <div className="flex gap-x-10">
      <LocationSection
        selectLocationData={selectLocationData}
        isLocationError={isLocationError}
        isLocationsLoading={isLocationsLoading}
        section="origin"
        control={control}
      />
      <LocationSection
        selectLocationData={selectLocationData}
        isLocationError={isLocationError}
        isLocationsLoading={isLocationsLoading}
        section="destination"
        control={control}
      />
    </div>
  );
}
