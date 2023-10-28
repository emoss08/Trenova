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

import React from "react";
import { TableSheetProps } from "@/types/tables";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Control, useForm } from "react-hook-form";
import { ChargeTypeFormValues as FormValues } from "@/types/billing";
import { SelectInput } from "@/components/common/fields/select-input";
import { statusChoices } from "@/lib/choices";
import { chargeTypeSchema } from "@/lib/validations/BillingSchema";
import { yupResolver } from "@hookform/resolvers/yup";
import { InputField } from "@/components/common/fields/input";
import { TextareaField } from "@/components/common/fields/textarea";
import { Button } from "../ui/button";
import { toast } from "../ui/use-toast";
import { useCustomMutation } from "@/hooks/useCustomMutation";

export function ChargeTypeForm({ control }: { control: Control<FormValues> }) {
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
            description="Status of the Charge Type"
            isClearable={false}
          />
        </div>
        <div className="grid w-full items-center gap-0.5">
          <InputField
            control={control}
            rules={{ required: true }}
            name="name"
            label="Name"
            autoCapitalize="none"
            autoCorrect="off"
            type="text"
            placeholder="Name"
            autoComplete="name"
            description="Unique name for the Charge Type"
          />
        </div>
      </div>
      <div className="my-2">
        <TextareaField
          name="description"
          control={control}
          label="Description"
          placeholder="Description"
          description="Description of the Charge Type"
        />
      </div>
    </div>
  );
}

export function ChargeTypeDialog({ onOpenChange, open }: TableSheetProps) {
  const [isSubmitting, setIsSubmitting] = React.useState<boolean>(false);

  const { control, reset, handleSubmit } = useForm<FormValues>({
    resolver: yupResolver(chargeTypeSchema),
    defaultValues: {
      status: "A",
      name: "",
      description: "",
    },
  });

  const mutation = useCustomMutation<FormValues>(
    control,
    toast,
    {
      method: "POST",
      path: "/charge_types/",
      successMessage: "Charge Type created successfully.",
      queryKeysToInvalidate: ["charge-type-table-data"],
      closeModal: true,
      errorMessage: "Failed to create new charge type.",
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
          <DialogTitle>Create New Charge Type</DialogTitle>
        </DialogHeader>
        <DialogDescription>
          Please fill out the form below to create a new Charge Type.
        </DialogDescription>
        <form onSubmit={handleSubmit(onSubmit)}>
          <ChargeTypeForm control={control} />
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
