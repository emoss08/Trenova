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

import { HazmatSegRulesForm } from "@/components/hazmat-seg-rules-table-dialog";
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
import { useHazmatSegRulesForm } from "@/lib/validations/ShipmentSchema";
import { useTableStore } from "@/stores/TableStore";
import type {
  HazardousMaterialSegregationRuleFormValues as FormValues,
  HazardousMaterialSegregationRule,
} from "@/types/shipment";
import React from "react";
import { FormProvider } from "react-hook-form";

function HazmatRuleEditForm({
  hazmatRule,
}: {
  hazmatRule: HazardousMaterialSegregationRule;
}) {
  const [isSubmitting, setIsSubmitting] = React.useState<boolean>(false);

  const { hazmatSegRulesForm } = useHazmatSegRulesForm(hazmatRule);

  const mutation = useCustomMutation<FormValues>(
    hazmatSegRulesForm.control,
    {
      method: "PUT",
      path: `/hazardous-material-segregations/${hazmatRule.id}/`,
      successMessage: "Hazardous Material updated successfully.",
      queryKeysToInvalidate: ["hazardous-material-table-data"],
      closeModal: true,
      errorMessage: "Failed to update Hazardous Material.",
    },
    () => setIsSubmitting(false),
    hazmatSegRulesForm.reset,
  );

  const onSubmit = (values: FormValues) => {
    setIsSubmitting(true);
    mutation.mutate(values);
  };

  return (
    <FormProvider {...hazmatSegRulesForm}>
      <form onSubmit={hazmatSegRulesForm.handleSubmit(onSubmit)}>
        <HazmatSegRulesForm />
        <DialogFooter className="mt-6">
          <Button type="submit" isLoading={isSubmitting}>
            Save
          </Button>
        </DialogFooter>
      </form>
    </FormProvider>
  );
}

export function HazardousMaterialEditDialog({
  open,
  onOpenChange,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const [hazmatRule] = useTableStore.use(
    "currentRecord",
  ) as HazardousMaterialSegregationRule[];

  return hazmatRule ? (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{hazmatRule.classA}</DialogTitle>
        </DialogHeader>
        <DialogDescription>
          Last updated on&nbsp;
          {formatDate(hazmatRule.updatedAt)}
        </DialogDescription>
        <HazmatRuleEditForm hazmatRule={hazmatRule} />
      </DialogContent>
    </Dialog>
  ) : null;
}
