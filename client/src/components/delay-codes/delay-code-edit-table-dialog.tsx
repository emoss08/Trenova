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

import { DelayCodeForm } from "@/components/delay-codes/delay-code-table-dialog";
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
import { formatDate } from "@/lib/date";
import { delayCodeSchema } from "@/lib/validations/DispatchSchema";
import { useTableStore } from "@/stores/TableStore";
import { DelayCode, DelayCodeFormValues } from "@/types/dispatch";
import { yupResolver } from "@hookform/resolvers/yup";
import React from "react";
import { useForm } from "react-hook-form";

function DelayCodeEditForm({ delayCode }: { delayCode: DelayCode }) {
  const [isSubmitting, setIsSubmitting] = React.useState<boolean>();

  const { control, reset, handleSubmit } = useForm<DelayCodeFormValues>({
    resolver: yupResolver(delayCodeSchema),
    defaultValues: {
      status: delayCode.status,
      code: delayCode.code,
      description: delayCode.description,
      fCarrierOrDriver: delayCode.fCarrierOrDriver,
    },
  });

  const mutation = useCustomMutation<DelayCodeFormValues>(
    control,
    {
      method: "PUT",
      path: `/delay_codes/${delayCode.id}/`,
      successMessage: "Delay Code updated successfully.",
      queryKeysToInvalidate: ["delay-code-table-data"],
      closeModal: true,
      errorMessage: "Failed to update new delay code.",
    },
    () => setIsSubmitting(false),
    reset,
  );

  const onSubmit = (values: DelayCodeFormValues) => {
    setIsSubmitting(true);
    mutation.mutate(values);
  };

  return (
    <form onSubmit={handleSubmit(onSubmit)}>
      <DelayCodeForm control={control} />
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

export function DelayCodeEditDialog({
  open,
  onOpenChange,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const [delayCode] = useTableStore.use("currentRecord");

  if (!delayCode) return null;

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{delayCode && delayCode.code}</DialogTitle>
        </DialogHeader>
        <DialogDescription>
          Last updated on&nbsp;
          {delayCode && formatDate(delayCode.modified)}
        </DialogDescription>
        {delayCode && <DelayCodeEditForm delayCode={delayCode} />}
      </DialogContent>
    </Dialog>
  );
}
