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

import { Button } from "@/components/ui/button";
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
import { useCustomMutation } from "@/hooks/useCustomMutation";
import { useTags } from "@/hooks/useQueries";
import axios from "@/lib/axiosConfig";
import {
  accountClassificationChoices,
  accountSubTypeChoices,
  accountTypeChoices,
  cashFlowTypeChoices,
  statusChoices,
} from "@/lib/choices";
import { cn } from "@/lib/utils";
import { glAccountSchema } from "@/lib/validations/AccountingSchema";
import { type GLAccountFormValues } from "@/types/accounting";
import { type TableSheetProps } from "@/types/tables";
import { yupResolver } from "@hookform/resolvers/yup";
import { useQueryClient } from "@tanstack/react-query";
import React from "react";
import {
  Control,
  UseFormGetValues,
  UseFormSetValue,
  useForm,
} from "react-hook-form";
import { CheckboxInput } from "./common/fields/checkbox";
import { InputField } from "./common/fields/input";
import {
  CreatableSelectField,
  SelectInput,
} from "./common/fields/select-input";
import { TextareaField } from "./common/fields/textarea";
import { Form, FormControl, FormGroup } from "./ui/form";
import { Separator } from "./ui/separator";

export function GLForm({
  open,
  control,
  getValues,
  setValue,
}: {
  open: boolean;
  control: Control<GLAccountFormValues>;
  getValues: UseFormGetValues<GLAccountFormValues>;
  setValue: UseFormSetValue<GLAccountFormValues>;
}) {
  const [isLoading, setIsLoading] = React.useState<boolean>(false);
  const queryClient = useQueryClient();

  const {
    selectTags,
    isError: isTagsError,
    isLoading: isTagsLoading,
  } = useTags(open);

  const createNewTag = async (inputValue: string) => {
    setIsLoading(true);
    try {
      const res = await axios.post("/tags/", { name: inputValue });
      const newOption = { label: inputValue, value: res.data.id };

      // Appending the new tag to the form's current value
      const currentTags = getValues("tagIds");

      const tagsArray = Array.isArray(currentTags) ? currentTags : [];
      setValue("tagIds", [...tagsArray, newOption.value]);

      return newOption;
    } catch (err) {
      console.error("[Trenova] GL Table Sheet", err);
    } finally {
      await queryClient.invalidateQueries({
        queryKey: ["tags"],
      });
      setIsLoading(false);
    }
  };

  const isTagsUpdating = isTagsLoading || isLoading;

  return (
    <Form>
      <FormGroup>
        <FormControl>
          <SelectInput
            name="status"
            rules={{ required: true }}
            control={control}
            label="Status"
            options={statusChoices}
            placeholder="Select Status"
            description="Status of the General Ledger Account."
            isClearable={false}
          />
        </FormControl>
        <FormControl>
          <InputField
            control={control}
            rules={{ required: true }}
            name="accountNumber"
            label="Account Number"
            autoCapitalize="none"
            autoCorrect="off"
            type="text"
            placeholder="Account Number"
            autoComplete="accountNumber"
            description="Account Number for GL Account. ex: XXXX-XX"
            maxLength={7}
          />
        </FormControl>
      </FormGroup>
      <Separator />
      <FormGroup>
        <FormControl>
          <SelectInput
            name="accountType"
            control={control}
            rules={{ required: true }}
            label="Account Type"
            options={accountTypeChoices}
            placeholder="Select Account Type"
            description="The Account Type of GL Account"
          />
        </FormControl>
        <FormControl>
          <SelectInput
            control={control}
            name="cashFlowType"
            label="Cash Flow Type"
            options={cashFlowTypeChoices}
            isClearable
            placeholder="Select Cash Flow Type"
            description="The Cash Flow Type of the GL Account"
          />
        </FormControl>
        <FormControl>
          <SelectInput
            name="accountSubType"
            control={control}
            label="Account Sub Type"
            options={accountSubTypeChoices}
            isClearable
            placeholder="Select Account Sub Type"
            description="The Account Sub Type of the GL Account"
          />
        </FormControl>
        <FormControl>
          <SelectInput
            name="accountClassification"
            control={control}
            label="Account Classification"
            isClearable
            options={accountClassificationChoices}
            placeholder="Select Account Classification"
            description="The Account Classification of the GL Account"
          />
        </FormControl>
        <FormControl>
          <InputField
            control={control}
            name="interestRate"
            label="Interest Rate"
            autoCapitalize="none"
            autoCorrect="off"
            type="text"
            placeholder="Interest Rate"
            autoComplete="interestRate"
            description="Interest rate associated with the account"
          />
        </FormControl>
        <FormControl>
          <CreatableSelectField
            name="tagIds"
            control={control}
            description="Tags or labels associated with the account"
            label="Tags"
            onCreate={createNewTag}
            placeholder="Select Tags"
            options={selectTags}
            isClearable
            isLoading={isTagsUpdating}
            isFetchError={isTagsError}
            isMulti
          />
        </FormControl>
        <FormControl>
          <CheckboxInput
            control={control}
            label="Is Reconciled"
            name="isReconciled"
            description="Indicates if the account is reconciled"
          />
        </FormControl>
        <FormControl>
          <CheckboxInput
            control={control}
            label="Is Tax Relevant"
            name="isTaxRelevant"
            description="Indicates if the account is relevant for tax calculations"
          />
        </FormControl>
      </FormGroup>
      <TextareaField
        name="notes"
        control={control}
        label="Notes"
        placeholder="Notes"
        description="Additional notes or comments for the account"
      />
    </Form>
  );
}

export function GeneralLedgerAccountTableSheet({
  onOpenChange,
  open,
}: TableSheetProps) {
  const { handleSubmit, reset, control, getValues, setValue } =
    useForm<GLAccountFormValues>({
      resolver: yupResolver(glAccountSchema),
      defaultValues: {
        status: "Active",
        accountNumber: "",
        accountType: "Asset",
        cashFlowType: undefined,
        accountSubType: undefined,
        accountClassification: undefined,
        interestRate: undefined,
        isReconciled: false,
        isTaxRelevant: false,
        notes: "",
        tags: [],
      },
    });

  const mutation = useCustomMutation<GLAccountFormValues>(control, {
    method: "POST",
    path: "/general-ledger-accounts/",
    successMessage: "General Ledger Account created successfully.",
    queryKeysToInvalidate: "glAccounts",
    closeModal: true,
    reset,
    errorMessage: "Failed to create new general ledger account.",
  });

  const onSubmit = (values: GLAccountFormValues) => mutation.mutate(values);

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent className={cn("w-full xl:w-1/2")}>
        <SheetHeader>
          <SheetTitle>Add New GL Account</SheetTitle>
          <SheetDescription>
            Use this form to add a new general ledger account to the system.
          </SheetDescription>
        </SheetHeader>
        <form
          onSubmit={handleSubmit(onSubmit)}
          className="flex h-full flex-col overflow-y-auto"
        >
          <GLForm
            getValues={getValues}
            setValue={setValue}
            control={control}
            open={open}
          />
          <SheetFooter className="mb-12">
            <Button
              type="reset"
              variant="secondary"
              onClick={() => onOpenChange(false)}
              className="w-full"
            >
              Cancel
            </Button>
            <Button
              type="submit"
              isLoading={mutation.isPending}
              className="w-full"
            >
              Save
            </Button>
          </SheetFooter>
        </form>
      </SheetContent>
    </Sheet>
  );
}
