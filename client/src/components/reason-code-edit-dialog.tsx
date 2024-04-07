import { Button } from "@/components/ui/button";
import { useCustomMutation } from "@/hooks/useCustomMutation";
import { formatDate } from "@/lib/date";
import { reasonCodeSchema } from "@/lib/validations/ShipmentSchema";
import { useTableStore } from "@/stores/TableStore";
import type {
  ReasonCodeFormValues as FormValues,
  ReasonCode,
} from "@/types/shipment";
import { type TableSheetProps } from "@/types/tables";
import { yupResolver } from "@hookform/resolvers/yup";
import React from "react";
import { useForm } from "react-hook-form";
import { ReasonCodeForm } from "./reason-code-table-dialog";
import {
  Credenza,
  CredenzaBody,
  CredenzaClose,
  CredenzaContent,
  CredenzaDescription,
  CredenzaFooter,
  CredenzaHeader,
  CredenzaTitle,
} from "./ui/credenza";

function ReasonCodeEditForm({ reasonCode }: { reasonCode: ReasonCode }) {
  const [isSubmitting, setIsSubmitting] = React.useState<boolean>(false);

  const { control, handleSubmit } = useForm<FormValues>({
    resolver: yupResolver(reasonCodeSchema),
    defaultValues: reasonCode,
  });

  const mutation = useCustomMutation<FormValues>(
    control,
    {
      method: "PUT",
      path: `/reason-codes/${reasonCode.id}/`,
      successMessage: "Reason Codes updated successfully.",
      queryKeysToInvalidate: ["reason-code-table-data"],
      closeModal: true,
      errorMessage: "Failed to update reason codes.",
    },
    () => setIsSubmitting(false),
  );

  const onSubmit = (values: FormValues) => {
    setIsSubmitting(true);
    mutation.mutate(values);
  };

  return (
    <CredenzaBody>
      <form onSubmit={handleSubmit(onSubmit)}>
        <ReasonCodeForm control={control} />
        <CredenzaFooter>
          <CredenzaClose asChild>
            <Button variant="outline" type="button">
              Cancel
            </Button>
          </CredenzaClose>
          <Button type="submit" isLoading={isSubmitting}>
            Save Changes
          </Button>
        </CredenzaFooter>
      </form>
    </CredenzaBody>
  );
}

export function ReasonCodeEditDialog({ onOpenChange, open }: TableSheetProps) {
  const [reasonCode] = useTableStore.use("currentRecord") as ReasonCode[];

  if (!reasonCode) return null;

  return (
    <Credenza open={open} onOpenChange={onOpenChange}>
      <CredenzaContent>
        <CredenzaHeader>
          <CredenzaTitle>{reasonCode && reasonCode.code}</CredenzaTitle>
        </CredenzaHeader>
        <CredenzaDescription>
          Last updated on&nbsp;
          {reasonCode && formatDate(reasonCode.updatedAt)}
        </CredenzaDescription>
        {reasonCode && <ReasonCodeEditForm reasonCode={reasonCode} />}
      </CredenzaContent>
    </Credenza>
  );
}
