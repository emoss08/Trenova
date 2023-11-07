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
import { DelayCodeFormValues } from "@/types/dispatch";
import { SelectInput } from "@/components/common/fields/select-input";
import { statusChoices } from "@/lib/choices";
import { InputField } from "@/components/common/fields/input";
import { TextareaField } from "@/components/common/fields/textarea";
import React from "react";
import { TableSheetProps } from "@/types/tables";
import { Control, useForm } from "react-hook-form";
import { yupResolver } from "@hookform/resolvers/yup";
import { useCustomMutation } from "@/hooks/useCustomMutation";
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
import { delayCodeSchema } from "@/lib/validations/DispatchSchema";
import { CheckboxInput } from "@/components/common/fields/checkbox";

export function DelayCodeForm({
  control,
}: {
  control: Control<DelayCodeFormValues>;
}) {
  return (
    <div className="flex-1 overflow-y-visible">
      <div className="grid md:grid-cols-2 lg:grid-cols-2 gap-2">
        <div className="grid w-full max-w-sm items-center gap-0.5">
          <SelectInput
            name="status"
            rules={{ required: true }}
            control={control}
            label="Status"
            options={statusChoices}
            placeholder="Select Status"
            description="Status of the Delay code"
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
            description="Unique Code for the Delay Code"
            maxLength={4}
          />
        </div>
      </div>
      <div className="grid w-full items-center gap-0.5 my-5">
        <TextareaField
          name="description"
          rules={{ required: true }}
          control={control}
          label="Description"
          placeholder="Description"
          description="Description of the Delay Code"
        />
      </div>
      <div className="w-full max-w-sm items-center gap-0.5">
        <CheckboxInput
          control={control}
          label="Fault of Carrier or Driver?"
          name="fCarrierOrDriver"
          description="Indicates if the delay is the fault of the carrier or driver."
        />
      </div>
    </div>
  );
}

export function DelayCodeDialog({ onOpenChange, open }: TableSheetProps) {
  const [isSubmitting, setIsSubmitting] = React.useState<boolean>(false);

  const { control, reset, handleSubmit } = useForm<DelayCodeFormValues>({
    resolver: yupResolver(delayCodeSchema),
    defaultValues: {
      status: "A",
      code: "",
      description: "",
      fCarrierOrDriver: false,
    },
  });

  const mutation = useCustomMutation<DelayCodeFormValues>(
    control,
    toast,
    {
      method: "POST",
      path: "/delay_codes/",
      successMessage: "Delay Code created successfully.",
      queryKeysToInvalidate: ["delay-code-table-data"],
      closeModal: true,
      errorMessage: "Failed to create new delay code.",
    },
    () => setIsSubmitting(false),
    reset,
  );

  const onSubmit = (values: DelayCodeFormValues) => {
    setIsSubmitting(true);
    mutation.mutate(values);
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Create New Delay Code</DialogTitle>
        </DialogHeader>
        <DialogDescription>
          Please fill out the form below to create a new Delay Code.
        </DialogDescription>
        <form onSubmit={handleSubmit(onSubmit)}>
          <DelayCodeForm control={control} />
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
