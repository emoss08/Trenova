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
import { EquipmentTypeFormValues as FormValues } from "@/types/equipment";
import { TableSheetProps } from "@/types/tables";
import { yupResolver } from "@hookform/resolvers/yup";
import React from "react";
import { Control, useForm } from "react-hook-form";
import { CheckboxInput } from "./common/fields/checkbox";
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
            name="name"
            label="Name"
            autoCapitalize="none"
            autoCorrect="off"
            type="text"
            placeholder="Name"
            description="Name for the Equipment Type"
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
        <FormControl>
          <DecimalField
            control={control}
            name="idlingFuelUsage"
            label="Idling Fuel Usage"
            placeholder="Idling Fuel Usage"
            description="Idling Fuel Usage of the Equipment Type"
          />
        </FormControl>
        <FormControl>
          <CheckboxInput
            control={control}
            label="Exempt From Tolls"
            name="exemptFromTolls"
            description="Indicates if the equipment type is exempt from tolls"
          />
        </FormControl>
      </FormGroup>
      <div className="my-2">
        <TextareaField
          name="description"
          control={control}
          label="Description"
          placeholder="Description"
          description="Description of the Equipment Type"
        />
      </div>
    </Form>
  );
}

export function EquipTypeDialog({ onOpenChange, open }: TableSheetProps) {
  const [isSubmitting, setIsSubmitting] = React.useState<boolean>(false);
  const { control, reset, handleSubmit } = useForm<FormValues>({
    resolver: yupResolver(equipmentTypeSchema),
    defaultValues: {
      status: "A",
      name: "",
      description: "",
      costPerMile: "",
      equipmentClass: "UNDEFINED",
      exemptFromTolls: false,
      fixedCost: "",
      height: "",
      length: "",
      idlingFuelUsage: "",
      weight: "",
      variableCost: "",
      width: "",
    },
  });

  const mutation = useCustomMutation<FormValues>(
    control,
    {
      method: "POST",
      path: "/equipment_types/",
      successMessage: "Equipment Type created successfully.",
      queryKeysToInvalidate: ["equipment-type-table-data"],
      closeModal: true,
      errorMessage: "Failed to create new equip. type.",
    },
    () => setIsSubmitting(false),
    reset,
  );

  const onSubmit = (values: FormValues) => {
    setIsSubmitting(true);
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
            <Button type="submit" isLoading={isSubmitting} className="w-full">
              Save
            </Button>
          </SheetFooter>
        </form>
      </SheetContent>
    </Sheet>
  );
}
