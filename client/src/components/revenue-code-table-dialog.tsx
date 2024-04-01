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

import { InputField } from "@/components/common/fields/input";
import { SelectInput } from "@/components/common/fields/select-input";
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
import { revenueCodeSchema } from "@/lib/validations/AccountingSchema";
import { TChoiceProps } from "@/types";
import { type RevenueCodeFormValues as FormValues } from "@/types/accounting";
import { type TableSheetProps } from "@/types/tables";
import { yupResolver } from "@hookform/resolvers/yup";
import React from "react";
import { Control, useForm } from "react-hook-form";
import { TextareaField } from "./common/fields/textarea";
import { Form, FormControl, FormGroup } from "./ui/form";

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
            description="Operational status of the Revenue Code"
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
            autoCorrect="off"
            type="text"
            placeholder="Code"
            description="Code for the Revenue Code"
          />
        </FormControl>
        <FormControl className="col-span-full">
          <TextareaField
            name="description"
            rules={{ required: true }}
            control={control}
            label="Description"
            placeholder="Description"
            description="Description of the Revenue Code"
          />
        </FormControl>
        <FormControl>
          <SelectInput
            name="expenseAccountId"
            control={control}
            label="Expense Account"
            options={glAccounts}
            isLoading={isLoading}
            isFetchError={isError}
            placeholder="Select Expense Account"
            description="The Expense Account associated with the Revenue Code"
            isClearable
            hasPopoutWindow
            popoutLink="/accounting/gl-accounts"
            popoutLinkLabel="GL Account"
          />
        </FormControl>
        <FormControl>
          <SelectInput
            name="revenueAccountId"
            control={control}
            label="Revenue Account"
            options={glAccounts}
            maxOptions={10}
            isLoading={isLoading}
            isFetchError={isError}
            placeholder="Select Revenue Account"
            description="The Revneue Account associated with the Revenue Code"
            isClearable
            hasPopoutWindow
            popoutLink="/accounting/gl-accounts"
            popoutLinkLabel="GL Account"
          />
        </FormControl>
      </FormGroup>
    </Form>
  );
}

export function RevenueCodeDialog({ onOpenChange, open }: TableSheetProps) {
  const [isSubmitting, setIsSubmitting] = React.useState<boolean>(false);

  const { control, reset, handleSubmit } = useForm<FormValues>({
    resolver: yupResolver(revenueCodeSchema),
    defaultValues: {
      status: "A",
      code: "",
      description: "",
      expenseAccountId: null,
      revenueAccountId: null,
    },
  });

  const mutation = useCustomMutation<FormValues>(
    control,
    {
      method: "POST",
      path: "/revenue-codes/",
      successMessage: "Revenue Code created successfully.",
      queryKeysToInvalidate: ["revenue-code-table-data"],
      additionalInvalidateQueries: ["revenueCodes"],
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
            <Button type="submit" isLoading={isSubmitting}>
              Save
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
