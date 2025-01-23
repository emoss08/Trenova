import { MetaTags } from "@/components/meta-tags";
import LocationsDataTable from "./_components/location-table";

export function Locations() {
  return (
    <>
      <MetaTags title="Locations" description="Locations" />
      <LocationsDataTable />
    </>
  );
}
