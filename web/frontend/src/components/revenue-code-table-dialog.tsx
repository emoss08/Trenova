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



import { InputField } from "@/components/common/fields/input";
import { SelectInput } from "@/components/common/fields/select-input";
import { Button } from "@/components/ui/button";
import { useCustomMutation } from "@/hooks/useCustomMutation";
import { statusChoices } from "@/lib/choices";
import { revenueCodeSchema } from "@/lib/validations/AccountingSchema";
import { type RevenueCodeFormValues as FormValues } from "@/types/accounting";
import { type TableSheetProps } from "@/types/tables";
import { yupResolver } from "@hookform/resolvers/yup";
import { Control, useForm } from "react-hook-form";
import { AsyncSelectInput } from "./common/fields/async-select-input";
import { GradientPicker } from "./common/fields/color-field";
import { TextareaField } from "./common/fields/textarea";
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
import { Form, FormControl, FormGroup } from "./ui/form";

export function RCForm({ control }: { control: Control<FormValues> }) {
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
          <AsyncSelectInput
            name="expenseAccountId"
            control={control}
            link="/general-ledger-accounts/"
            valueKey="accountNumber"
            label="Expense Account"
            placeholder="Select Expense Account"
            description="The Expense Account associated with the Revenue Code"
            isClearable
            hasPopoutWindow
            popoutLink="/accounting/gl-accounts"
            popoutLinkLabel="GL Account"
          />
        </FormControl>
        <FormControl>
          <AsyncSelectInput
            name="revenueAccountId"
            control={control}
            link="/general-ledger-accounts/"
            valueKey="accountNumber"
            label="Revenue Account"
            placeholder="Select Revenue Account"
            description="The Revneue Account associated with the Revenue Code"
            isClearable
            hasPopoutWindow
            popoutLink="/accounting/gl-accounts"
            popoutLinkLabel="GL Account"
          />
        </FormControl>
        <FormControl className="col-span-full">
          <GradientPicker
            name="color"
            label="Color"
            description="Color Code of the Revenue Code"
            control={control}
          />
        </FormControl>
      </FormGroup>
    </Form>
  );
}

export function RevenueCodeDialog({ onOpenChange, open }: TableSheetProps) {
  const { control, reset, handleSubmit } = useForm<FormValues>({
    resolver: yupResolver(revenueCodeSchema),
    defaultValues: {
      status: "Active",
      code: "",
      description: "",
      expenseAccountId: null,
      revenueAccountId: null,
    },
  });

  const mutation = useCustomMutation<FormValues>(control, {
    method: "POST",
    path: "/revenue-codes/",
    successMessage: "Revenue Code created successfully.",
    queryKeysToInvalidate: "revenueCodes",
    closeModal: true,
    reset,
    errorMessage: "Failed to create new revenue code.",
  });

  const onSubmit = (values: FormValues) => mutation.mutate(values);

  return (
    <Credenza open={open} onOpenChange={onOpenChange}>
      <CredenzaContent>
        <CredenzaHeader>
          <CredenzaTitle>Create New Revenue Code</CredenzaTitle>
        </CredenzaHeader>
        <CredenzaDescription>
          Please fill out the form below to create a new Revenue Code.
        </CredenzaDescription>
        <CredenzaBody>
          <form onSubmit={handleSubmit(onSubmit)}>
            <RCForm control={control} />
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
