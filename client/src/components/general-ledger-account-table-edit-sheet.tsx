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
import { formatToUserTimezone } from "@/lib/date";
import { cn } from "@/lib/utils";
import { glAccountSchema } from "@/lib/validations/AccountingSchema";
import { useTableStore } from "@/stores/TableStore";
import type {
  GLAccountFormValues,
  GeneralLedgerAccount,
} from "@/types/accounting";
import type { TableSheetProps } from "@/types/tables";
import { yupResolver } from "@hookform/resolvers/yup";
import React from "react";
import { useForm } from "react-hook-form";
import { GLForm } from "./general-ledger-account-table-sheet";

function GLEditForm({
  glAccount,
  open,
  onOpenChange,
}: {
  glAccount: GeneralLedgerAccount;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const [isSubmitting, setIsSubmitting] = React.useState<boolean>(false);

  const { handleSubmit, control, getValues, setValue } =
    useForm<GLAccountFormValues>({
      resolver: yupResolver(glAccountSchema),
      defaultValues: glAccount,
    });

  const mutation = useCustomMutation<GLAccountFormValues>(
    control,
    {
      method: "PUT",
      path: `/general-ledger-accounts/${glAccount.id}/`,
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
    mutation.mutate(values);
  };

  return (
    <form
      onSubmit={handleSubmit(onSubmit)}
      className="flex h-full flex-col overflow-y-auto"
    >
      <GLForm
        control={control}
        getValues={getValues}
        setValue={setValue}
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
        <Button type="submit" isLoading={isSubmitting} className="w-full">
          Save Changes
        </Button>
      </SheetFooter>
    </form>
  );
}

export function GeneralLedgerAccountTableEditSheet({
  onOpenChange,
  open,
}: TableSheetProps) {
  const [glAccount] = useTableStore.use(
    "currentRecord",
  ) as GeneralLedgerAccount[];

  if (!glAccount) return null;

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent className={cn("w-full xl:w-1/2")}>
        <SheetHeader>
          <SheetTitle>{glAccount && glAccount.accountNumber}</SheetTitle>
          <SheetDescription>
            Last updated on{" "}
            {glAccount && formatToUserTimezone(glAccount.updatedAt)}
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
