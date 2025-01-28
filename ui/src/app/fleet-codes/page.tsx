import { MetaTags } from "@/components/meta-tags";
import FleetCodesDataTable from "./_components/fleet-code-table";
import { SuspenseLoader } from "@/components/ui/component-loader";

export function FleetCodes() {
  return (
    <>
      <MetaTags title="Fleet Codes" description="Fleet Codes" />
      <SuspenseLoader>
        <FleetCodesDataTable />
      </SuspenseLoader>
    </>
  );
}
