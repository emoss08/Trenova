import { ACForm } from "@/components/accessorial-charge-table-dialog";
import { Button } from "@/components/ui/button";
import { useCustomMutation } from "@/hooks/useCustomMutation";
import { formatToUserTimezone } from "@/lib/date";
import { accessorialChargeSchema } from "@/lib/validations/BillingSchema";
import { useTableStore } from "@/stores/TableStore";
import type {
  AccessorialCharge,
  AccessorialChargeFormValues as FormValues,
} from "@/types/billing";
import { yupResolver } from "@hookform/resolvers/yup";
import { useForm } from "react-hook-form";
import { Badge } from "./ui/badge";
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

function ACEditForm({
  accessorialCharge,
}: {
  accessorialCharge: AccessorialCharge;
}) {
  const { control, handleSubmit, reset } = useForm<FormValues>({
    resolver: yupResolver(accessorialChargeSchema),
    defaultValues: accessorialCharge,
  });

  const mutation = useCustomMutation<FormValues>(control, {
    method: "PUT",
    path: `/accessorial-charges/${accessorialCharge.id}/`,
    successMessage: "Accesorial Charge updated successfully.",
    queryKeysToInvalidate: "accessorialCharges",
    closeModal: true,
    reset,
    errorMessage: "Failed to update accesorial charge.",
  });

  const onSubmit = (values: FormValues) => {
    mutation.mutate(values);
  };

  return (
    <CredenzaBody>
      <form onSubmit={handleSubmit(onSubmit)}>
        <ACForm control={control} />
        <CredenzaFooter>
          <CredenzaClose asChild>
            <Button variant="outline" type="button">
              Cancel
            </Button>
          </CredenzaClose>
          <Button type="submit" isLoading={mutation.isPending}>
            Save Changes
          </Button>
        </CredenzaFooter>
      </form>
    </CredenzaBody>
  );
}

export function AccessorialChargeTableEditDialog({
  open,
  onOpenChange,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const [accessorialCharge] = useTableStore.use(
    "currentRecord",
  ) as AccessorialCharge[];

  if (!accessorialCharge) return null;

  return (
    <Credenza open={open} onOpenChange={onOpenChange}>
      <CredenzaContent>
        <CredenzaHeader>
          <CredenzaTitle className="flex">
            <span>{accessorialCharge.code}</span>
            <Badge className="ml-5" variant="purple">
              {accessorialCharge.id}
            </Badge>
          </CredenzaTitle>
        </CredenzaHeader>
        <CredenzaDescription>
          Last updated on {formatToUserTimezone(accessorialCharge.updatedAt)}
        </CredenzaDescription>
        <ACEditForm accessorialCharge={accessorialCharge} />
      </CredenzaContent>
    </Credenza>
  );
}
