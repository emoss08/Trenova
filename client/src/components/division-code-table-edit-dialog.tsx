import { Button } from "@/components/ui/button";
import { useCustomMutation } from "@/hooks/useCustomMutation";
import { useGLAccounts } from "@/hooks/useQueries";
import { formatDate } from "@/lib/date";
import { divisionCodeSchema } from "@/lib/validations/AccountingSchema";
import { useTableStore } from "@/stores/TableStore";
import type {
  DivisionCode,
  DivisionCodeFormValues as FormValues,
} from "@/types/accounting";
import { type TableSheetProps } from "@/types/tables";
import { yupResolver } from "@hookform/resolvers/yup";
import React from "react";
import { useForm } from "react-hook-form";
import { DCForm } from "./division-code-table-dialog";
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
    defaultValues: divisionCode,
  });

  const mutation = useCustomMutation<FormValues>(
    control,
    {
      method: "PUT",
      path: `/division-codes/${divisionCode.id}/`,
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
    <CredenzaBody>
      <form onSubmit={handleSubmit(onSubmit)}>
        <DCForm
          control={control}
          glAccounts={selectGLAccounts}
          isError={isError}
          isLoading={isLoading}
        />
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

export function DivisionCodeEditDialog({
  open,
  onOpenChange,
}: TableSheetProps) {
  const [divisionCode] = useTableStore.use("currentRecord") as DivisionCode[];

  if (!divisionCode) return null;

  return (
    <Credenza open={open} onOpenChange={onOpenChange}>
      <CredenzaContent>
        <CredenzaHeader>
          <CredenzaTitle>{divisionCode && divisionCode.code}</CredenzaTitle>
        </CredenzaHeader>
        <CredenzaDescription>
          Last updated on&nbsp;
          {divisionCode && formatDate(divisionCode.updatedAt)}
        </CredenzaDescription>
        {divisionCode && <DCEditForm divisionCode={divisionCode} open={open} />}
      </CredenzaContent>
    </Credenza>
  );
}
