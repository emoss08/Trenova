import { MetaTags } from "@/components/meta-tags";
import { SuspenseLoader } from "@/components/ui/component-loader";
import LocationCategoryTable from "./_components/location-category-table";

export function LocationCategories() {
  return (
    <>
      <MetaTags title="Location Categories" description="Location Categories" />
      <SuspenseLoader>
        <LocationCategoryTable />
      </SuspenseLoader>
    </>
  );
}
