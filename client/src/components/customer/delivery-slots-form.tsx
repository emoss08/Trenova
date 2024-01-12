/*
 * COPYRIGHT(c) 2024 Trenova
 *
 * This file is part of Trenova.
 *
 * The Monta software is licensed under the Business Source License 1.1. You are granted the right
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

import { Control, useFieldArray } from "react-hook-form";
import { TimeField } from "../common/fields/input";

import { useLocations } from "@/hooks/useQueries";
import { DayOfWeekChoices } from "@/lib/choices";
import { CustomerFormValues as FormValues } from "@/types/customer";
import { PlusIcon } from "@radix-ui/react-icons";
import { AlertOctagonIcon, InfoIcon } from "lucide-react";
import { SelectInput } from "../common/fields/select-input";
import { Alert, AlertDescription, AlertTitle } from "../ui/alert";
import { Button } from "../ui/button";

function DeliverySlotAlert() {
  return (
    <Alert className="my-5  bg-foreground text-background">
      <InfoIcon className="h-5 w-5 stroke-background" />
      <AlertTitle>Information!</AlertTitle>
      <AlertDescription>
        Delivery slots are used to define the time slots for delivery. You can
        add multiple delivery slots for a location.
      </AlertDescription>
    </Alert>
  );
}

export function DeliverySlotForm({
  control,
  open,
}: {
  control: Control<FormValues>;
  open: boolean;
}) {
  const {
    selectLocationData,
    isLoading: isLocationsLoading,
    isError: isLocationError,
  } = useLocations(open);

  const { fields, append, remove } = useFieldArray({
    control,
    name: "deliverySlots",
    keyName: "id",
  });

  const handleAddSlot = () => {
    append({ dayOfWeek: 0, startTime: "", endTime: "", location: "" });
  };

  return (
    <>
      <DeliverySlotAlert />
      <div className="flex h-full w-full flex-col">
        {fields.length > 0 ? (
          <>
            <div className="max-h-[500px] overflow-y-auto">
              {fields.map((field, index) => (
                <div
                  key={field.id}
                  className="my-4 grid grid-cols-2 gap-2 border-b pb-2"
                >
                  <div className="flex w-full max-w-sm flex-col justify-between gap-0.5">
                    <div className="min-h-[4em]">
                      <SelectInput
                        name={`deliverySlots.${index}.dayOfWeek`}
                        rules={{ required: true }}
                        control={control}
                        label="Day of Week"
                        options={DayOfWeekChoices}
                        placeholder="Select Day of week"
                        description="Specify the operational day of the week for customer transactions."
                        isClearable={false}
                        menuPlacement="bottom"
                        menuPosition="fixed"
                      />
                    </div>
                  </div>
                  <div className="flex w-full max-w-sm flex-col justify-between gap-0.5">
                    <div className="min-h-[4em]">
                      <SelectInput
                        name={`deliverySlots.${index}.location`}
                        rules={{ required: true }}
                        control={control}
                        label="Location"
                        options={selectLocationData}
                        isFetchError={isLocationError}
                        isLoading={isLocationsLoading}
                        placeholder="Select Location"
                        description="Select the delivery location from the predefined list."
                        isClearable={false}
                        menuPlacement="bottom"
                        menuPosition="fixed"
                        hasPopoutWindow
                        popoutLink="/dispatch/locations/" // TODO: Change once Document Classification is added.
                        popoutLinkLabel="Location"
                      />
                    </div>
                  </div>
                  <div className="flex w-full max-w-sm flex-col justify-between gap-0.5">
                    <div className="min-h-[4em]">
                      <TimeField
                        rules={{ required: true }}
                        control={control}
                        name={`deliverySlots.${index}.startTime`}
                        label="Start Time"
                        placeholder="Start Time"
                        description="Enter the commencement time for the delivery window."
                      />
                    </div>
                  </div>
                  <div className="flex w-full max-w-sm flex-col justify-between gap-0.5">
                    <div className="min-h-[4em]">
                      <TimeField
                        rules={{ required: true }}
                        control={control}
                        name={`deliverySlots.${index}.endTime`}
                        label="End Time"
                        placeholder="End Time"
                        description="Enter the concluding time for the delivery window."
                      />
                    </div>
                  </div>
                  <div className="flex max-w-sm flex-col justify-between gap-1">
                    <div className="min-h-[4em]">
                      <Button
                        size="sm"
                        className="bg-background text-red-600 hover:bg-background hover:text-red-700"
                        type="button"
                        onClick={() => remove(index)}
                      >
                        Remove
                      </Button>
                    </div>
                  </div>
                </div>
              ))}
            </div>
            <Button
              type="button"
              size="sm"
              className="mb-10 w-fit"
              onClick={handleAddSlot}
            >
              <PlusIcon className="mr-2 h-4 w-4" />
              Add Another Delivery Slot
            </Button>
          </>
        ) : (
          <div className="mt-44 flex grow flex-col items-center justify-center">
            <span className="text-6xl mb-4">
              <AlertOctagonIcon />
            </span>
            <p className="mb-4">
              No delivery slots yet. Please add a new devliery slot.
            </p>
            <Button type="button" size="sm" onClick={handleAddSlot}>
              Add Delivery Slot
            </Button>
          </div>
        )}
      </div>
    </>
  );
}
