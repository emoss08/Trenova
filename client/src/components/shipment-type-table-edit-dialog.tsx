import { Button } from "@/components/ui/button";
import { useCustomMutation } from "@/hooks/useCustomMutation";
import { formatToUserTimezone } from "@/lib/date";
import { shipmentTypeSchema } from "@/lib/validations/ShipmentSchema";
import { useTableStore } from "@/stores/TableStore";
import type {
  ShipmentTypeFormValues as FormValues,
  ShipmentType,
} from "@/types/shipment";
import { type TableSheetProps } from "@/types/tables";
import { yupResolver } from "@hookform/resolvers/yup";
import React from "react";
import { useForm } from "react-hook-form";
import { ShipmentTypeForm } from "./shipment-type-table-dialog";
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

function ShipmentTypeEditForm({
  shipmentType,
}: {
  shipmentType: ShipmentType;
}) {
  const [isSubmitting, setIsSubmitting] = React.useState<boolean>(false);

  const { control, handleSubmit } = useForm<FormValues>({
    resolver: yupResolver(shipmentTypeSchema),
    defaultValues: shipmentType,
  });

  const mutation = useCustomMutation<FormValues>(
    control,
    {
      method: "PUT",
      path: `/shipment-types/${shipmentType.id}/`,
      successMessage: "Shipment Type updated successfully.",
      queryKeysToInvalidate: ["shipment-type-table-data"],
      closeModal: true,
      errorMessage: "Failed to update shipment type.",
    },
    () => setIsSubmitting(false),
  );

  const onSubmit = (values: FormValues) => {
    setIsSubmitting(true);
    mutation.mutate(values);
  };

  return (
    <CredenzaBody>
      <form
        onSubmit={handleSubmit(onSubmit)}
        className="flex h-full flex-col overflow-y-auto"
      >
        <ShipmentTypeForm control={control} />
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

export function ShipmentTypeEditDialog({
  onOpenChange,
  open,
}: TableSheetProps) {
  const [shipmentType] = useTableStore.use("currentRecord") as ShipmentType[];

  if (!shipmentType) return null;

  return (
    <Credenza open={open} onOpenChange={onOpenChange}>
      <CredenzaContent>
        <CredenzaHeader>
          <CredenzaTitle>{shipmentType && shipmentType.code} </CredenzaTitle>
        </CredenzaHeader>
        <CredenzaDescription>
          Last updated on&nbsp;
          {shipmentType && formatToUserTimezone(shipmentType.updatedAt)}
        </CredenzaDescription>
        {shipmentType && <ShipmentTypeEditForm shipmentType={shipmentType} />}
      </CredenzaContent>
    </Credenza>
  );
}
