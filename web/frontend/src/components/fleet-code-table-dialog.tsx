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
import { Form, FormControl, FormGroup } from "@/components/ui/form";
import { useCustomMutation } from "@/hooks/useCustomMutation";
import { useUsers } from "@/hooks/useQueries";
import { statusChoices } from "@/lib/choices";
import { cleanObject } from "@/lib/utils";
import { fleetCodeSchema } from "@/lib/validations/DispatchSchema";
import { type FleetCodeFormValues as FormValues } from "@/types/dispatch";
import { type TableSheetProps } from "@/types/tables";
import { yupResolver } from "@hookform/resolvers/yup";
import { Control, useForm } from "react-hook-form";
import { GradientPicker } from "./common/fields/color-field";
import {
  Credenza,
  CredenzaBody,
  CredenzaClose,
  CredenzaContent,
  CredenzaDescription,
  CredenzaFooter,
  CredenzaHeader,
  CredenzaTitle,
} from "./ui/credenza";

export function FleetCodeForm({
  control,
  open,
}: {
  control: Control<FormValues>;
  open: boolean;
}) {
  const { selectUsersData, isLoading, isError } = useUsers(open);

  return (
    <Form>
      <FormGroup className="grid gap-6 md:grid-cols-2 lg:grid-cols-2 xl:grid-cols-2">
        <FormControl>
          <SelectInput
            name="status"
            rules={{ required: true }}
            control={control}
            label="Status"
            options={statusChoices}
            placeholder="Select Status"
            description="Status of the Fleet Code"
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
            type="text"
            placeholder="Code"
            description="Code for the Fleet Code"
            maxLength={10}
          />
        </FormControl>
        <FormControl className="col-span-full">
          <TextareaField
            name="description"
            rules={{ required: true }}
            control={control}
            label="Description"
            placeholder="Description"
            description="Description of the Fleet Code"
          />
        </FormControl>
        <FormControl>
          <DecimalField
            control={control}
            name="revenueGoal"
            label="Revenue Goal"
            placeholder="Revenue Goal"
            description="Revenue Goal for the Fleet Code"
          />
        </FormControl>
        <FormControl>
          <DecimalField
            control={control}
            name="deadheadGoal"
            label="Deadhead Goal"
            placeholder="Deadhead Goal"
            description="Deadhead Goal for the Fleet Code"
          />
        </FormControl>
        <FormControl>
          <DecimalField
            control={control}
            name="mileageGoal"
            label="Mileage Goal"
            placeholder="Mileage Goal"
            description="Mileage Goal for the Fleet Code"
          />
        </FormControl>
        <FormControl>
          <SelectInput
            name="managerId"
            control={control}
            label="Manager"
            options={selectUsersData}
            isLoading={isLoading}
            isFetchError={isError}
            placeholder="Select Manager"
            description="User who manages the Fleet Code"
            isClearable
          />
        </FormControl>
        <FormControl className="col-span-full min-h-0">
          <GradientPicker
            name="color"
            label="Color"
            description="Color Code of the Fleet Code"
            control={control}
          />
        </FormControl>
      </FormGroup>
    </Form>
  );
}

export function FleetCodeDialog({ onOpenChange, open }: TableSheetProps) {
  const { control, reset, handleSubmit } = useForm<FormValues>({
    resolver: yupResolver(fleetCodeSchema),
    defaultValues: {
      status: "Active",
      code: "",
      description: "",
      revenueGoal: undefined,
      deadheadGoal: undefined,
      mileageGoal: undefined,
      color: "",
      managerId: "",
    },
  });

  const mutation = useCustomMutation<FormValues>(control, {
    method: "POST",
    path: "/fleet-codes/",
    successMessage: "Fleet Code created successfully.",
    queryKeysToInvalidate: "fleetCodes",
    closeModal: true,
    reset,
    errorMessage: "Failed to create new fleet code.",
  });

  const onSubmit = (values: FormValues) => {
    const cleanedValues = cleanObject(values);
    mutation.mutate(cleanedValues);
  };

  return (
    <Credenza open={open} onOpenChange={onOpenChange}>
      <CredenzaContent>
        <CredenzaHeader>
          <CredenzaTitle>Create New Fleet Code</CredenzaTitle>
        </CredenzaHeader>
        <CredenzaDescription>
          Please fill out the form below to create a new Fleet Code.
        </CredenzaDescription>
        <CredenzaBody>
          <form onSubmit={handleSubmit(onSubmit)}>
            <FleetCodeForm control={control} open={open} />
            <CredenzaFooter>
              <CredenzaClose asChild>
                <Button variant="outline" type="button">
                  Cancel
                </Button>
              </CredenzaClose>
              <Button type="submit" isLoading={mutation.isPending}>
                Save Changes
              </Button>
            </CredenzaFooter>
          </form>
        </CredenzaBody>
      </CredenzaContent>
    </Credenza>
  );
}
