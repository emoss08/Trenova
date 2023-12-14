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
import { useGLAccounts, useTags, useUsers } from "@/hooks/useQueries";
import axios from "@/lib/axiosConfig";
import {
  accountClassificationChoices,
  accountSubTypeChoices,
  accountTypeChoices,
  cashFlowTypeChoices,
  statusChoices,
} from "@/lib/choices";
import { cleanObject, cn } from "@/lib/utils";
import { glAccountSchema } from "@/lib/validations/accounting";
import { GLAccountFormValues } from "@/types/accounting";
import { TableSheetProps } from "@/types/tables";
import { yupResolver } from "@hookform/resolvers/yup";
import { useQueryClient } from "@tanstack/react-query";
import React from "react";
import {
  Control,
  UseFormGetValues,
  UseFormSetValue,
  useForm,
} from "react-hook-form";
import { CheckboxInput } from "../common/fields/checkbox";
import { FileField, InputField } from "../common/fields/input";
import {
  CreatableSelectField,
  SelectInput,
} from "../common/fields/select-input";
import { TextareaField } from "../common/fields/textarea";
import { Form, FormControl, FormGroup } from "../ui/form";
import { Separator } from "../ui/separator";

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
    selectGLAccounts,
    isError: isGLAccountsError,
    isLoading: isGLAccountsLoading,
  } = useGLAccounts(open);

  const {
    selectUsersData,
    isError: isUsersError,
    isLoading: isUsersLoading,
  } = useUsers(open);

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
      const currentTags = getValues("tags");
      const tagsArray = Array.isArray(currentTags) ? currentTags : [];
      setValue("tags", [...tagsArray, newOption.value]);

      setTagOptions((prev) => [...prev, newOption]);

      return newOption;
    } catch (err) {
      console.log(err);
    } finally {
      await queryClient.invalidateQueries({
        queryKey: ["tags"],
      });
      setIsLoading(false);
    }
  };

  const [tagOptions, setTagOptions] = React.useState(selectTags);
  const isTagsUpdating = isTagsLoading || isLoading;

  React.useEffect(() => {
    setTagOptions(selectTags);
  }, [selectTags]);

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
            placeholder="Select Account Sub Type"
            description="The Account Sub Type of the GL Account"
          />
        </FormControl>
        <FormControl>
          <SelectInput
            name="accountClassification"
            control={control}
            label="Account Classification"
            options={accountClassificationChoices}
            placeholder="Select Account Classification"
            description="The Account Classification of the GL Account"
          />
        </FormControl>
        <FormControl>
          <SelectInput
            name="parentAccount"
            control={control}
            label="Parent Account"
            options={selectGLAccounts}
            isLoading={isGLAccountsLoading}
            isFetchError={isGLAccountsError}
            placeholder="Select Parent Account"
            description="Parent account for hierarchical accounting"
          />
        </FormControl>
        <FormControl>
          <FileField
            name="attachment"
            control={control}
            label="Attachment"
            description="Attach relevant documents or receipts"
          />
        </FormControl>
        <FormControl>
          <SelectInput
            name="owner"
            control={control}
            label="Owner"
            options={selectUsersData}
            isLoading={isUsersLoading}
            isFetchError={isUsersError}
            placeholder="Select Owner"
            description="User responsible for the account"
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
            name="tags"
            control={control}
            description="Tags or labels associated with the account"
            label="Tags"
            onCreate={createNewTag}
            placeholder="Select Tags"
            options={tagOptions}
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

export function GLTableSheet({ onOpenChange, open }: TableSheetProps) {
  const [isSubmitting, setIsSubmitting] = React.useState<boolean>(false);

  const { handleSubmit, control, getValues, setValue, reset } =
    useForm<GLAccountFormValues>({
      resolver: yupResolver(glAccountSchema),
      defaultValues: {
        status: "A",
        accountNumber: "",
        accountType: "",
        cashFlowType: "",
        accountSubType: "",
        accountClassification: "",
        parentAccount: "",
        attachment: null,
        owner: "",
        interestRate: null,
        isReconciled: false,
        isTaxRelevant: false,
        notes: "",
        tags: [],
      },
    });

  const mutation = useCustomMutation<GLAccountFormValues>(
    control,
    {
      method: "POST",
      path: "/gl_accounts/",
      successMessage: "General Ledger Account created successfully.",
      queryKeysToInvalidate: ["gl-account-table-data"],
      additionalInvalidateQueries: ["glAccounts"],
      closeModal: true,
      errorMessage: "Failed to create new general ledger account.",
    },
    () => setIsSubmitting(false),
    reset,
  );

  const onSubmit = (values: GLAccountFormValues) => {
    const cleanedValues = cleanObject(values);

    setIsSubmitting(true);
    mutation.mutate(cleanedValues);
  };

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
          className="flex flex-col h-full overflow-y-auto"
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
              isLoading={isSubmitting}
              loadingText="Saving Changes..."
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
