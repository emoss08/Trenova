import { MetaTags } from "@/components/meta-tags";
import WorkersDataTable from "./_components/workers-table";

export function Workers() {
  return (
    <>
      <MetaTags title="Workers" description="Workers" />
      <WorkersDataTable />
    </>
  );
}
