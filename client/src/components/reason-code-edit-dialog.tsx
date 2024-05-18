import { Button } from "@/components/ui/button";
import { useCustomMutation } from "@/hooks/useCustomMutation";
import { formatToUserTimezone } from "@/lib/date";
import { reasonCodeSchema } from "@/lib/validations/ShipmentSchema";
import { useTableStore } from "@/stores/TableStore";
import type {
  ReasonCodeFormValues as FormValues,
  ReasonCode,
} from "@/types/shipment";
import { type TableSheetProps } from "@/types/tables";
import { yupResolver } from "@hookform/resolvers/yup";
import { useForm } from "react-hook-form";
import { ReasonCodeForm } from "./reason-code-table-dialog";
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

function ReasonCodeEditForm({ reasonCode }: { reasonCode: ReasonCode }) {
  const { control, reset, handleSubmit } = useForm<FormValues>({
    resolver: yupResolver(reasonCodeSchema),
    defaultValues: reasonCode,
  });

  const mutation = useCustomMutation<FormValues>(control, {
    method: "PUT",
    path: `/reason-codes/${reasonCode.id}/`,
    successMessage: "Reason Codes updated successfully.",
    queryKeysToInvalidate: "reasonCodes",
    closeModal: true,
    reset,
    errorMessage: "Failed to update reason codes.",
  });

  const onSubmit = (values: FormValues) => {
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
          <Button type="submit" isLoading={mutation.isPending}>
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
          <CredenzaTitle className="flex">
            <span>{reasonCode.code}</span>
            <Badge className="ml-5" variant="purple">
              {reasonCode.id}
            </Badge>
          </CredenzaTitle>
        </CredenzaHeader>
        <CredenzaDescription>
          Last updated on {formatToUserTimezone(reasonCode.updatedAt)}
        </CredenzaDescription>
        <ReasonCodeEditForm reasonCode={reasonCode} />
      </CredenzaContent>
    </Credenza>
  );
}
