import { FormCreateModal } from "@/components/ui/form-create-modal";
import { shipmentTypeSchema } from "@/lib/schemas/shipment-type-schema";
import { Status } from "@/types/common";
import { type TableSheetProps } from "@/types/data-table";
import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";
import { ShipmentTypeForm } from "./shipment-type-form";

export function CreateShipmentTypeModal({
  open,
  onOpenChange,
}: TableSheetProps) {
  const form = useForm({
    resolver: zodResolver(shipmentTypeSchema),
    defaultValues: {
      code: "",
      status: Status.Active,
      description: "",
      color: "",
    },
  });

  return (
    <FormCreateModal
      open={open}
      onOpenChange={onOpenChange}
      title="Shipment Type"
      formComponent={<ShipmentTypeForm />}
      form={form}
      url="/shipment-types/"
      queryKey="shipment-type-list"
      className="max-w-[400px]"
    />
  );
}
