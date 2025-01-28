import { MetaTags } from "@/components/meta-tags";
import { SuspenseLoader } from "@/components/ui/component-loader";
import LocationsDataTable from "./_components/location-table";

export function Locations() {
  return (
    <>
      <MetaTags title="Locations" description="Locations" />
      <SuspenseLoader>
        <LocationsDataTable />
      </SuspenseLoader>
    </>
  );
}
