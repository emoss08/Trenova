import { LazyComponent } from "@/components/error-boundary";
import { MetaTags } from "@/components/meta-tags";
import { lazy } from "react";

const LocationCategoryTable = lazy(
  () => import("./_components/location-category-table"),
);

export function LocationCategories() {
  return (
    <>
      <MetaTags title="Location Categories" description="Location Categories" />
      <LazyComponent>
        <LocationCategoryTable />
      </LazyComponent>
    </>
  );
}
