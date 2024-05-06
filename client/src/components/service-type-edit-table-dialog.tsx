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
import React from "react";
import { useForm } from "react-hook-form";
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
  const [isSubmitting, setIsSubmitting] = React.useState<boolean>(false);

  const { control, handleSubmit } = useForm<FormValues>({
    resolver: yupResolver(serviceTypeSchema),
    defaultValues: serviceType,
  });

  const mutation = useCustomMutation<FormValues>(
    control,
    {
      method: "PUT",
      path: `/service-types/${serviceType.id}/`,
      successMessage: "Service Type updated successfully.",
      queryKeysToInvalidate: ["service-type-table-data"],
      closeModal: true,
      errorMessage: "Failed to update service type.",
    },
    () => setIsSubmitting(false),
  );

  const onSubmit = (values: FormValues) => {
    setIsSubmitting(true);
    mutation.mutate(values);
  };

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
          <Button type="submit" isLoading={isSubmitting}>
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
          <CredenzaTitle>{serviceType && serviceType.code}</CredenzaTitle>
        </CredenzaHeader>
        <CredenzaDescription>
          Last updated on&nbsp;
          {serviceType && formatToUserTimezone(serviceType.updatedAt)}
        </CredenzaDescription>
        {serviceType && <ServiceTypeEditForm serviceType={serviceType} />}
      </CredenzaContent>
    </Credenza>
  );
}
