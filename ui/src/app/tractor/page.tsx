import { MetaTags } from "@/components/meta-tags";
import { SuspenseLoader } from "@/components/ui/component-loader";
import TractorTable from "./_components/tractor-table";

export function Tractor() {
  return (
    <>
      <MetaTags title="Tractors" description="Tractors" />
      <SuspenseLoader>
        <TractorTable />
      </SuspenseLoader>
    </>
  );
}
