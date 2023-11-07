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
import {
  DivisionCode,
  DivisionCodeFormValues as FormValues,
} from "@/types/accounting";
import { yupResolver } from "@hookform/resolvers/yup";
import { divisionCodeSchema } from "@/lib/validations/accounting";
import { useForm } from "react-hook-form";
import { useCustomMutation } from "@/hooks/useCustomMutation";
import { toast } from "@/components/ui/use-toast";
import { DCForm } from "./dc-table-dialog";
import { useGLAccounts } from "@/hooks/useQueries";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { useTableStore } from "@/stores/TableStore";
import { formatDate } from "@/lib/date";

export function DCEditForm({
  divisionCode,
  open,
}: {
  divisionCode: DivisionCode;
  open: boolean;
}) {
  const [isSubmitting, setIsSubmitting] = React.useState<boolean>(false);

  const { control, reset, handleSubmit } = useForm<FormValues>({
    resolver: yupResolver(divisionCodeSchema),
    defaultValues: {
      status: divisionCode.status,
      code: divisionCode.code,
      description: divisionCode.description,
      expenseAccount: divisionCode?.expenseAccount || "",
      cashAccount: divisionCode?.cashAccount || "",
      apAccount: divisionCode?.apAccount || "",
    },
  });

  const mutation = useCustomMutation<FormValues>(
    control,
    toast,
    {
      method: "PUT",
      path: `/division_codes/${divisionCode.id}/`,
      successMessage: "Division Code updated successfully.",
      queryKeysToInvalidate: ["division-code-table-data"],
      closeModal: true,
      errorMessage: "Failed to update division code.",
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
    <form onSubmit={handleSubmit(onSubmit)}>
      <DCForm
        control={control}
        glAccounts={selectGLAccounts}
        isError={isError}
        isLoading={isLoading}
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

export function DCTableEditDialog({
  open,
  onOpenChange,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const [divisionCode] = useTableStore.use("currentRecord");

  if (!divisionCode) return null;

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{divisionCode && divisionCode.code}</DialogTitle>
        </DialogHeader>
        <DialogDescription>
          Last updated on&nbsp;
          {divisionCode && formatDate(divisionCode.modified)}
        </DialogDescription>
        {divisionCode && <DCEditForm divisionCode={divisionCode} open={open} />}
      </DialogContent>
    </Dialog>
  );
}
