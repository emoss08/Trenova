/**
 * Copyright (c) 2024 Trenova Technologies, LLC
 *
 * Licensed under the Business Source License 1.1 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     https://trenova.app/pricing/
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *
 * Key Terms:
 * - Non-production use only
 * - Change Date: 2026-11-16
 * - Change License: GNU General Public License v2 or later
 *
 * For full license text, see the LICENSE file in the root directory.
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
