import { useCustomMutation } from "@/hooks/useCustomMutation";
import { formatToUserTimezone } from "@/lib/date";
import { fleetCodeSchema } from "@/lib/validations/DispatchSchema";
import { useTableStore } from "@/stores/TableStore";
import type {
  FleetCode,
  FleetCodeFormValues as FormValues,
} from "@/types/dispatch";
import { yupResolver } from "@hookform/resolvers/yup";
import { useForm } from "react-hook-form";
import { FleetCodeForm } from "./fleet-code-table-dialog";
import { Badge } from "./ui/badge";
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
  const { control, handleSubmit } = useForm<FormValues>({
    resolver: yupResolver(fleetCodeSchema),
    defaultValues: fleetCode,
  });

  const mutation = useCustomMutation<FormValues>(control, {
    method: "PUT",
    path: `/fleet-codes/${fleetCode.id}/`,
    successMessage: "Fleet Code updated successfully.",
    queryKeysToInvalidate: ["fleet-code-table-data"],
    closeModal: true,
    errorMessage: "Failed to update fleet code.",
  });

  const onSubmit = (values: FormValues) => {
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
          <Button type="submit" isLoading={mutation.isPending}>
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
          <CredenzaTitle className="flex">
            <span>{fleetCode.code}</span>
            <Badge className="ml-5" variant="purple">
              {fleetCode.id}
            </Badge>
          </CredenzaTitle>
        </CredenzaHeader>
        <CredenzaDescription>
          Last updated on {formatToUserTimezone(fleetCode.updatedAt)}
        </CredenzaDescription>
        <FleetCodeEditForm fleetCode={fleetCode} open={open} />
      </CredenzaContent>
    </Credenza>
  );
}
