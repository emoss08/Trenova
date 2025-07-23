/**
 * # Copyright 2023-2025 Eric Moss
 * # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md
 */

import { DataTable } from "@/components/data-table/data-table";
import { type EquipmentManufacturerSchema } from "@/lib/schemas/equipment-manufacturer-schema";
import { Resource } from "@/types/audit-entry";
import { useMemo } from "react";
import { getColumns } from "./equip-manufacturer-columns";
import { CreateEquipManufacturerModal } from "./equip-manufacturer-create-modal";
import { EditEquipManufacturerModal } from "./equip-manufacturer-edit-modal";

export default function EquipManufacturerTable() {
  const columns = useMemo(() => getColumns(), []);

  return (
    <DataTable<EquipmentManufacturerSchema>
      resource={Resource.EquipmentManufacturer}
      name="Equipment Type"
      link="/equipment-manufacturers/"
      queryKey="equip-manufacturer-list"
      exportModelName="equipment-manufacturer"
      TableModal={CreateEquipManufacturerModal}
      TableEditModal={EditEquipManufacturerModal}
      columns={columns}
    />
  );
}
