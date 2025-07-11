import { DataTable } from "@/components/data-table/data-table";
import { type EquipmentTypeSchema } from "@/lib/schemas/equipment-type-schema";
import { Resource } from "@/types/audit-entry";
import { useMemo } from "react";
import { getColumns } from "./equip-type-columns";
import { CreateEquipTypeModal } from "./equip-type-create-modal";
import { EditEquipTypeModal } from "./equip-type-edit-modal";

export default function EquipmentTypeTable() {
  const columns = useMemo(() => getColumns(), []);

  return (
    <DataTable<EquipmentTypeSchema>
      resource={Resource.EquipmentType}
      name="Equipment Type"
      link="/equipment-types/"
      queryKey="equipment-type-list"
      exportModelName="equipment-type"
      TableModal={CreateEquipTypeModal}
      TableEditModal={EditEquipTypeModal}
      columns={columns}
    />
  );
}
