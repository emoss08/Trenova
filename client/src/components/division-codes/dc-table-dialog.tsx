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
import { useGLAccounts } from "@/hooks/useQueries";
import { statusChoices } from "@/lib/choices";
import { divisionCodeSchema } from "@/lib/validations/accounting";
import { TChoiceProps } from "@/types";
import { DivisionCodeFormValues as FormValues } from "@/types/accounting";
import { TableSheetProps } from "@/types/tables";
import { yupResolver } from "@hookform/resolvers/yup";
import React from "react";
import { Control, useForm } from "react-hook-form";

export function DCForm({
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
      <div className="grid md:grid-cols-1 lg:grid-cols-2 gap-2">
        <div className="grid w-full items-center gap-0.5">
          <SelectInput
            name="status"
            rules={{ required: true }}
            control={control}
            label="Status"
            options={statusChoices}
            placeholder="Select Status"
            description="Status of the Division Code"
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
            description="Code for the Division Code"
          />
        </div>
      </div>
      <div className="my-2">
        <TextareaField
          name="description"
          rules={{ required: true }}
          control={control}
          label="Description"
          placeholder="Description"
          description="Description of the Division Code"
        />
      </div>
      <div className="grid md:grid-cols-1 lg:grid-cols-2 gap-2">
        <div className="grid w-full max-w-sm items-center gap-0.5">
          <SelectInput
            name="cashAccount"
            control={control}
            label="Cash Account"
            options={glAccounts}
            maxOptions={10}
            isLoading={isLoading}
            isFetchError={isError}
            placeholder="Select Cash Account"
            description="The Cash Account associated with the Division Code"
            isClearable
          />
        </div>
        <div className="grid w-full max-w-sm items-center gap-0.5">
          <SelectInput
            name="apAccount"
            control={control}
            label="AP Account"
            options={glAccounts}
            maxOptions={10}
            isLoading={isLoading}
            isFetchError={isError}
            placeholder="Select AP Account"
            description="The AP Account associated with the Division Code"
            isClearable
          />
        </div>
        <div className="grid w-full max-w-sm items-center gap-0.5">
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
            isClearable
          />
        </div>
      </div>
    </div>
  );
}

export function DCDialog({ onOpenChange, open }: TableSheetProps) {
  const [isSubmitting, setIsSubmitting] = React.useState<boolean>(false);
  const { control, reset, handleSubmit } = useForm<FormValues>({
    resolver: yupResolver(divisionCodeSchema),
    defaultValues: {
      status: "A",
      code: "",
      description: "",
      expenseAccount: "",
      cashAccount: "",
      apAccount: "",
    },
  });

  const mutation = useCustomMutation<FormValues>(
    control,
    {
      method: "POST",
      path: "/division_codes/",
      successMessage: "Division Code created successfully.",
      queryKeysToInvalidate: ["division-code-table-data"],
      closeModal: true,
      errorMessage: "Failed to create new division code.",
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
          <DialogTitle>Create New Division Code</DialogTitle>
        </DialogHeader>
        <DialogDescription>
          Please fill out the form below to create a new Division Code.
        </DialogDescription>
        <form onSubmit={handleSubmit(onSubmit)}>
          <DCForm
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
