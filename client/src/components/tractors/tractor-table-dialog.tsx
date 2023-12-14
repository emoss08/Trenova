/*
 * COPYRIGHT(c) 2023 MONTA
 *
 * This file is part of Monta.
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
import { AsyncSelectInput } from "@/components/common/fields/async-select-input";
import { CheckboxInput } from "@/components/common/fields/checkbox";
import { DatepickerField } from "@/components/common/fields/date-picker";
import { InputField } from "@/components/common/fields/input";
import { SelectInput } from "@/components/common/fields/select-input";
import { Button } from "@/components/ui/button";
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
import { useCustomMutation } from "@/hooks/useCustomMutation";
import {
  useEquipManufacturers,
  useEquipmentTypes,
  useFleetCodes,
  useUSStates,
} from "@/hooks/useQueries";
import { cleanObject, cn } from "@/lib/utils";
import { trailerSchema } from "@/lib/validations/EquipmentSchema";
import {
  TrailerFormValues as FormValues,
  trailerStatusChoices,
} from "@/types/equipment";
import { TableSheetProps } from "@/types/tables";
import { yupResolver } from "@hookform/resolvers/yup";
import { useState } from "react";
import { Control, useForm } from "react-hook-form";
import { Form, FormControl, FormGroup } from "../ui/form";
import { Separator } from "../ui/separator";

export function TrailerForm({
  control,
  open,
}: {
  control: Control<FormValues>;
  open: boolean;
}) {
  const { selectEquipmentType, isLoading, isError } = useEquipmentTypes(open);

  const {
    selectEquipManufacturers,
    isLoading: isEquipManuLoading,
    isError: isEquipManuError,
  } = useEquipManufacturers(open);

  const {
    selectFleetCodes,
    isError: isFleetCodeError,
    isLoading: isFleetCodesLoading,
  } = useFleetCodes(open);

  const {
    selectUSStates,
    isError: isStateError,
    isLoading: isStatesLoading,
  } = useUSStates(open);

  return (
    <Form>
      <FormGroup>
        <FormControl>
          <SelectInput
            name="status"
            rules={{ required: true }}
            control={control}
            label="Status"
            options={trailerStatusChoices}
            placeholder="Select Status"
            description="Select the current operational status of the trailer."
            isClearable={false}
          />
        </FormControl>
        <FormControl>
          <InputField
            control={control}
            rules={{ required: true }}
            name="code"
            label="Code"
            autoCapitalize="none"
            autoCorrect="off"
            type="text"
            placeholder="Code"
            description="Enter a unique identifier or code for the trailer."
          />
        </FormControl>
      </FormGroup>
      <Separator />
      <FormGroup>
        <FormControl>
          <AsyncSelectInput
            name="equipmentType"
            rules={{ required: true }}
            control={control}
            label="Equip. Type"
            options={selectEquipmentType}
            isFetchError={isError}
            isLoading={isLoading}
            placeholder="Select Equip. Type"
            description="Select the equipment type of the trailer, to categorize it based on its functionality and usage."
            isClearable={false}
          />
        </FormControl>
        <FormControl>
          <AsyncSelectInput
            name="manufacturer"
            control={control}
            label="Manufacturer"
            options={selectEquipManufacturers}
            isFetchError={isEquipManuError}
            isLoading={isEquipManuLoading}
            placeholder="Select Manufacturer"
            description="Select the manufacturer of the trailer, to categorize it based on its functionality and usage."
            isClearable={false}
          />
        </FormControl>
        <FormControl>
          <InputField
            control={control}
            name="make"
            label="Make"
            placeholder="Make"
            autoCapitalize="none"
            autoCorrect="off"
            description="Specify the manufacturer of the trailer."
          />
        </FormControl>
        <FormControl>
          <InputField
            name="model"
            control={control}
            label="Model"
            placeholder="Model"
            autoCapitalize="none"
            autoCorrect="off"
            description="Indicate the model of the trailer as provided by the manufacturer."
          />
        </FormControl>
        <FormControl>
          <InputField
            type="number"
            name="year"
            control={control}
            label="Year"
            placeholder="Year"
            autoCapitalize="none"
            autoCorrect="off"
            description="Enter the year of manufacture of the trailer."
            maxLength={4}
          />
        </FormControl>
        <FormControl>
          <InputField
            name="vinNumber"
            control={control}
            label="Vin Number"
            placeholder="Vin Number"
            autoCapitalize="none"
            autoCorrect="off"
            description="Input the Vehicle Identification Number."
          />
        </FormControl>
        <FormControl>
          <AsyncSelectInput
            name="fleetCode"
            control={control}
            label="Fleet Code"
            options={selectFleetCodes}
            isFetchError={isFleetCodeError}
            isLoading={isFleetCodesLoading}
            placeholder="Select Fleet Code"
            description="Select the code that identifies the trailer within your fleet."
          />
        </FormControl>
        <FormControl>
          <SelectInput
            name="state"
            control={control}
            label="State"
            options={selectUSStates}
            isFetchError={isStateError}
            isLoading={isStatesLoading}
            placeholder="Select State"
            description="Choose the state where the trailer is primarily operated or registered.."
            isClearable={false}
          />
        </FormControl>
        <FormControl>
          <InputField
            name="licensePlateNumber"
            control={control}
            label="License Plate #"
            placeholder="License Plate Number"
            autoCapitalize="none"
            autoCorrect="off"
            description="Enter the license plate number of the trailer, crucial for legal identification and registration."
          />
        </FormControl>
        <FormControl>
          <SelectInput
            name="licensePlateState"
            control={control}
            label="License Plate State"
            options={selectUSStates}
            isFetchError={isStateError}
            isLoading={isStatesLoading}
            placeholder="Select License Plate State"
            description="Select the state of registration of the trailer's license plate."
          />
        </FormControl>
        <FormControl>
          <DatepickerField
            name="lastInspection"
            control={control}
            label="Last Inspection"
            placeholder="Last Inspection Date"
            description="Input the date of the last inspection the trailer underwent."
          />
        </FormControl>
        <FormControl>
          <InputField
            name="registrationNumber"
            control={control}
            label="Registration #"
            placeholder="Registration Number"
            autoCapitalize="none"
            autoCorrect="off"
            description="Enter the registration number assigned to the trailer by the motor vehicle department."
          />
        </FormControl>
        <FormControl>
          <SelectInput
            name="registrationState"
            control={control}
            label="Registration State"
            options={selectUSStates}
            isFetchError={isStateError}
            isLoading={isStatesLoading}
            placeholder="Select Registration State"
            description="Select the state where the trailer is registered."
          />
        </FormControl>
        <FormControl>
          <DatepickerField
            name="registrationExpiration"
            control={control}
            placeholder="Registration Expiration Date"
            label="Registration Expiration"
            description="Choose the date when the current registration of the trailer expires."
          />
        </FormControl>
        <FormControl className="mt-5">
          <CheckboxInput
            control={control}
            label="Is Leased?"
            name="isLeased"
            description="Indicate whether the trailer is leased."
          />
        </FormControl>
      </FormGroup>
    </Form>
  );
}

export function TrailerDialog({ onOpenChange, open }: TableSheetProps) {
  const [isSubmitting, setIsSubmitting] = useState<boolean>(false);

  const { control, reset, handleSubmit } = useForm<FormValues>({
    resolver: yupResolver(trailerSchema),
    defaultValues: {
      code: "",
      status: "A",
      equipmentType: "",
      manufacturer: "",
      make: "",
      model: "",
      year: undefined,
      vinNumber: "",
      fleetCode: "",
      licensePlateNumber: "",
      lastInspection: undefined,
      state: "",
      isLeased: false,
      registrationNumber: "",
      registrationState: "",
      registrationExpiration: "",
    },
  });

  const mutation = useCustomMutation<FormValues>(
    control,
    {
      method: "POST",
      path: "/trailers/",
      successMessage: "Trailer created successfully.",
      queryKeysToInvalidate: ["trailer-table-data"],
      closeModal: true,
      errorMessage: "Failed to create new trailer.",
    },
    () => setIsSubmitting(false),
    reset,
  );

  const onSubmit = (values: FormValues) => {
    const cleanedValues = cleanObject(values);

    setIsSubmitting(true);
    mutation.mutate(cleanedValues);
  };

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent className={cn("w-full xl:w-1/2")}>
        <SheetHeader>
          <SheetTitle>Add New Trailer</SheetTitle>
          <SheetDescription>
            Use this form to add a new trailer to the system.
          </SheetDescription>
        </SheetHeader>
        <form
          onSubmit={handleSubmit(onSubmit)}
          className="flex flex-col h-full overflow-y-auto"
        >
          <TrailerForm control={control} open={open} />
          <SheetFooter className="mb-12">
            <Button
              type="reset"
              variant="secondary"
              onClick={() => onOpenChange(false)}
              className="w-full"
            >
              Cancel
            </Button>
            <Button
              type="submit"
              isLoading={isSubmitting}
              loadingText="Saving Changes..."
              className="w-full"
            >
              Save
            </Button>
          </SheetFooter>
        </form>
      </SheetContent>
    </Sheet>
  );
}
