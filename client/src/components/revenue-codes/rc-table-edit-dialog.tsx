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

import { RCForm } from "@/components/revenue-codes/rc-table-dialog";
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
import { formatDate } from "@/lib/date";
import { revenueCodeSchema } from "@/lib/validations/accounting";
import { useTableStore } from "@/stores/TableStore";
import {
  RevenueCodeFormValues as FormValues,
  RevenueCode,
} from "@/types/accounting";
import { yupResolver } from "@hookform/resolvers/yup";
import React from "react";
import { useForm } from "react-hook-form";

function RCEditForm({
  revenueCode,
  open,
}: {
  revenueCode: RevenueCode;
  open: boolean;
}) {
  const [isSubmitting, setIsSubmitting] = React.useState<boolean>(false);
  const { selectGLAccounts, isLoading, isError } = useGLAccounts(open);

  const { handleSubmit, control, reset } = useForm<FormValues>({
    resolver: yupResolver(revenueCodeSchema),
    defaultValues: {
      code: revenueCode.code,
      description: revenueCode?.description || "",
      expenseAccount: revenueCode?.expenseAccount || "",
      revenueAccount: revenueCode?.revenueAccount || "",
    },
  });

  const mutation = useCustomMutation<FormValues>(
    control,
    {
      method: "PUT",
      path: `/revenue_codes/${revenueCode.id}/`,
      successMessage: "Revenue Code updated successfully.",
      queryKeysToInvalidate: ["revenue-code-table-data"],
      additionalInvalidateQueries: ["revenueCodes"],
      closeModal: true,
      errorMessage: "Failed to update revenue code.",
    },
    () => setIsSubmitting(false),
    reset,
  );

  const onSubmit = (values: FormValues) => {
    setIsSubmitting(true);
    mutation.mutate(values);
  };

  return (
    <form onSubmit={handleSubmit(onSubmit)}>
      <RCForm
        control={control}
        glAccounts={selectGLAccounts}
        isLoading={isLoading}
        isError={isError}
      />
      <DialogFooter className="mt-6">
        <Button
          type="submit"
          isLoading={isSubmitting}
          loadingText="Saving Changes..."
        >
          Save
        </Button>
      </DialogFooter>
    </form>
  );
}

export function RCTableEditDialog({
  open,
  onOpenChange,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const [revenueCode] = useTableStore.use("currentRecord");

  if (!revenueCode) return null;

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{revenueCode.code}</DialogTitle>
        </DialogHeader>
        <DialogDescription>
          Last updated on&nbsp;
          {revenueCode && formatDate(revenueCode.modified)}
        </DialogDescription>
        {revenueCode && <RCEditForm revenueCode={revenueCode} open={open} />}
      </DialogContent>
    </Dialog>
  );
}
