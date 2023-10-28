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
import { useUsers } from "@/hooks/useUsers";
import { cn } from "@/lib/utils";
import { glAccountSchema } from "@/lib/validations/accounting";
import { useTableStore } from "@/stores/TableStore";
import { GeneralLedgerAccount, GLAccountFormValues } from "@/types/accounting";
import { TableSheetProps } from "@/types/tables";
import { yupResolver } from "@hookform/resolvers/yup";
import React from "react";
import { useForm } from "react-hook-form";
import { toast } from "../ui/use-toast";
import { GLForm } from "./gl-table-sheet";
import { formatDate } from "@/lib/date";
import { useGLAccounts, useTags } from "@/hooks/useQueries";

function GLEditForm({
  glAccount,
  open,
  onOpenChange,
}: {
  glAccount: GeneralLedgerAccount;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
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

  const [isSubmitting, setIsSubmitting] = React.useState<boolean>(false);

  const { handleSubmit, control, getValues, setValue } =
    useForm<GLAccountFormValues>({
      resolver: yupResolver(glAccountSchema),
      defaultValues: {
        status: glAccount.status,
        accountNumber: glAccount.accountNumber,
        accountType: glAccount.accountType,
        cashFlowType: glAccount.cashFlowType,
        accountSubType: glAccount.accountSubType,
        accountClassification: glAccount.accountClassification,
        parentAccount: glAccount.parentAccount,
        attachment: glAccount.attachment,
        owner: glAccount.owner,
        interestRate: glAccount.interestRate,
        isReconciled: glAccount.isReconciled,
        isTaxRelevant: glAccount.isTaxRelevant,
        notes: glAccount.notes,
        tags: glAccount.tags,
      },
    });

  const mutation = useCustomMutation<GLAccountFormValues>(
    control,
    toast,
    {
      method: "PUT",
      path: `/gl_accounts/${glAccount.id}/`,
      successMessage: "General Ledger Account updated successfully.",
      queryKeysToInvalidate: ["gl-account-table-data"],
      additionalInvalidateQueries: ["glAccounts"],
      closeModal: true,
      errorMessage: "Failed to update general ledger account.",
    },
    () => setIsSubmitting(false),
  );

  const onSubmit = (values: GLAccountFormValues) => {
    setIsSubmitting(true);
    console.log(values);
    mutation.mutate(values);
  };

  return (
    <form
      onSubmit={handleSubmit(onSubmit)}
      className="flex flex-col h-full overflow-y-auto"
    >
      <GLForm
        control={control}
        getValues={getValues}
        setValue={setValue}
        users={selectUsersData}
        isUsersError={usersError}
        isUsersLoading={usersLoading}
        tags={selectTags}
        isTagsError={tagsError}
        isTagsLoading={tagsLoading}
        glAccounts={selectGLAccounts}
        isGLAccountsLoading={glAccountsError}
        isGLAccountsError={glAccountsLoading}
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
          Save Changes
        </Button>
      </SheetFooter>
    </form>
  );
}

export function GLTableEditSheet({ onOpenChange, open }: TableSheetProps) {
  const [glAccount] = useTableStore.use("currentRecord");

  if (!glAccount) return null;

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent className={cn("w-full xl:w-1/2")}>
        <SheetHeader>
          <SheetTitle>{glAccount && glAccount.accountNumber}</SheetTitle>
          <SheetDescription>
            Last updated on {glAccount && formatDate(glAccount.modified)}
          </SheetDescription>
        </SheetHeader>
        {glAccount && (
          <GLEditForm
            glAccount={glAccount}
            open={open}
            onOpenChange={onOpenChange}
          />
        )}
      </SheetContent>
    </Sheet>
  );
}
