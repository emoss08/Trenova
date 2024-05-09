import { ServiceTypeForm } from "@/components/service-type-table-dialog";
import { Button } from "@/components/ui/button";
import { useCustomMutation } from "@/hooks/useCustomMutation";
import { formatToUserTimezone } from "@/lib/date";
import { serviceTypeSchema } from "@/lib/validations/ShipmentSchema";
import { useTableStore } from "@/stores/TableStore";
import type {
  ServiceTypeFormValues as FormValues,
  ServiceType,
} from "@/types/shipment";
import { type TableSheetProps } from "@/types/tables";
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

function ServiceTypeEditForm({ serviceType }: { serviceType: ServiceType }) {
  const { control, handleSubmit } = useForm<FormValues>({
    resolver: yupResolver(serviceTypeSchema),
    defaultValues: serviceType,
  });

  const mutation = useCustomMutation<FormValues>(control, {
    method: "PUT",
    path: `/service-types/${serviceType.id}/`,
    successMessage: "Service Type updated successfully.",
    queryKeysToInvalidate: "serviceTypes",
    closeModal: true,
    errorMessage: "Failed to update service type.",
  });

  const onSubmit = (values: FormValues) => mutation.mutate(values);

  return (
    <CredenzaBody>
      <form onSubmit={handleSubmit(onSubmit)}>
        <ServiceTypeForm control={control} />
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

export function ServiceTypeEditDialog({ onOpenChange, open }: TableSheetProps) {
  const [serviceType] = useTableStore.use("currentRecord") as ServiceType[];

  if (!serviceType) return null;

  return (
    <Credenza open={open} onOpenChange={onOpenChange}>
      <CredenzaContent>
        <CredenzaHeader>
          <CredenzaTitle className="flex">
            <span>{serviceType.code}</span>
            <Badge className="ml-5" variant="purple">
              {serviceType.id}
            </Badge>
          </CredenzaTitle>
        </CredenzaHeader>
        <CredenzaDescription>
          Last updated on {formatToUserTimezone(serviceType.updatedAt)}
        </CredenzaDescription>
        <ServiceTypeEditForm serviceType={serviceType} />
      </CredenzaContent>
    </Credenza>
  );
}
