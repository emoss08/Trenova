import { Button } from "@/components/ui/button";
import { useCustomMutation } from "@/hooks/useCustomMutation";
import { formatToUserTimezone } from "@/lib/date";
import { chargeTypeSchema } from "@/lib/validations/BillingSchema";
import { useTableStore } from "@/stores/TableStore";
import type {
  ChargeType,
  ChargeTypeFormValues as FormValues,
} from "@/types/billing";
import { type TableSheetProps } from "@/types/tables";
import { yupResolver } from "@hookform/resolvers/yup";
import { useForm } from "react-hook-form";
import { ChargeTypeForm } from "./charge-type-dialog";
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

function ChargeTypeEditForm({ chargeType }: { chargeType: ChargeType }) {
  const { control, reset, handleSubmit } = useForm<FormValues>({
    resolver: yupResolver(chargeTypeSchema),
    defaultValues: chargeType,
  });

  const mutation = useCustomMutation<FormValues>(control, {
    method: "PUT",
    path: `/charge-types/${chargeType.id}/`,
    successMessage: "Charge Type updated successfully.",
    queryKeysToInvalidate: ["charge-type-table-data"],
    closeModal: true,
    errorMessage: "Failed to create update charge type.",
  });

  const onSubmit = (values: FormValues) => {
    mutation.mutate(values);
    reset(values);
  };

  return (
    <CredenzaBody>
      <form onSubmit={handleSubmit(onSubmit)}>
        <ChargeTypeForm control={control} />
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

export function ChargeTypeEditSheet({ onOpenChange, open }: TableSheetProps) {
  const [chargeType] = useTableStore.use("currentRecord") as ChargeType[];

  if (!chargeType) return null;

  return (
    <Credenza open={open} onOpenChange={onOpenChange}>
      <CredenzaContent>
        <CredenzaHeader>
          <CredenzaTitle className="flex">
            <span>{chargeType.name}</span>
            <Badge className="ml-5" variant="purple">
              {chargeType.id}
            </Badge>
          </CredenzaTitle>
        </CredenzaHeader>
        <CredenzaDescription>
          Last updated on {formatToUserTimezone(chargeType.updatedAt)}
        </CredenzaDescription>
        <ChargeTypeEditForm chargeType={chargeType} />
      </CredenzaContent>
    </Credenza>
  );
}
