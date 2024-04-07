import { Button } from "@/components/ui/button";
import { useCustomMutation } from "@/hooks/useCustomMutation";
import { formatDate } from "@/lib/date";
import { qualifierCodeSchema } from "@/lib/validations/StopSchema";
import { useTableStore } from "@/stores/TableStore";
import type {
  QualifierCodeFormValues as FormValues,
  QualifierCode,
} from "@/types/stop";
import { type TableSheetProps } from "@/types/tables";
import { yupResolver } from "@hookform/resolvers/yup";
import React from "react";
import { useForm } from "react-hook-form";
import { QualifierCodeForm } from "./qualifier-code-table-dialog";
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

function QualifierCodeEditForm({
  qualifierCode,
}: {
  qualifierCode: QualifierCode;
}) {
  const [isSubmitting, setIsSubmitting] = React.useState<boolean>(false);

  const { control, handleSubmit } = useForm<FormValues>({
    resolver: yupResolver(qualifierCodeSchema),
    defaultValues: qualifierCode,
  });

  const mutation = useCustomMutation<FormValues>(
    control,
    {
      method: "PUT",
      path: `/qualifier-codes/${qualifierCode.id}/`,
      successMessage: "Qualifier Code updated successfully.",
      queryKeysToInvalidate: ["qualifier-code-table-data"],
      closeModal: true,
      errorMessage: "Failed to update qualifier code",
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
        <QualifierCodeForm control={control} />
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

export function QualifierCodeEditDialog({
  onOpenChange,
  open,
}: TableSheetProps) {
  const [qualifierCode] = useTableStore.use("currentRecord") as QualifierCode[];

  if (!qualifierCode) return null;

  return (
    <Credenza open={open} onOpenChange={onOpenChange}>
      <CredenzaContent>
        <CredenzaHeader>
          <CredenzaTitle>{qualifierCode && qualifierCode.code} </CredenzaTitle>
        </CredenzaHeader>
        <CredenzaDescription>
          Last updated on&nbsp;
          {qualifierCode && formatDate(qualifierCode.createdAt)}
        </CredenzaDescription>
        {qualifierCode && (
          <QualifierCodeEditForm qualifierCode={qualifierCode} />
        )}
      </CredenzaContent>
    </Credenza>
  );
}
