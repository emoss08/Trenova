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
import { useUSStates } from "@/hooks/useQueries";
import { trailerSchema } from "@/lib/validations/EquipmentSchema";
import {
    equipmentStatusChoices,
    type TrailerFormValues as FormValues,
} from "@/types/equipment";
import { type TableSheetProps } from "@/types/tables";
import { yupResolver } from "@hookform/resolvers/yup";
import { useForm, type Control } from "react-hook-form";
import { AsyncSelectInput } from "./common/fields/async-select-input";
import { Form, FormControl, FormGroup } from "./ui/form";
import { Separator } from "./ui/separator";

export function TrailerForm({ control }: { control: Control<FormValues> }) {
  const {
    selectUSStates,
    isLoading: isUsStatesLoading,
    isError: isUSStatesError,
  } = useUSStates();

  return (
    <Form>
      <FormGroup>
        <FormControl>
          <SelectInput
            name="status"
            rules={{ required: true }}
            control={control}
            label="Status"
            options={equipmentStatusChoices}
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
            name="equipmentTypeId"
            rules={{ required: true }}
            control={control}
            link="/equipment-types/"
            valueKey="code"
            label="Equip. Type"
            placeholder="Select Equip. Type"
            description="Select the equipment type of the trailer, to categorize it based on its functionality and usage."
            hasPopoutWindow
            popoutLink="/equipment/equipment-types/"
            popoutLinkLabel="Equipment Type"
            noOptionsMessage={() => "Search for equipment types..."}
          />
        </FormControl>
        <FormControl>
          <AsyncSelectInput
            name="equipmentManufacturerId"
            rules={{ required: true }}
            control={control}
            link="/equipment-manufacturers/"
            valueKey="name"
            label="Equip. Manufacturer"
            placeholder="Select Manufacturer"
            description="Select the manufacturer of the trailer, to categorize it based on its functionality and usage."
            hasPopoutWindow
            popoutLink="/equipment/equipment-manufacturers/"
            popoutLinkLabel="Equipment Manufacturer"
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
            name="vin"
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
            name="fleetCodeId"
            control={control}
            link="/fleet-codes/"
            valueKey="code"
            label="Fleet Code"
            placeholder="Select Fleet Code"
            isClearable
            description="Select the code that identifies the trailer within your fleet."
            hasPopoutWindow
            popoutLink="/dispatch/fleet-codes/"
            popoutLinkLabel="Fleet Code"
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
            name="stateId"
            control={control}
            label="State"
            maxOptions={selectUSStates.length}
            options={selectUSStates}
            isFetchError={isUSStatesError}
            isLoading={isUsStatesLoading}
            placeholder="Select License Plate State"
            description="Select the state of registration of the trailer's license plate."
          />
        </FormControl>
        <FormControl>
          <DatepickerField
            name="lastInspectionDate"
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
            name="registrationStateId"
            control={control}
            label="Registration State"
            maxOptions={selectUSStates.length}
            options={selectUSStates}
            isFetchError={isUSStatesError}
            isLoading={isUsStatesLoading}
            placeholder="Select Registration State"
            description="Select the state where the trailer is registered."
          />
        </FormControl>
        <FormControl>
          <DatepickerField
            name="registrationExpirationDate"
            control={control}
            placeholder="Registration Expiration Date"
            label="Registration Expiration"
            description="Choose the date when the current registration of the trailer expires."
          />
        </FormControl>
      </FormGroup>
    </Form>
  );
}

export function TrailerDialog({ onOpenChange, open }: TableSheetProps) {
  const { control, reset, handleSubmit } = useForm<FormValues>({
    resolver: yupResolver(trailerSchema),
    defaultValues: {
      code: "",
      status: "Available",
      equipmentTypeId: undefined,
      equipmentManufacturerId: undefined,
      model: "",
      year: undefined,
      vin: "",
      fleetCodeId: undefined,
      licensePlateNumber: "",
      lastInspectionDate: undefined,
      stateId: undefined,
      registrationNumber: "",
      registrationStateId: undefined,
      registrationExpirationDate: undefined,
    },
  });

  const mutation = useCustomMutation<FormValues>(control, {
    method: "POST",
    path: "/trailers/",
    successMessage: "Trailer created successfully.",
    queryKeysToInvalidate: "trailers",
    closeModal: true,
    reset,
    errorMessage: "Failed to create new trailer.",
  });

  const onSubmit = (values: FormValues) => mutation.mutate(values);

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent className="w-full xl:w-1/2">
        <SheetHeader>
          <SheetTitle>Add New Trailer</SheetTitle>
          <SheetDescription>
            Use this form to add a new trailer to the system.
          </SheetDescription>
        </SheetHeader>
        <form
          onSubmit={handleSubmit(onSubmit)}
          className="flex h-full flex-col overflow-y-auto"
        >
          <TrailerForm control={control} />
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
              isLoading={mutation.isPending}
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
