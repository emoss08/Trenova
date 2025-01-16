import { DataTable } from "@/components/data-table/data-table";
import { type ShipmentTypeSchema } from "@/lib/schemas/shipment-type-schema";
import { useMemo } from "react";
import { getColumns } from "./service-type-columns";
import { CreateServiceTypeModal } from "./service-type-create-modal";
import { EditServiceTypeModal } from "./service-type-edit-modal";

export default function ServiceTypesDataTable() {
  const columns = useMemo(() => getColumns(), []);

  return (
    <DataTable<ShipmentTypeSchema>
      name="Service Type"
      link="/service-types/"
      queryKey="service-type-list"
      TableModal={CreateServiceTypeModal}
      TableEditModal={EditServiceTypeModal}
      columns={columns}
    />
  );
}
