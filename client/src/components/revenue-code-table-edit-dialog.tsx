import { Button } from "@/components/ui/button";
import { useCustomMutation } from "@/hooks/useCustomMutation";
import { useGLAccounts } from "@/hooks/useQueries";
import { formatDate } from "@/lib/date";
import { revenueCodeSchema } from "@/lib/validations/AccountingSchema";
import { useTableStore } from "@/stores/TableStore";
import type {
  RevenueCodeFormValues as FormValues,
  RevenueCode,
} from "@/types/accounting";
import { yupResolver } from "@hookform/resolvers/yup";
import React from "react";
import { useForm } from "react-hook-form";
import { RCForm } from "./revenue-code-table-dialog";
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

function RCEditForm({
  revenueCode,
  open,
}: {
  revenueCode: RevenueCode;
  open: boolean;
}) {
  const [isSubmitting, setIsSubmitting] = React.useState<boolean>(false);
  const { selectGLAccounts, isLoading, isError } = useGLAccounts(open);

  const { handleSubmit, control, reset } = useForm<FormValues>({
    resolver: yupResolver(revenueCodeSchema),
    defaultValues: revenueCode,
  });

  const mutation = useCustomMutation<FormValues>(
    control,
    {
      method: "PUT",
      path: `/revenue-codes/${revenueCode.id}/`,
      successMessage: "Revenue Code updated successfully.",
      queryKeysToInvalidate: ["revenue-code-table-data"],
      additionalInvalidateQueries: ["revenueCodes"],
      closeModal: true,
      errorMessage: "Failed to update revenue code.",
    },
    () => setIsSubmitting(false),
    reset,
  );

  const onSubmit = (values: FormValues) => {
    console.info("Submitting", values);
    setIsSubmitting(true);
    mutation.mutate(values);
  };

  return (
    <CredenzaBody>
      <form onSubmit={handleSubmit(onSubmit)}>
        <RCForm
          control={control}
          glAccounts={selectGLAccounts}
          isLoading={isLoading}
          isError={isError}
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

export function RevenueCodeTableEditDialog({
  open,
  onOpenChange,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const [revenueCode] = useTableStore.use("currentRecord") as RevenueCode[];
  if (!revenueCode) return null;

  return (
    <Credenza open={open} onOpenChange={onOpenChange}>
      <CredenzaContent>
        <CredenzaHeader>
          <CredenzaTitle>{revenueCode.code}</CredenzaTitle>
        </CredenzaHeader>
        <CredenzaDescription>
          Last updated on&nbsp;
          {revenueCode && formatDate(revenueCode.updatedAt)}
        </CredenzaDescription>
        {revenueCode && <RCEditForm revenueCode={revenueCode} open={open} />}
      </CredenzaContent>
    </Credenza>
  );
}
