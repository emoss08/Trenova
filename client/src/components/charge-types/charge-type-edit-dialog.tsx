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

import { useCustomMutation } from "@/hooks/useCustomMutation";
import { formatDate } from "@/lib/date";
import { chargeTypeSchema } from "@/lib/validations/BillingSchema";
import { useTableStore } from "@/stores/TableStore";
import { ChargeTypeFormValues as FormValues } from "@/types/billing";
import { TableSheetProps } from "@/types/tables";
import { yupResolver } from "@hookform/resolvers/yup";
import React from "react";
import { useForm } from "react-hook-form";
import { Button } from "../ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "../ui/dialog";
import { ChargeTypeForm } from "./charge-type-dialog";

function ChargeTypeEditForm({
  chargeType,
}: {
  chargeType: Record<string, any>;
}) {
  const [isSubmitting, setIsSubmitting] = React.useState<boolean>(false);
  const { control, reset, handleSubmit } = useForm<FormValues>({
    resolver: yupResolver(chargeTypeSchema),
    defaultValues: {
      status: chargeType.status,
      name: chargeType.name,
      description: chargeType.description,
    },
  });

  const mutation = useCustomMutation<FormValues>(
    control,
    {
      method: "PUT",
      path: `/charge_types/${chargeType.id}/`,
      successMessage: "Charge Type updated successfully.",
      queryKeysToInvalidate: ["charge-type-table-data"],
      closeModal: true,
      errorMessage: "Failed to create update charge type.",
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
      <ChargeTypeForm control={control} />
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

export function ChargeTypeEditSheet({ onOpenChange, open }: TableSheetProps) {
  const [chargeType] = useTableStore.use("currentRecord");

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{chargeType && chargeType.name}</DialogTitle>
        </DialogHeader>
        <DialogDescription>
          Last updated on {chargeType && formatDate(chargeType.modified)}
        </DialogDescription>
        {chargeType && <ChargeTypeEditForm chargeType={chargeType} />}
      </DialogContent>
    </Dialog>
  );
}
