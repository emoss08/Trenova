import { DataTable } from "@/components/data-table/data-table";
import { type ShipmentTypeSchema } from "@/lib/schemas/shipment-type-schema";
import { useMemo } from "react";
import { getColumns } from "./shipment-type-columns";
import { EditShipmentTypeModal } from "./shipment-type-edit-modal";
import { CreateShipmentTypeModal } from "./shipment-type-create-modal";

export default function ShipmentTypesDataTable() {
  const columns = useMemo(() => getColumns(), []);

  return (
    <DataTable<ShipmentTypeSchema>
      name="Shipment Type"
      link="/shipment-types/"
      queryKey="shipment-type-list"
      TableModal={CreateShipmentTypeModal}
      TableEditModal={EditShipmentTypeModal}
      columns={columns}
    />
  );
}
