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

import { statusChoices } from "@/lib/choices";
import { AccessorialChargeFormValues as FormValues } from "@/types/billing";
import { Control, useForm } from "react-hook-form";
import { fuelMethodChoices } from "@/utils/apps/billing";
import { TableSheetProps } from "@/types/tables";
import React from "react";
import { accessorialChargeSchema } from "@/lib/validations/BillingSchema";
import { yupResolver } from "@hookform/resolvers/yup";
import { useCustomMutation } from "@/hooks/useCustomMutation";
import { TextareaField } from "@/components/common/fields/textarea";
import { SelectInput } from "@/components/common/fields/select-input";
import { CheckboxInput } from "@/components/common/fields/checkbox";
import { InputField } from "@/components/common/fields/input";
import { toast } from "@/components/ui/use-toast";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";

export function ACForm({ control }: { control: Control<FormValues> }) {
  return (
    <div className="flex-1 overflow-y-visible">
      <div className="grid md:grid-cols-1 lg:grid-cols-2 gap-6">
        <div className="grid w-full max-w-sm items-center gap-0.5">
          <SelectInput
            name="status"
            rules={{ required: true }}
            control={control}
            label="Status"
            options={statusChoices}
            placeholder="Select Status"
            description="Status of the Accesorial Charge"
            isClearable={false}
          />
        </div>
        <div className="grid w-full items-center gap-0.5">
          <InputField
            control={control}
            rules={{ required: true }}
            name="code"
            label="Code"
            autoCapitalize="none"
            autoCorrect="off"
            type="text"
            placeholder="Code"
            description="Code for the Accesorial Charge"
          />
        </div>
      </div>
      <div className="my-2">
        <TextareaField
          name="description"
          control={control}
          label="Description"
          placeholder="Description"
          description="Description of the accessorial charge"
        />
      </div>
      <div className="grid md:grid-cols-1 lg:grid-cols-2 gap-6">
        <div className="grid w-full max-w-sm items-center gap-0.5">
          <SelectInput
            name="method"
            rules={{ required: true }}
            control={control}
            label="Method"
            options={fuelMethodChoices}
            placeholder="Select Fuel Method"
            description="Method for calculating the Accesorial Charge"
            isClearable={false}
          />
        </div>
        <div className="grid w-full items-center gap-0.5">
          <InputField
            control={control}
            rules={{ required: true }}
            name="chargeAmount"
            label="Charge Amount"
            autoCapitalize="none"
            autoCorrect="off"
            type="text"
            placeholder="Charge Amount"
            description="Charge amount for the Accesorial Charge"
          />
        </div>
        <div className="grid w-full max-w-sm items-center gap-0.5">
          <CheckboxInput
            control={control}
            label="Is Detention"
            name="isDetention"
            description="Is this a detention charge?"
          />
        </div>
      </div>
    </div>
  );
}

export function ACDialog({ onOpenChange, open }: TableSheetProps) {
  const [isSubmitting, setIsSubmitting] = React.useState<boolean>(false);

  const { control, reset, handleSubmit } = useForm<FormValues>({
    resolver: yupResolver(accessorialChargeSchema),
    defaultValues: {
      status: "A",
      code: "",
      description: "",
      isDetention: false,
      method: "D",
      chargeAmount: "",
    },
  });

  const mutation = useCustomMutation<FormValues>(
    control,
    toast,
    {
      method: "POST",
      path: "/accessorial_charges/",
      successMessage: "Accesorial Charge created successfully.",
      queryKeysToInvalidate: ["accessorial-charges-table-data"],
      closeModal: true,
      errorMessage: "Failed to create new accesorial charge.",
    },
    () => setIsSubmitting(false),
    reset,
  );

  const onSubmit = (values: FormValues) => {
    setIsSubmitting(true);
    mutation.mutate(values);
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Create New Accesorial Charge</DialogTitle>
        </DialogHeader>
        <DialogDescription>
          Please fill out the form below to create a new Accesorial Charge.
        </DialogDescription>
        <form onSubmit={handleSubmit(onSubmit)}>
          <ACForm control={control} />
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
