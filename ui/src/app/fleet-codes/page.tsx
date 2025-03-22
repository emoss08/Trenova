import { LazyComponent } from "@/components/error-boundary";
import { MetaTags } from "@/components/meta-tags";
import { lazy } from "react";

const FleetCodesDataTable = lazy(
  () => import("./_components/fleet-code-table"),
);

export function FleetCodes() {
  return (
    <>
      <MetaTags title="Fleet Codes" description="Fleet Codes" />
      <LazyComponent>
        <FleetCodesDataTable />
      </LazyComponent>
    </>
  );
}
