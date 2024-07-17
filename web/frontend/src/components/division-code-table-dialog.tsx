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
import { TextareaField } from "@/components/common/fields/textarea";
import { Button } from "@/components/ui/button";
import { useCustomMutation } from "@/hooks/useCustomMutation";
import { useGLAccounts } from "@/hooks/useQueries";
import { statusChoices } from "@/lib/choices";
import { divisionCodeSchema } from "@/lib/validations/AccountingSchema";
import { type TChoiceProps } from "@/types";
import { type DivisionCodeFormValues as FormValues } from "@/types/accounting";
import { type TableSheetProps } from "@/types/tables";
import { yupResolver } from "@hookform/resolvers/yup";
import { Control, useForm } from "react-hook-form";
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
    <Form>
      <FormGroup className="grid gap-6 md:grid-cols-1 lg:grid-cols-2">
        <FormControl>
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
            description="Code for the Division Code"
          />
        </FormControl>
      </FormGroup>
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
      <FormGroup className="grid gap-6 md:grid-cols-1 lg:grid-cols-2">
        <FormControl>
          <SelectInput
            name="cashAccountId"
            control={control}
            label="Cash Account"
            options={glAccounts}
            isLoading={isLoading}
            isFetchError={isError}
            placeholder="Select Cash Account"
            description="The Cash Account associated with the Division Code"
            isClearable
            hasPopoutWindow
            popoutLink="/accounting/gl-accounts"
            popoutLinkLabel="GL Account"
          />
        </FormControl>
        <FormControl>
          <SelectInput
            name="apAccountId"
            control={control}
            label="AP Account"
            options={glAccounts}
            isLoading={isLoading}
            isFetchError={isError}
            placeholder="Select AP Account"
            description="The AP Account associated with the Division Code"
            isClearable
            hasPopoutWindow
            popoutLink="/accounting/gl-accounts"
            popoutLinkLabel="GL Account"
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
      </FormGroup>
    </Form>
  );
}

export function DivisionCodeDialog({ onOpenChange, open }: TableSheetProps) {
  const { control, reset, handleSubmit } = useForm<FormValues>({
    resolver: yupResolver(divisionCodeSchema),
    defaultValues: {
      status: "Active",
      code: "",
      description: "",
      expenseAccountId: null,
      cashAccountId: null,
      apAccountId: null,
    },
  });

  const mutation = useCustomMutation<FormValues>(control, {
    method: "POST",
    path: "/division-codes/",
    successMessage: "Division Code created successfully.",
    queryKeysToInvalidate: "divisionCodes",
    closeModal: true,
    reset,
    errorMessage: "Failed to create new division code.",
  });

  const onSubmit = (values: FormValues) => {
    mutation.mutate(values);
  };

  const { selectGLAccounts, isLoading, isError } = useGLAccounts(open);

  return (
    <Credenza open={open} onOpenChange={onOpenChange}>
      <CredenzaContent>
        <CredenzaHeader>
          <CredenzaTitle>Create New Division Code</CredenzaTitle>
        </CredenzaHeader>
        <CredenzaDescription>
          Please fill out the form below to create a new Division Code.
        </CredenzaDescription>
        <CredenzaBody>
          <form onSubmit={handleSubmit(onSubmit)}>
            <DCForm
              control={control}
              glAccounts={selectGLAccounts}
              isLoading={isLoading}
              isError={isError}
            />
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
