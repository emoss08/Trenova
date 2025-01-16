import { DataTable } from "@/components/data-table/data-table";
import { type EquipmentManufacturerSchema } from "@/lib/schemas/equipment-manufacturer-schema";
import { useMemo } from "react";
import { getColumns } from "./equip-manufacturer-columns";
import { CreateEquipManufacturerModal } from "./equip-manufacturer-create-modal";
import { EditEquipManufacturerModal } from "./equip-manufacturer-edit-modal";

export default function EquipManufacturerTable() {
  const columns = useMemo(() => getColumns(), []);

  return (
    <DataTable<EquipmentManufacturerSchema>
      name="Equipment Type"
      link="/equipment-manufacturers/"
      queryKey="equip-manufacturer-list"
      TableModal={CreateEquipManufacturerModal}
      TableEditModal={EditEquipManufacturerModal}
      columns={columns}
    />
  );
}
