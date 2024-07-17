/**
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
import { DialogFooter } from "@/components/ui/dialog";
import { useCustomMutation } from "@/hooks/useCustomMutation";
import { getHazardousClassLabel } from "@/lib/choices";
import { formatToUserTimezone } from "@/lib/date";
import { useHazmatSegRulesForm } from "@/lib/validations/ShipmentSchema";
import { useTableStore } from "@/stores/TableStore";
import type {
    HazardousMaterialSegregationRuleFormValues as FormValues,
    HazardousMaterialSegregationRule,
} from "@/types/shipment";
import { FormProvider } from "react-hook-form";
import { Badge } from "./ui/badge";
import {
    Credenza,
    CredenzaContent,
    CredenzaDescription,
    CredenzaHeader,
    CredenzaTitle,
} from "./ui/credenza";

function HazmatRuleEditForm({
  hazmatRule,
}: {
  hazmatRule: HazardousMaterialSegregationRule;
}) {
  const { hazmatSegRulesForm } = useHazmatSegRulesForm(hazmatRule);

  const { reset } = hazmatSegRulesForm;

  const mutation = useCustomMutation<FormValues>(hazmatSegRulesForm.control, {
    method: "PUT",
    path: `/hazardous-material-segregations/${hazmatRule.id}/`,
    successMessage: "Hazardous Material updated successfully.",
    queryKeysToInvalidate: "hazardousMaterialsSegregations",
    closeModal: true,
    reset,
    errorMessage: "Failed to update Hazardous Material.",
  });

  const onSubmit = (values: FormValues) => mutation.mutate(values);

  return (
    <FormProvider {...hazmatSegRulesForm}>
      <form onSubmit={hazmatSegRulesForm.handleSubmit(onSubmit)}>
        <HazmatSegRulesForm />
        <DialogFooter className="mt-6">
          <Button type="submit" isLoading={mutation.isPending}>
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

  if (!hazmatRule) return null;

  return (
    <Credenza open={open} onOpenChange={onOpenChange}>
      <CredenzaContent>
        <CredenzaHeader>
          <CredenzaTitle className="flex">
            <span className="w-32 truncate">
              {getHazardousClassLabel(hazmatRule.classA)}
            </span>
            <Badge className="ml-5" variant="purple">
              {hazmatRule.id}
            </Badge>
          </CredenzaTitle>
        </CredenzaHeader>
        <CredenzaDescription>
          Last updated on {formatToUserTimezone(hazmatRule.updatedAt)}
        </CredenzaDescription>
        <HazmatRuleEditForm hazmatRule={hazmatRule} />
      </CredenzaContent>
    </Credenza>
  );
}
