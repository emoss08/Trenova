import { LazyComponent } from "@/components/error-boundary";
import { MetaTags } from "@/components/meta-tags";
import { lazy } from "react";

const EquipManufacturerTable = lazy(
  () => import("./_components/equip-manufacturer-table"),
);

export function EquipmentManufacturers() {
  return (
    <>
      <MetaTags
        title="Equipment Manufacturers"
        description="Equipment Manufacturers"
      />
      <LazyComponent>
        <EquipManufacturerTable />
      </LazyComponent>
    </>
  );
}
