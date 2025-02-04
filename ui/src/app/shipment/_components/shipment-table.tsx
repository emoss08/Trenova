import { DataTable } from "@/components/data-table/data-table";
import { Shipment } from "@/types/shipment";
import { useMemo } from "react";
import { getColumns } from "./shipment-columns";

export default function ShipmentTable() {
  const columns = useMemo(() => getColumns(), []);

  return (
    <DataTable<Shipment>
      name="Shipment"
      link="/shipments"
      extraSearchParams={{
        includeMoveDetails: true,
        includeStopDetails: true,
        includeCustomerDetails: true,
      }}
      queryKey="shipment-list"
      exportModelName="shipment"
      //   TableModal={CreateTractorModal}
      //   TableEditModal={EditTractorModal}
      columns={columns}
    />
  );
}
