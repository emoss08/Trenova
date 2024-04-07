import { HazmatSegRulesForm } from "@/components/hazmat-seg-rules-table-dialog";
import { Button } from "@/components/ui/button";
import { DialogFooter } from "@/components/ui/dialog";
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
    <Credenza open={open} onOpenChange={onOpenChange}>
      <CredenzaContent>
        <CredenzaHeader>
          <CredenzaTitle>{hazmatRule.classA} </CredenzaTitle>
        </CredenzaHeader>
        <CredenzaDescription>
          Last updated on&nbsp;
          {formatDate(hazmatRule.updatedAt)}
        </CredenzaDescription>
        <HazmatRuleEditForm hazmatRule={hazmatRule} />
      </CredenzaContent>
    </Credenza>
  ) : null;
}
