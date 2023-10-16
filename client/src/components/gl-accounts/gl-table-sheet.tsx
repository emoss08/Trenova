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

import React from "react";
import { Button } from "@/components/ui/button";
import {
  Sheet,
  SheetClose,
  SheetContent,
  SheetDescription,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
import { cn } from "@/lib/utils";
import { TableSheetProps } from "@/types/tables";
import {
  accountClassificationChoices,
  accountSubTypeChoices,
  accountTypeChoices,
  cashFlowTypeChoices,
  statusChoices,
} from "@/lib/choices";
import { FileField, InputField } from "../ui/input";
import { TChoiceProps } from "@/types";
import { useGLAccounts } from "@/hooks/useGLAccounts";
import { TextareaField } from "../ui/textarea";
import {
  CreatableSelectField,
  SelectInput,
  type Option,
} from "../ui/select-input";
import { CheckboxInput } from "../ui/checkbox";
import { useUsers } from "@/hooks/useUsers";
import { useTags } from "@/hooks/useTags";
import axios from "@/lib/AxiosConfig";
import { useQueryClient } from "react-query";

function GLForm({
  glAccounts,
  isGLAccountsError,
  isGLAccountsLoading,
  users,
  isUsersLoading,
  isUsersError,
  tags,
  isTagsLoading,
  isTagsError,
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
}) {
  const [isLoading, setIsLoading] = React.useState<boolean>(false);
  const [tagOptions, setTagOptions] = React.useState(tags);
  const [tagValue, setTagValue] = React.useState<Option | null>();

  const queryClient = useQueryClient();

  const createNewTag = (inputValue: string) => {
    axios.post("/tags/", { name: inputValue }).then((res) => {
      setTagOptions((prev) => [
        ...prev,
        { label: inputValue, value: res.data.id },
      ]);
      setTagValue({ label: inputValue, value: res.data.id });
    });
    queryClient.invalidateQueries("tags");
  };

  React.useEffect(() => {
    setTagOptions(tags);
  }, [tags]);

  const handleCreate = (inputValue: string) => {
    setIsLoading(true);
    createNewTag(inputValue);
    setIsLoading(false);
  };

  const isTagsUpdating = isTagsLoading || isLoading;

  return (
    <>
      <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-6 my-4">
        <div className="grid w-full max-w-sm items-center gap-0.5">
          <SelectInput
            label="Status"
            options={statusChoices}
            withAsterisk
            placeholder="Select Status"
            description="Status of the General Ledger Account"
          />
        </div>
        <div className="grid w-full max-w-sm items-center gap-0.5">
          <InputField
            label="Account Number"
            autoCapitalize="none"
            autoCorrect="off"
            type="text"
            placeholder="Account Number"
            autoComplete="accountNumber"
            description="The account number of the account"
            withAsterisk
          />
        </div>
      </div>
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6 my-4">
        <div className="grid w-full max-w-sm items-center gap-0.5">
          <SelectInput
            label="Account Type"
            options={accountTypeChoices}
            placeholder="Select Account Type"
            description="The Account Type of GL Account"
            withAsterisk
          />
        </div>
        <div className="grid w-full max-w-sm items-center gap-0.5">
          <SelectInput
            label="Cash Flow Type"
            options={cashFlowTypeChoices}
            placeholder="Select Cash Flow Type"
            description="The Cash Flow Type of the GL Account"
          />
        </div>
        <div className="grid w-full max-w-sm items-center gap-0.5">
          <SelectInput
            label="Account Sub Type"
            options={accountSubTypeChoices}
            placeholder="Select Account Sub Type"
            description="The Account Sub Type of the GL Account"
          />
        </div>
        <div className="grid w-full max-w-sm items-center gap-0.5">
          <SelectInput
            label="Account Classification"
            options={accountClassificationChoices}
            placeholder="Select Account Classification"
            description="The Account Classification of the GL Account"
          />
        </div>
        <div className="grid w-full max-w-sm items-center gap-0.5">
          <SelectInput
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
            label="Attachment"
            description="Attach relevant documents or receipts"
          />
        </div>
        <div className="grid w-full max-w-sm items-center gap-0.5">
          <SelectInput
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
            id="tags"
            description="Tags or labels associated with the account"
            label="Tags"
            onCreateOption={handleCreate}
            placeholder="Select Tags"
            options={tagOptions}
            isLoading={isTagsUpdating}
            isFetchError={isTagsError}
            value={tagValue}
            onChange={(newValue) => setTagValue(newValue as Option)}
            isMulti
          />
        </div>
        <div className="grid w-full max-w-sm items-center gap-0.5">
          <CheckboxInput
            label="Is Reconciled"
            description="Indicates if the account is reconciled"
          />
        </div>
        <div className="grid w-full max-w-sm items-center gap-0.5">
          <CheckboxInput
            label="Is Tax Relevant"
            description="Indicates if the account is relevant for tax calculations"
          />
        </div>
      </div>
      <TextareaField
        label="Notes"
        placeholder="Notes"
        description="Additional notes or comments for the account"
        withAsterisk
      />
    </>
  );
}

export function GLTableSheet({ onOpenChange, open }: TableSheetProps) {
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

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent className={cn("w-full xl:w-1/2")}>
        <SheetHeader>
          <SheetTitle>Add New GL Account</SheetTitle>
          <SheetDescription>
            <SheetDescription>
              Use this form to add a new general ledger account to the system.
            </SheetDescription>
          </SheetDescription>
        </SheetHeader>
        <GLForm
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
        <SheetFooter>
          <SheetClose asChild>
            <Button type="submit">Save changes</Button>
          </SheetClose>
        </SheetFooter>
      </SheetContent>
    </Sheet>
  );
}
