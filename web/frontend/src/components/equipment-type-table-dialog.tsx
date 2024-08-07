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
import { TextareaField } from "@/components/common/fields/textarea";
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
import { equipmentClassChoices, statusChoices } from "@/lib/choices";
import { cn } from "@/lib/utils";
import { equipmentTypeSchema } from "@/lib/validations/EquipmentSchema";
import { type EquipmentTypeFormValues as FormValues } from "@/types/equipment";
import { type TableSheetProps } from "@/types/tables";
import { yupResolver } from "@hookform/resolvers/yup";
import { Control, useForm } from "react-hook-form";
import { CheckboxInput } from "./common/fields/checkbox";
import { GradientPicker } from "./common/fields/color-field";
import { Form, FormControl, FormGroup } from "./ui/form";

export function EquipTypeForm({ control }: { control: Control<FormValues> }) {
  return (
    <Form>
      <FormGroup>
        <FormControl>
          <SelectInput
            name="status"
            rules={{ required: true }}
            control={control}
            label="Status"
            options={statusChoices}
            placeholder="Select Status"
            description="Status of the Equipment Type"
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
            description="Code for the Equipment Type"
          />
        </FormControl>
      </FormGroup>
      <FormGroup>
        <FormControl>
          <SelectInput
            name="equipmentClass"
            rules={{ required: true }}
            control={control}
            label="Equipment Class"
            options={equipmentClassChoices}
            placeholder="Select Equipment Class"
            description="Class of Equipment Type"
            isClearable={false}
          />
        </FormControl>
        <FormControl>
          <DecimalField
            control={control}
            name="costPerMile"
            label="Cost Per Mile"
            type="text"
            placeholder="Cost Per Mile"
            description="Cost Per Mile for the Equipment Type"
          />
        </FormControl>
        <FormControl>
          <DecimalField
            control={control}
            name="fixedCost"
            label="Fixed Cost"
            type="text"
            placeholder="Fixed Cost"
            description="Fixed Cost of the Equipment Type"
          />
        </FormControl>
        <FormControl>
          <DecimalField
            control={control}
            name="variableCost"
            label="Variable Cost"
            placeholder="Variable Cost"
            description="Variable Cost of the Equipment Type"
          />
        </FormControl>
        <FormControl>
          <DecimalField
            control={control}
            name="height"
            label="Height"
            placeholder="Height"
            description="Height of the Equipment Type"
          />
        </FormControl>
        <FormControl>
          <DecimalField
            control={control}
            name="length"
            label="Length"
            placeholder="Length"
            description="Length of the Equipment Type"
          />
        </FormControl>
        <FormControl>
          <DecimalField
            control={control}
            name="width"
            label="Width"
            placeholder="Width"
            description="Width of the Equipment Type"
          />
        </FormControl>
        <FormControl>
          <DecimalField
            control={control}
            name="weight"
            label="Weight"
            placeholder="Weight"
            description="Weight of the Equipment Type"
          />
        </FormControl>
        <FormControl className="mt-3">
          <CheckboxInput
            control={control}
            label="Exempt From Tolls"
            name="exemptFromTolls"
            description="Indicates if the equipment type is exempt from tolls"
          />
        </FormControl>
        <FormControl className="col-span-full">
          <TextareaField
            name="description"
            control={control}
            label="Description"
            placeholder="Description"
            description="Description of the Equipment Type"
          />
        </FormControl>
        <FormControl className="col-span-full">
          <GradientPicker
            name="color"
            label="Color"
            description="Color Code of the Equipment Type"
            control={control}
          />
        </FormControl>
      </FormGroup>
    </Form>
  );
}

export function EquipTypeDialog({ onOpenChange, open }: TableSheetProps) {
  const { control, reset, handleSubmit } = useForm<FormValues>({
    resolver: yupResolver(equipmentTypeSchema),
    defaultValues: {
      status: "Active",
      code: "",
      description: "",
      costPerMile: undefined,
      equipmentClass: "Undefined",
      exemptFromTolls: false,
      fixedCost: undefined,
      height: undefined,
      length: undefined,
      idlingFuelUsage: undefined,
      weight: undefined,
      variableCost: undefined,
      width: undefined,
      color: undefined,
    },
  });

  const mutation = useCustomMutation<FormValues>(control, {
    method: "POST",
    path: "/equipment-types/",
    successMessage: "Equipment Type created successfully.",
    queryKeysToInvalidate: "equipmentTypes",
    closeModal: true,
    reset,
    errorMessage: "Failed to create new equip. type.",
  });

  const onSubmit = (values: FormValues) => {
    mutation.mutate(values);
  };

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent className={cn("w-full xl:w-1/2")}>
        <SheetHeader>
          <SheetTitle>Add New Equipment Type</SheetTitle>
          <SheetDescription>
            Use this form to add a new equipment type to the system.
          </SheetDescription>
        </SheetHeader>
        <form
          onSubmit={handleSubmit(onSubmit)}
          className="flex h-full flex-col overflow-y-auto"
        >
          <EquipTypeForm control={control} />
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
