import { MetaTags } from "@/components/meta-tags";
import EquipManufacturerTable from "./_components/equip-manufacturer-table";

export function EquipmentManufacturers() {
  return (
    <>
      <MetaTags
        title="Equipment Manufacturers"
        description="Equipment Manufacturers"
      />
      <EquipManufacturerTable />
    </>
  );
}
