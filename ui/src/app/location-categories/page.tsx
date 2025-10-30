import { DataTableLazyComponent } from "@/components/error-boundary";
import { MetaTags } from "@/components/meta-tags";
import { lazy } from "react";

const LocationCategoryTable = lazy(
  () => import("./_components/location-category-table"),
);

export function LocationCategories() {
  return (
    <>
      <MetaTags title="Location Categories" description="Location Categories" />
      <div className="flex flex-col gap-y-3">
        <Header />
        <DataTableLazyComponent>
          <LocationCategoryTable />
        </DataTableLazyComponent>
      </div>
    </>
  );
}

function Header() {
  return (
    <div className="flex justify-between items-start">
      <div>
        <h1 className="text-3xl font-bold tracking-tight">
          Location Categories
        </h1>
        <p className="text-muted-foreground">
          Manage and configure location categories for your organization
        </p>
      </div>
    </div>
  );
}
