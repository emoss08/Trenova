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
  useWorkers,
} from "@/hooks/useQueries";
import { cleanObject, cn } from "@/lib/utils";
import { tractorSchema } from "@/lib/validations/EquipmentSchema";
import {
  TractorFormValues as FormValues,
  equipmentStatusChoices,
} from "@/types/equipment";
import { TableSheetProps } from "@/types/tables";
import { yupResolver } from "@hookform/resolvers/yup";
import { useState } from "react";
import { Control, useForm } from "react-hook-form";
import { DatepickerField } from "../common/fields/date-picker";
import { Form, FormControl, FormGroup } from "../ui/form";
import { Separator } from "../ui/separator";

export function TractorForm({
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

  const {
    selectWorkers,
    isError: isWorkerError,
    isLoading: isWorkersLoading,
  } = useWorkers(open);

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
            description="Select the current operational status of the tractor."
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
            description="Enter a unique identifier or code for the tractor."
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
            popoutLink="/equipment/equipment-types/"
            hasPopoutWindow
            placeholder="Select Equip. Type"
            description="Select the equipment type of the tractor, to categorize it based on its functionality and usage."
          />
        </FormControl>
        <FormControl>
          <AsyncSelectInput
            name="manufacturer"
            control={control}
            label="Manufacturer"
            popoutLink="/equipment/equipment-manufacturers/"
            hasPopoutWindow
            options={selectEquipManufacturers}
            isFetchError={isEquipManuError}
            isLoading={isEquipManuLoading}
            placeholder="Select Manufacturer"
            description="Select the manufacturer of the tractor, to categorize it based on its functionality and usage."
            isClearable
          />
        </FormControl>
        <FormControl>
          <InputField
            control={control}
            name="vinNumber"
            label="Vin Number"
            placeholder="Vin Number"
            autoCapitalize="none"
            autoCorrect="off"
            description="Input the Vehicle Identification Number."
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
            description="Indicate the model of the tractor as provided by the manufacturer."
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
            description="Enter the year of manufacture of the tractor."
            maxLength={4}
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
            description="Choose the state where the tractor is primarily operated or registered.."
            isClearable
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
            description="Select the code that identifies the tractor within your fleet."
            hasPopoutWindow
            popoutLink="/dispatch/fleet-codes/"
            isClearable
          />
        </FormControl>
        <FormControl>
          <AsyncSelectInput
            name="primaryWorker"
            control={control}
            label="Primary Worker"
            options={selectWorkers}
            isFetchError={isWorkerError}
            isLoading={isWorkersLoading}
            placeholder="Select Primary Worker"
            description="Select the primary worker assigned to the tractor."
            isClearable
          />
        </FormControl>
        <FormControl>
          <AsyncSelectInput
            name="secondaryWorker"
            control={control}
            label="Secondary Worker"
            options={selectWorkers}
            isFetchError={isWorkerError}
            isLoading={isWorkersLoading}
            placeholder="Select Secondary Worker"
            description="Select the secondary worker assigned to the tractor."
            isClearable
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
            description="Enter the license plate number of the tractor, crucial for legal identification and registration."
          />
        </FormControl>
        <FormControl>
          <DatepickerField
            name="leasedDate"
            control={control}
            label="Leased Date"
            placeholder="Leased Date"
            description="Input the date when the tractor was leased."
          />
        </FormControl>
        <FormControl className="mt-5">
          <CheckboxInput
            control={control}
            label="HOS Exempt?"
            name="hosExempt"
            description="Indicate whether the tractor is exempt from Hours of Service (HOS) regulations."
          />
        </FormControl>
        <FormControl className="mt-5">
          <CheckboxInput
            control={control}
            label="Owner Operated?"
            name="ownerOperated"
            description="Indicate whether the tractor is not owned by the company."
          />
        </FormControl>
        <FormControl className="mt-5">
          <CheckboxInput
            control={control}
            label="Leased?"
            name="leased"
            description="Indicate whether the tractor is leased."
          />
        </FormControl>
      </FormGroup>
    </Form>
  );
}

export function TractorDialog({ onOpenChange, open }: TableSheetProps) {
  const [isSubmitting, setIsSubmitting] = useState<boolean>(false);

  const { control, reset, handleSubmit } = useForm<FormValues>({
    resolver: yupResolver(tractorSchema),
    defaultValues: {
      status: "A",
      code: "",
      equipmentType: "",
      manufacturer: "",
      vinNumber: "",
      model: "",
      year: undefined,
      state: "",
      fleetCode: "",
      primaryWorker: "",
      secondaryWorker: "",
      licensePlateNumber: "",
      leasedDate: "",
      hosExempt: false,
      ownerOperated: false,
      leased: false,
    },
  });

  const mutation = useCustomMutation<FormValues>(
    control,
    {
      method: "POST",
      path: "/tractors/",
      successMessage: "Tractor created successfully.",
      queryKeysToInvalidate: ["tractor-table-data"],
      additionalInvalidateQueries: ["tractors"],
      closeModal: true,
      errorMessage: "Failed to create new tractor.",
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
          <SheetTitle>Add New Tractor</SheetTitle>
          <SheetDescription>
            Use this form to add a new tractor to the system.
          </SheetDescription>
        </SheetHeader>
        <form
          onSubmit={handleSubmit(onSubmit)}
          className="flex flex-col h-full overflow-y-auto"
        >
          <TractorForm control={control} open={open} />
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
