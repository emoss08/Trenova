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
import { fleetCodeSchema } from "@/lib/validations/DispatchSchema";
import { useTableStore } from "@/stores/TableStore";
import { FleetCode, FleetCodeFormValues as FormValues } from "@/types/dispatch";
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
import { FleetCodeForm } from "./fleet-code-table-dialog";

function FleetCodeEditForm({
  fleetCode,
  open,
}: {
  fleetCode: FleetCode;
  open: boolean;
}) {
  const [isSubmitting, setIsSubmitting] = React.useState<boolean>(false);
  const { control, reset, handleSubmit } = useForm<FormValues>({
    resolver: yupResolver(fleetCodeSchema),
    defaultValues: {
      status: fleetCode.status,
      code: fleetCode.code,
      description: fleetCode?.description || "",
      revenueGoal: fleetCode?.revenueGoal || "",
      deadheadGoal: fleetCode?.deadheadGoal || "",
      mileageGoal: fleetCode?.mileageGoal || "",
      manager: fleetCode?.manager || "",
    },
  });

  const mutation = useCustomMutation<FormValues>(
    control,
    {
      method: "PUT",
      path: `/fleet_codes/${fleetCode.id}/`,
      successMessage: "Fleet Code updated successfully.",
      queryKeysToInvalidate: ["fleet-code-table-data"],
      closeModal: true,
      errorMessage: "Failed to update fleet code.",
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
      <FleetCodeForm control={control} open={open} />
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

export function FleetCodeEditDialog({
  open,
  onOpenChange,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const [fleetCode] = useTableStore.use("currentRecord");

  if (!fleetCode) return null;

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{fleetCode && fleetCode.code}</DialogTitle>
        </DialogHeader>
        <DialogDescription>
          Last updated on&nbsp;
          {fleetCode && formatDate(fleetCode.modified)}
        </DialogDescription>
        {fleetCode && <FleetCodeEditForm fleetCode={fleetCode} open={open} />}
      </DialogContent>
    </Dialog>
  );
}
