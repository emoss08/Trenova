import { DataTable } from "@/components/data-table/data-table";
import { Shipment } from "@/types/shipment";
import { useMemo } from "react";
import { getColumns } from "./shipment-columns";
import { ShipmentCreateSheet } from "./shipment-create-sheet";
import { ShipmentEditSheet } from "./shipment-edit-sheet";

export default function ShipmentTable() {
  const columns = useMemo(() => getColumns(), []);

  return (
    <DataTable<Shipment>
      name="Shipment"
      link="/shipments/"
      extraSearchParams={{
        expandShipmentDetails: true,
      }}
      queryKey="shipment-list"
      exportModelName="shipment"
      TableModal={ShipmentCreateSheet}
      TableEditModal={ShipmentEditSheet}
      columns={columns}
    />
  );
}
