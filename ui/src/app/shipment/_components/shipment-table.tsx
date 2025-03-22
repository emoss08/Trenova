import { DataTable } from "@/components/data-table/data-table";
import { BetaTag } from "@/components/ui/beta-tag";
import { Shipment } from "@/types/shipment";
import { faFileImport } from "@fortawesome/pro-regular-svg-icons";
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
      extraActions={[
        {
          key: "import-from-rate",
          label: "Import from Rate Conf.",
          description: "Import shipment from rate confirmation",
          icon: faFileImport,
          onClick: () => {
            console.log("Import from Rate Conf.");
          },
          endContent: <BetaTag label="Preview" />,
        },
      ]}
    />
  );
}
