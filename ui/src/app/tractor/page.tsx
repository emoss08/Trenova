import { MetaTags } from "@/components/meta-tags";
import TractorTable from "./_components/tractor-table";

export function Tractor() {
  return (
    <>
      <MetaTags title="Tractors" description="Tractors" />
      <TractorTable />
    </>
  );
}
