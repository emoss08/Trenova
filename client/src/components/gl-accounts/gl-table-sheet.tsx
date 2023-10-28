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
import axios from "@/lib/axiosConfig";
import {
  accountClassificationChoices,
  accountSubTypeChoices,
  accountTypeChoices,
  cashFlowTypeChoices,
  statusChoices,
} from "@/lib/choices";
import { cn } from "@/lib/utils";
import { glAccountSchema } from "@/lib/validations/accounting";
import { TChoiceProps } from "@/types";
import { GLAccountFormValues } from "@/types/accounting";
import { TableSheetProps } from "@/types/tables";
import { yupResolver } from "@hookform/resolvers/yup";
import React from "react";
import {
  Control,
  useForm,
  UseFormGetValues,
  UseFormSetValue,
} from "react-hook-form";
import { useQueryClient } from "@tanstack/react-query";
import { CheckboxInput } from "../common/fields/checkbox";
import { FileField, InputField } from "../common/fields/input";
import {
  CreatableSelectField,
  SelectInput,
} from "../common/fields/select-input";
import { TextareaField } from "../common/fields/textarea";
import { useCustomMutation } from "@/hooks/useCustomMutation";
import { toast } from "@/components/ui/use-toast";
import { useGLAccounts, useTags, useUsers } from "@/hooks/useQueries";

export function GLForm({
  glAccounts,
  isGLAccountsError,
  isGLAccountsLoading,
  users,
  isUsersLoading,
  isUsersError,
  tags,
  isTagsLoading,
  isTagsError,
  control,
  getValues,
  setValue,
}: {
  glAccounts: TChoiceProps[];
  isGLAccountsLoading: boolean;
  isGLAccountsError: boolean;
  users: TChoiceProps[];
  isUsersLoading: boolean;
  isUsersError: boolean;
  tags: TChoiceProps[];
  isTagsLoading: boolean;
  isTagsError: boolean;
  control: Control<GLAccountFormValues>;
  getValues: UseFormGetValues<GLAccountFormValues>;
  setValue: UseFormSetValue<GLAccountFormValues>;
}) {
  const [isLoading, setIsLoading] = React.useState<boolean>(false);
  const queryClient = useQueryClient();

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

  const [tagOptions, setTagOptions] = React.useState(tags);
  const isTagsUpdating = isTagsLoading || isLoading;

  React.useEffect(() => {
    setTagOptions(tags);
  }, [tags]);

  return (
    <>
      <div className="flex-1 overflow-y-auto">
        <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-6 my-4">
          <div className="grid w-full max-w-sm items-center gap-0.5">
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
          </div>
          <div className="grid w-full max-w-sm items-center gap-0.5">
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
          </div>
        </div>
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6 my-4">
          <div className="grid w-full max-w-sm items-center gap-0.5">
            <SelectInput
              name="accountType"
              control={control}
              rules={{ required: true }}
              label="Account Type"
              options={accountTypeChoices}
              placeholder="Select Account Type"
              description="The Account Type of GL Account"
            />
          </div>
          <div className="grid w-full max-w-sm items-center gap-0.5">
            <SelectInput
              control={control}
              name="cashFlowType"
              label="Cash Flow Type"
              options={cashFlowTypeChoices}
              placeholder="Select Cash Flow Type"
              description="The Cash Flow Type of the GL Account"
            />
          </div>
          <div className="grid w-full max-w-sm items-center gap-0.5">
            <SelectInput
              name="accountSubType"
              control={control}
              label="Account Sub Type"
              options={accountSubTypeChoices}
              placeholder="Select Account Sub Type"
              description="The Account Sub Type of the GL Account"
            />
          </div>
          <div className="grid w-full max-w-sm items-center gap-0.5">
            <SelectInput
              name="accountClassification"
              control={control}
              label="Account Classification"
              options={accountClassificationChoices}
              placeholder="Select Account Classification"
              description="The Account Classification of the GL Account"
            />
          </div>
          <div className="grid w-full max-w-sm items-center gap-0.5">
            <SelectInput
              name="parentAccount"
              control={control}
              label="Parent Account"
              options={glAccounts}
              isLoading={isGLAccountsLoading}
              isFetchError={isGLAccountsError}
              placeholder="Select Parent Account"
              description="Parent account for hierarchical accounting"
            />
          </div>
          <div className="grid w-full max-w-sm items-center gap-0.5">
            <FileField
              name="attachment"
              control={control}
              label="Attachment"
              description="Attach relevant documents or receipts"
            />
          </div>
          <div className="grid w-full max-w-sm items-center gap-0.5">
            <SelectInput
              name="owner"
              control={control}
              label="Owner"
              options={users}
              isLoading={isUsersLoading}
              isFetchError={isUsersError}
              placeholder="Select Owner"
              description="User responsible for the account"
            />
          </div>
          <div className="grid w-full max-w-sm items-center gap-0.5">
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
          </div>
          <div className="grid w-full max-w-sm items-center gap-0.5">
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
          </div>
          <div className="grid w-full max-w-sm items-center gap-0.5">
            <CheckboxInput
              control={control}
              label="Is Reconciled"
              name="isReconciled"
              description="Indicates if the account is reconciled"
            />
          </div>
          <div className="grid w-full max-w-sm items-center gap-0.5">
            <CheckboxInput
              control={control}
              label="Is Tax Relevant"
              name="isTaxRelevant"
              description="Indicates if the account is relevant for tax calculations"
            />
          </div>
        </div>
        <TextareaField
          name="notes"
          control={control}
          label="Notes"
          placeholder="Notes"
          description="Additional notes or comments for the account"
        />
      </div>
    </>
  );
}

export function GLTableSheet({ onOpenChange, open }: TableSheetProps) {
  const [isSubmitting, setIsSubmitting] = React.useState<boolean>(false);
  const {
    selectGLAccounts,
    isError: glAccountsError,
    isLoading: glAccountsLoading,
  } = useGLAccounts(open);

  const {
    selectUsersData,
    isError: usersError,
    isLoading: usersLoading,
  } = useUsers(open);

  const {
    selectTags,
    isError: tagsError,
    isLoading: tagsLoading,
  } = useTags(open);

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
    toast,
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
    setIsSubmitting(true);
    mutation.mutate(values);
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
            glAccounts={selectGLAccounts}
            isGLAccountsLoading={glAccountsLoading}
            isGLAccountsError={glAccountsError}
            users={selectUsersData}
            isUsersLoading={usersLoading}
            isUsersError={usersError}
            tags={selectTags}
            isTagsLoading={tagsLoading}
            isTagsError={tagsError}
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
