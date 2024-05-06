import { useCustomMutation } from "@/hooks/useCustomMutation";
import { formatToUserTimezone } from "@/lib/date";
import { fleetCodeSchema } from "@/lib/validations/DispatchSchema";
import { useTableStore } from "@/stores/TableStore";
import type {
  FleetCode,
  FleetCodeFormValues as FormValues,
} from "@/types/dispatch";
import { yupResolver } from "@hookform/resolvers/yup";
import React from "react";
import { useForm } from "react-hook-form";
import { FleetCodeForm } from "./fleet-code-table-dialog";
import { Button } from "./ui/button";
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
    defaultValues: fleetCode,
  });

  const mutation = useCustomMutation<FormValues>(
    control,
    {
      method: "PUT",
      path: `/fleet-codes/${fleetCode.id}/`,
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
    <CredenzaBody>
      <form onSubmit={handleSubmit(onSubmit)}>
        <FleetCodeForm control={control} open={open} />
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

export function FleetCodeEditDialog({
  open,
  onOpenChange,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const [fleetCode] = useTableStore.use("currentRecord") as FleetCode[];

  if (!fleetCode) return null;

  return (
    <Credenza open={open} onOpenChange={onOpenChange}>
      <CredenzaContent>
        <CredenzaHeader>
          <CredenzaTitle>{fleetCode && fleetCode.code} </CredenzaTitle>
        </CredenzaHeader>
        <CredenzaDescription>
          Last updated on&nbsp;
          {fleetCode && formatToUserTimezone(fleetCode.updatedAt)}
        </CredenzaDescription>
        {fleetCode && <FleetCodeEditForm fleetCode={fleetCode} open={open} />}
      </CredenzaContent>
    </Credenza>
  );
}
