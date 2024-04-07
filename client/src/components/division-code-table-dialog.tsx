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
import React from "react";
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
  const [isSubmitting, setIsSubmitting] = React.useState<boolean>(false);
  const { control, reset, handleSubmit } = useForm<FormValues>({
    resolver: yupResolver(divisionCodeSchema),
    defaultValues: {
      status: "A",
      code: "",
      description: "",
      expenseAccountId: "",
      cashAccountId: "",
      apAccountId: "",
    },
  });

  const mutation = useCustomMutation<FormValues>(
    control,
    {
      method: "POST",
      path: "/division-codes/",
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
              <Button type="submit" isLoading={isSubmitting}>
                Save Changes
              </Button>
            </CredenzaFooter>
          </form>
        </CredenzaBody>
      </CredenzaContent>
    </Credenza>
  );
}
