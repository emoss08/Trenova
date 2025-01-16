import { MetaTags } from "@/components/meta-tags";
import FleetCodesDataTable from "./_components/fleet-code-table";

export function FleetCodes() {
  return (
    <>
      <MetaTags title="Fleet Codes" description="Fleet Codes" />
      <FleetCodesDataTable />
    </>
  );
}
