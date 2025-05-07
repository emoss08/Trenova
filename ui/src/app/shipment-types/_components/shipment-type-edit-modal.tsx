import { FormEditModal } from "@/components/ui/form-edit-modal";
import {
  shipmentTypeSchema,
  ShipmentTypeSchema,
} from "@/lib/schemas/shipment-type-schema";
import { type EditTableSheetProps } from "@/types/data-table";
import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";
import { ShipmentTypeForm } from "./shipment-type-form";

export function EditShipmentTypeModal({
  currentRecord,
}: EditTableSheetProps<ShipmentTypeSchema>) {
  const form = useForm({
    resolver: zodResolver(shipmentTypeSchema),
    defaultValues: currentRecord,
  });

  return (
    <FormEditModal
      currentRecord={currentRecord}
      url="/shipment-types/"
      title="Shipment Type"
      queryKey="shipment-type-list"
      formComponent={<ShipmentTypeForm />}
      fieldKey="code"
      form={form}
    />
  );
}
