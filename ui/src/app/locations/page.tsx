import { DataTableLazyComponent } from "@/components/error-boundary";
import { MetaTags } from "@/components/meta-tags";
import { lazy } from "react";

const LocationTable = lazy(() => import("./_components/location-table"));

export function Locations() {
  return (
    <>
      <MetaTags title="Locations" description="Locations" />
      <DataTableLazyComponent>
        <LocationTable />
      </DataTableLazyComponent>
    </>
  );
}
