import { DataTableLazyComponent } from "@/components/error-boundary";
import { MetaTags } from "@/components/meta-tags";
import { lazy } from "react";

const DistanceOverrideTable = lazy(
  () => import("./_components/distance-override-table"),
);

export function DistanceOverrides() {
  return (
    <DistanceOverrideInner>
      <MetaTags title="Distance Overrides" description="Distance Overrides" />
      <Header />
      <DataTableLazyComponent>
        <DistanceOverrideTable />
      </DataTableLazyComponent>
    </DistanceOverrideInner>
  );
}

function DistanceOverrideInner({ children }: { children: React.ReactNode }) {
  return <div className="flex flex-col gap-y-3">{children}</div>;
}

function Header() {
  return (
    <div className="flex justify-between items-center">
      <div>
        <h1 className="text-3xl font-bold tracking-tight">
          Distance Overrides
        </h1>
        <p className="text-muted-foreground">
          Manage and configure distance overrides for your organization
        </p>
      </div>
    </div>
  );
}
