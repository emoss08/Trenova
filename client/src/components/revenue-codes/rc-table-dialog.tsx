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

import { Control, useForm } from "react-hook-form";
import { RevenueCodeFormValues as FormValues } from "@/types/accounting";
import { SelectInput } from "@/components/common/fields/select-input";
import { InputField } from "@/components/common/fields/input";
import { TextareaField } from "@/components/common/fields/textarea";
import React from "react";
import { TChoiceProps } from "@/types";
import { TableSheetProps } from "@/types/tables";
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
import { revenueCodeSchema } from "@/lib/validations/accounting";
import { useGLAccounts } from "@/hooks/useQueries";

export function RCForm({
  control,
  glAccounts,
  isLoading,
  isError,
}: {
  control: Control<FormValues>;
  glAccounts: TChoiceProps[];
  isLoading: boolean;
  isError: boolean;
}) {
  return (
    <div className="flex-1 overflow-y-visible">
      <div className="grid md:grid-cols-1 lg:grid-cols-1 gap-2">
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
            description="Code for the Revenue Code"
          />
        </div>
        <div className="grid w-full items-center gap-0.5 mb-2">
          <TextareaField
            name="description"
            rules={{ required: true }}
            control={control}
            label="Description"
            placeholder="Description"
            description="Description of the Revenue Code"
          />
        </div>
      </div>
      <div className="grid md:grid-cols-1 lg:grid-cols-2 gap-6">
        <div className="grid w-full items-center gap-0.5">
          <SelectInput
            name="expenseAccount"
            control={control}
            label="Expense Account"
            options={glAccounts}
            maxOptions={10}
            isLoading={isLoading}
            isFetchError={isError}
            placeholder="Select Expense Account"
            description="The Expense Account associated with the Revenue Code"
            isClearable={false}
          />
        </div>
        <div className="grid w-full items-center gap-0.5">
          <SelectInput
            name="revenueAccount"
            control={control}
            label="Revenue Account"
            options={glAccounts}
            maxOptions={10}
            isLoading={isLoading}
            isFetchError={isError}
            placeholder="Select Revenue Account"
            description="The Revneue Account associated with the Revenue Code"
            isClearable={false}
          />
        </div>
      </div>
    </div>
  );
}

export function RCDialog({ onOpenChange, open }: TableSheetProps) {
  const [isSubmitting, setIsSubmitting] = React.useState<boolean>(false);

  const { control, reset, handleSubmit } = useForm<FormValues>({
    resolver: yupResolver(revenueCodeSchema),
    defaultValues: {
      code: "",
      description: "",
      expenseAccount: "",
      revenueAccount: "",
    },
  });

  const mutation = useCustomMutation<FormValues>(
    control,
    toast,
    {
      method: "POST",
      path: "/revenue_codes/",
      successMessage: "Revenue Code created successfully.",
      queryKeysToInvalidate: ["revenue-code-table-data"],
      closeModal: true,
      errorMessage: "Failed to create new revenue code.",
    },
    () => setIsSubmitting(false),
    reset,
  );

  const onSubmit = (values: FormValues) => {
    setIsSubmitting(true);
    mutation.mutate(values);
  };

  const { selectGLAccounts, isLoading, isError } = useGLAccounts(open);

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Create New Revenue Code</DialogTitle>
        </DialogHeader>
        <DialogDescription>
          Please fill out the form below to create a new Revenue Code.
        </DialogDescription>
        <form onSubmit={handleSubmit(onSubmit)}>
          <RCForm
            control={control}
            glAccounts={selectGLAccounts}
            isLoading={isLoading}
            isError={isError}
          />
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
