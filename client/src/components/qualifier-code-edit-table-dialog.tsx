import { Button } from "@/components/ui/button";
import { useCustomMutation } from "@/hooks/useCustomMutation";
import { formatToUserTimezone } from "@/lib/date";
import { qualifierCodeSchema } from "@/lib/validations/StopSchema";
import { useTableStore } from "@/stores/TableStore";
import type {
  QualifierCodeFormValues as FormValues,
  QualifierCode,
} from "@/types/stop";
import { type TableSheetProps } from "@/types/tables";
import { yupResolver } from "@hookform/resolvers/yup";
import { useForm } from "react-hook-form";
import { QualifierCodeForm } from "./qualifier-code-table-dialog";
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

function QualifierCodeEditForm({
  qualifierCode,
}: {
  qualifierCode: QualifierCode;
}) {
  const { control, reset, handleSubmit } = useForm<FormValues>({
    resolver: yupResolver(qualifierCodeSchema),
    defaultValues: qualifierCode,
  });

  const mutation = useCustomMutation<FormValues>(control, {
    method: "PUT",
    path: `/qualifier-codes/${qualifierCode.id}/`,
    successMessage: "Qualifier Code updated successfully.",
    queryKeysToInvalidate: "qualifierCodes",
    closeModal: true,
    reset,
    errorMessage: "Failed to update qualifier code",
  });

  const onSubmit = (values: FormValues) => mutation.mutate(values);

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
          <Button type="submit" isLoading={mutation.isPending}>
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
          <CredenzaTitle className="flex">
            <span>{qualifierCode.code}</span>
            <Badge className="ml-5" variant="purple">
              {qualifierCode.id}
            </Badge>
          </CredenzaTitle>
        </CredenzaHeader>
        <CredenzaDescription>
          Last updated on {formatToUserTimezone(qualifierCode.updatedAt)}
        </CredenzaDescription>
        <QualifierCodeEditForm qualifierCode={qualifierCode} />
      </CredenzaContent>
    </Credenza>
  );
}
