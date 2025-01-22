import { MetaTags } from "@/components/meta-tags";
import LocationCategoryTable from "./_components/location-category-table";

export function LocationCategories() {
  return (
    <>
      <MetaTags title="Location Categories" description="Location Categories" />
      <LocationCategoryTable />
    </>
  );
}
