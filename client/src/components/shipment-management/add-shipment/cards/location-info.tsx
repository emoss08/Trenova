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

import { TimeField } from "@/components/common/fields/input";
import { SelectInput } from "@/components/common/fields/select-input";
import { LocationAutoComplete } from "@/components/ui/autocomplete";
import { TitleWithTooltip } from "@/components/ui/title-with-tooltip";
import { useLocations } from "@/hooks/useQueries";
import { ShipmentFormValues } from "@/types/order";
import { Control } from "react-hook-form";
import { useTranslation } from "react-i18next";

export function LocationInformation({
  control,
}: {
  control: Control<ShipmentFormValues>;
}) {
  const { t } = useTranslation("shipment.addshipment");
  const {
    selectLocationData,
    isError: isLocationError,
    isLoading: isLocationsLoading,
  } = useLocations();

  return (
    <div className="flex space-x-10">
      <div className="flex-1">
        <div className="flex flex-col">
          <div className="border-border rounded-md border">
            <div className="border-border bg-background flex justify-center rounded-t-md border-b p-2">
              <TitleWithTooltip
                title={t("card.origin.label")}
                tooltip={t("card.origin.description")}
              />
            </div>
            <div className="bg-card grid grid-cols-1 gap-y-4 p-4">
              <div className="col-span-3">
                <SelectInput
                  name="originLocation"
                  control={control}
                  options={selectLocationData}
                  isLoading={isLocationsLoading}
                  isFetchError={isLocationError}
                  label={t("card.origin.fields.originLocation.label")}
                  placeholder={t(
                    "card.origin.fields.originLocation.placeholder",
                  )}
                  description={t(
                    "card.origin.fields.originLocation.description",
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
                  name="originAddress"
                  rules={{ required: true }}
                  autoCapitalize="none"
                  autoCorrect="off"
                  type="text"
                  label={t("card.origin.fields.originAddress.label")}
                  placeholder={t(
                    "card.origin.fields.originAddress.placeholder",
                  )}
                  description={t(
                    "card.origin.fields.originAddress.description",
                  )}
                />
              </div>
              <div className="grid grid-cols-2 gap-x-4">
                <div className="col-span-1">
                  <TimeField
                    control={control}
                    rules={{ required: true }}
                    name="originAppointmentWindowStart"
                    label={t(
                      "card.origin.fields.originAppointmentWindowStart.label",
                    )}
                    placeholder={t(
                      "card.origin.fields.originAppointmentWindowStart.placeholder",
                    )}
                    description={t(
                      "card.origin.fields.originAppointmentWindowStart.description",
                    )}
                  />
                </div>
                <div className="col-span-1">
                  <TimeField
                    control={control}
                    rules={{ required: true }}
                    name="originAppointmentWindowEnd"
                    label={t(
                      "card.origin.fields.originAppointmentWindowEnd.label",
                    )}
                    placeholder={t(
                      "card.origin.fields.originAppointmentWindowEnd.placeholder",
                    )}
                    description={t(
                      "card.origin.fields.originAppointmentWindowEnd.description",
                    )}
                  />
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
      <div className="flex-1">
        <div className="flex flex-col">
          <div className="border-border rounded-md border">
            <div className="border-border bg-background flex justify-center rounded-t-md border-b p-2">
              <TitleWithTooltip
                title={t("card.destination.label")}
                tooltip={t("card.destination.description")}
              />
            </div>
            <div className="bg-card grid grid-cols-1 gap-y-4 p-4">
              <div className="col-span-3">
                <SelectInput
                  name="destinationLocation"
                  control={control}
                  options={selectLocationData}
                  isLoading={isLocationsLoading}
                  isFetchError={isLocationError}
                  label={t("card.destination.fields.destinationLocation.label")}
                  placeholder={t(
                    "card.destination.fields.destinationLocation.placeholder",
                  )}
                  description={t(
                    "card.destination.fields.destinationLocation.description",
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
                  name="destinationAddress"
                  rules={{ required: true }}
                  autoCapitalize="none"
                  autoCorrect="off"
                  type="text"
                  label={t("card.destination.fields.destinationAddress.label")}
                  placeholder={t(
                    "card.destination.fields.destinationAddress.placeholder",
                  )}
                  description={t(
                    "card.destination.fields.destinationAddress.description",
                  )}
                />
              </div>
              <div className="grid grid-cols-2 gap-x-4">
                <div className="col-span-1">
                  <TimeField
                    control={control}
                    rules={{ required: true }}
                    name="destinationAppointmentWindowStart"
                    label={t(
                      "card.destination.fields.destinationAppointmentWindowStart.label",
                    )}
                    placeholder={t(
                      "card.destination.fields.destinationAppointmentWindowStart.placeholder",
                    )}
                    description={t(
                      "card.destination.fields.destinationAppointmentWindowStart.description",
                    )}
                  />
                </div>
                <div className="col-span-1">
                  <TimeField
                    control={control}
                    rules={{ required: true }}
                    name="destinationAppointmentWindowEnd"
                    label={t(
                      "card.destination.fields.destinationAppointmentWindowEnd.label",
                    )}
                    placeholder={t(
                      "card.destination.fields.destinationAppointmentWindowEnd.placeholder",
                    )}
                    description={t(
                      "card.destination.fields.destinationAppointmentWindowEnd.description",
                    )}
                  />
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
