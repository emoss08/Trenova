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
import { SelectInput } from "@/components/common/fields/select-input";
import { statusChoices } from "@/lib/choices";
import { yupResolver } from "@hookform/resolvers/yup";
import { InputField } from "@/components/common/fields/input";
import { TextareaField } from "@/components/common/fields/textarea";
import { useCustomMutation } from "@/hooks/useCustomMutation";
import { Button } from "@/components/ui/button";
import { toast } from "@/components/ui/use-toast";
import { serviceTypeSchema } from "@/lib/validations/ShipmentSchema";
import { ServiceTypeFormValues as FormValues } from "@/types/order";

export function ServiceTypeForm({ control }: { control: Control<FormValues> }) {
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
            description="Status of the Service Type"
            isClearable={false}
          />
        </div>
        <div className="grid w-full items-center gap-0.5">
          <InputField
            control={control}
            rules={{ required: true }}
            name="code"
            label="Code"
            maxLength={10}
            autoCapitalize="none"
            autoCorrect="off"
            type="text"
            placeholder="Code"
            autoComplete="code"
            description="Code for the Service Type"
          />
        </div>
      </div>
      <div className="my-2">
        <TextareaField
          name="description"
          control={control}
          label="Description"
          placeholder="Description"
          description="Description of the Service Type"
        />
      </div>
    </div>
  );
}

export function ServiceTypeDialog({ onOpenChange, open }: TableSheetProps) {
  const [isSubmitting, setIsSubmitting] = React.useState<boolean>(false);

  const { control, reset, handleSubmit } = useForm<FormValues>({
    resolver: yupResolver(serviceTypeSchema),
    defaultValues: {
      status: "A",
      code: "",
      description: "",
    },
  });

  const mutation = useCustomMutation<FormValues>(
    control,
    toast,
    {
      method: "POST",
      path: "/service_types/",
      successMessage: "Service Type created successfully.",
      queryKeysToInvalidate: ["service-type-table-data"],
      closeModal: true,
      errorMessage: "Failed to create new service type.",
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
          <DialogTitle>Create New Service Type</DialogTitle>
        </DialogHeader>
        <DialogDescription>
          Please fill out the form below to create a new Service Type.
        </DialogDescription>
        <form onSubmit={handleSubmit(onSubmit)}>
          <ServiceTypeForm control={control} />
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
