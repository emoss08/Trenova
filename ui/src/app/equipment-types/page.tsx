import { DataTableLazyComponent } from "@/components/error-boundary";
import { MetaTags } from "@/components/meta-tags";
import { lazy } from "react";

const EquipmentTypeTable = lazy(() => import("./_components/equip-type-table"));

export function EquipmentTypes() {
  return (
    <>
      <MetaTags title="Equipment Types" description="Equipment Types" />
      <DataTableLazyComponent>
        <EquipmentTypeTable />
      </DataTableLazyComponent>
    </>
  );
}
