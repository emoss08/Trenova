/**
 * # Copyright 2023-2025 Eric Moss
 * # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md
 */

import { DataTable } from "@/components/data-table/data-table";
import { type ShipmentTypeSchema } from "@/lib/schemas/shipment-type-schema";
import { Resource } from "@/types/audit-entry";
import { useMemo } from "react";
import { getColumns } from "./shipment-type-columns";
import { CreateShipmentTypeModal } from "./shipment-type-create-modal";
import { EditShipmentTypeModal } from "./shipment-type-edit-modal";

export default function ShipmentTypesDataTable() {
  const columns = useMemo(() => getColumns(), []);

  return (
    <DataTable<ShipmentTypeSchema>
      resource={Resource.ShipmentType}
      name="Shipment Type"
      link="/shipment-types/"
      queryKey="shipment-type-list"
      exportModelName="shipment-type"
      TableModal={CreateShipmentTypeModal}
      TableEditModal={EditShipmentTypeModal}
      columns={columns}
    />
  );
}
