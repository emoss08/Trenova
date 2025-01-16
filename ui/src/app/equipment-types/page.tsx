import { MetaTags } from "@/components/meta-tags";
import EquipmentTypeTable from "./_components/equip-type-table";

export function EquipmentTypes() {
  return (
    <>
      <MetaTags title="Equipment Types" description="Equipment Types" />
      <EquipmentTypeTable />
    </>
  );
}
