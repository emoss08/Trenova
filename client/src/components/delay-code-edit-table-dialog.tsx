import { Button } from "@/components/ui/button";
import { useCustomMutation } from "@/hooks/useCustomMutation";
import { formatDate } from "@/lib/date";
import { delayCodeSchema } from "@/lib/validations/DispatchSchema";
import { useTableStore } from "@/stores/TableStore";
import type { DelayCode, DelayCodeFormValues } from "@/types/dispatch";
import { yupResolver } from "@hookform/resolvers/yup";
import React from "react";
import { useForm } from "react-hook-form";
import { DelayCodeForm } from "./delay-code-table-dialog";
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

function DelayCodeEditForm({ delayCode }: { delayCode: DelayCode }) {
  const [isSubmitting, setIsSubmitting] = React.useState<boolean>();

  const { control, reset, handleSubmit } = useForm<DelayCodeFormValues>({
    resolver: yupResolver(delayCodeSchema),
    defaultValues: delayCode,
  });

  const mutation = useCustomMutation<DelayCodeFormValues>(
    control,
    {
      method: "PUT",
      path: `/delay-codes/${delayCode.id}/`,
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
    <CredenzaBody>
      <form onSubmit={handleSubmit(onSubmit)}>
        <DelayCodeForm control={control} />
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

export function DelayCodeEditDialog({
  open,
  onOpenChange,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const [delayCode] = useTableStore.use("currentRecord") as DelayCode[];

  if (!delayCode) return null;

  return (
    <Credenza open={open} onOpenChange={onOpenChange}>
      <CredenzaContent>
        <CredenzaHeader>
          <CredenzaTitle>{delayCode && delayCode.code}</CredenzaTitle>
        </CredenzaHeader>
        <CredenzaDescription>
          Last updated on&nbsp;
          {delayCode && formatDate(delayCode.updatedAt)}
        </CredenzaDescription>
        {delayCode && <DelayCodeEditForm delayCode={delayCode} />}
      </CredenzaContent>
    </Credenza>
  );
}
