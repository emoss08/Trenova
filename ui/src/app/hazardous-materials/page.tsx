import { DataTableLazyComponent } from "@/components/error-boundary";
import { MetaTags } from "@/components/meta-tags";
import { lazy } from "react";

const HazardousMaterialTable = lazy(
  () => import("./_components/hazardous-material-table"),
);

export function HazardousMaterials() {
  return (
    <>
      <MetaTags title="Hazardous Materials" description="Hazardous Materials" />
      <div className="flex flex-col gap-y-3">
        <Header />
        <DataTableLazyComponent>
          <HazardousMaterialTable />
        </DataTableLazyComponent>
      </div>
    </>
  );
}

function Header() {
  return (
    <div className="flex justify-between items-center">
      <div>
        <h1 className="text-3xl font-bold tracking-tight">
          Hazardous Materials
        </h1>
        <p className="text-muted-foreground">
          Manage and configure hazardous materials for your organization
        </p>
      </div>
    </div>
  );
}
