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

import { DecimalField } from "@/components/common/fields/decimal-input";
import { InputField } from "@/components/common/fields/input";
import { SelectInput } from "@/components/common/fields/select-input";
import { TextareaField } from "@/components/common/fields/textarea";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { useCustomMutation } from "@/hooks/useCustomMutation";
import { useUsers } from "@/hooks/useQueries";
import { statusChoices } from "@/lib/choices";
import { fleetCodeSchema } from "@/lib/validations/DispatchSchema";
import { FleetCodeFormValues as FormValues } from "@/types/dispatch";
import { TableSheetProps } from "@/types/tables";
import { yupResolver } from "@hookform/resolvers/yup";
import React from "react";
import { Control, useForm } from "react-hook-form";
import { Form, FormControl, FormGroup } from "@/components/ui/form";
import { cleanObject } from "@/lib/utils";

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
      <FormGroup className="md:grid-cols-1 lg:grid-cols-2">
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
      </FormGroup>
      <div>
        <TextareaField
          name="description"
          rules={{ required: true }}
          control={control}
          label="Description"
          placeholder="Description"
          description="Description of the Fleet Code"
        />
      </div>
      <FormGroup className="md:grid-cols-1 lg:grid-cols-2">
        <FormControl>
          <DecimalField
            control={control}
            name="revenueGoal"
            label="Revenue Goal"
            type="text"
            placeholder="Revenue Goal"
            description="Revenue Goal for the Fleet Code"
          />
        </FormControl>
        <FormControl className="grid w-full max-w-sm items-center gap-0.5">
          <DecimalField
            control={control}
            name="deadheadGoal"
            label="Deadhead Goal"
            type="text"
            placeholder="Deadhead Goal"
            description="Deadhead Goal for the Fleet Code"
          />
        </FormControl>
        <FormControl className="grid w-full max-w-sm items-center gap-0.5">
          <DecimalField
            control={control}
            name="mileageGoal"
            label="Mileage Goal"
            type="text"
            placeholder="Mileage Goal"
            description="Mileage Goal for the Fleet Code"
          />
        </FormControl>
        <FormControl className="grid w-full max-w-sm items-center gap-0.5">
          <SelectInput
            name="manager"
            control={control}
            label="Manager"
            options={selectUsersData}
            maxOptions={10}
            isLoading={isLoading}
            isFetchError={isError}
            placeholder="Select Manager"
            description="User who manages the Fleet Code"
            isClearable
            hasPopoutWindow
            popoutLink="#" // TODO: Change once Document Classification is added.
            popoutLinkLabel="User"
          />
        </FormControl>
      </FormGroup>
    </Form>
  );
}

export function FleetCodeDialog({ onOpenChange, open }: TableSheetProps) {
  const [isSubmitting, setIsSubmitting] = React.useState<boolean>(false);
  const { control, reset, handleSubmit } = useForm<FormValues>({
    resolver: yupResolver(fleetCodeSchema),
    defaultValues: {
      status: "A",
      code: "",
      description: "",
      revenueGoal: "",
      deadheadGoal: "",
      mileageGoal: "",
      manager: "",
    },
  });

  const mutation = useCustomMutation<FormValues>(
    control,
    {
      method: "POST",
      path: "/fleet_codes/",
      successMessage: "Fleet Code created successfully.",
      queryKeysToInvalidate: ["fleet-code-table-data"],
      closeModal: true,
      errorMessage: "Failed to create new fleet code.",
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
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Create New Fleet Code</DialogTitle>
        </DialogHeader>
        <DialogDescription>
          Please fill out the form below to create a new Fleet Code.
        </DialogDescription>
        <form onSubmit={handleSubmit(onSubmit)}>
          <FleetCodeForm control={control} open={open} />
          <DialogFooter className="mt-6">
            <Button
              type="submit"
              isLoading={isSubmitting}
              loadingText="Saving Changes..."
            >
              Save
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
