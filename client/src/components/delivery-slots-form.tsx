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

import {
  Control,
  useFieldArray,
  UseFieldArrayRemove,
  useFormContext,
} from "react-hook-form";

import { useLocations } from "@/hooks/useQueries";
import { DayOfWeekChoices } from "@/lib/choices";
import { CustomerFormValues } from "@/types/customer";
import { faRoadCircleXmark } from "@fortawesome/pro-duotone-svg-icons";
import { faInfo } from "@fortawesome/pro-regular-svg-icons";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { PlusIcon } from "@radix-ui/react-icons";
import { TimeField } from "./common/fields/input";
import { SelectInput } from "./common/fields/select-input";
import { Alert, AlertDescription, AlertTitle } from "./ui/alert";
import { Button } from "./ui/button";
import { ScrollArea } from "./ui/scroll-area";

function DeliverySlotAlert() {
  return (
    <Alert className="my-2">
      <FontAwesomeIcon icon={faInfo} className="size-4" />
      <AlertTitle>Information!</AlertTitle>
      <AlertDescription>
        Delivery slots are used to define the time slots for delivery. You can
        add multiple delivery slots for a location.
      </AlertDescription>
    </Alert>
  );
}

function DeliverySlotItem({
  control,
  index,
  field,
  selectLocationData,
  isLocationsLoading,
  isLocationError,
  remove,
}: {
  control: Control<CustomerFormValues>;
  index: number;
  field: Record<string, any>;
  selectLocationData: { value: string; label: string }[];
  isLocationsLoading: boolean;
  isLocationError: boolean;
  remove: UseFieldArrayRemove;
}) {
  return (
    <div
      key={field.id}
      className="border-border mb-4 grid grid-cols-2 gap-2 rounded-md border p-2"
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
            popoutLink="/dispatch/locations/"
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
        <div className="min-h-[2em]">
          <Button
            size="sm"
            variant="linkHover2"
            type="button"
            onClick={() => remove(index)}
          >
            Remove
          </Button>
        </div>
      </div>
    </div>
  );
}

export function DeliverySlotForm({ open }: { open: boolean }) {
  const { control } = useFormContext<CustomerFormValues>();

  const {
    selectLocationData,
    isLoading: isLocationsLoading,
    isError: isLocationError,
  } = useLocations("A", open);

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
      <div className="flex size-full flex-col">
        {fields.length > 0 ? (
          <>
            <ScrollArea className="h-[55vh] p-4">
              {fields.map((field, index) => (
                <DeliverySlotItem
                  key={field.id}
                  control={control}
                  index={index}
                  field={field}
                  selectLocationData={selectLocationData}
                  isLocationsLoading={isLocationsLoading}
                  isLocationError={isLocationError}
                  remove={remove}
                />
              ))}
            </ScrollArea>
            <Button
              type="button"
              size="sm"
              className="mb-10 w-fit"
              onClick={handleAddSlot}
            >
              <PlusIcon className="mr-2 size-4" />
              Add Another Delivery Slot
            </Button>
          </>
        ) : (
          <div className="mt-44 flex grow flex-col items-center justify-center">
            <FontAwesomeIcon
              icon={faRoadCircleXmark}
              className="fill-foreground size-10"
            />
            <h3 className="mt-4 text-lg font-semibold">
              No Delivery Slot added
            </h3>
            <p className="text-muted-foreground mb-4 mt-2 text-sm">
              You have not added any delivery slots. Add one below.
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
