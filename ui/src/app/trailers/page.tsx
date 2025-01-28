import { MetaTags } from "@/components/meta-tags";
import { SuspenseLoader } from "@/components/ui/component-loader";
import TrailerTable from "./_components/trailer-table";

export function Trailers() {
  return (
    <>
      <MetaTags title="Trailers" description="Trailers" />
      <SuspenseLoader>
        <TrailerTable />
      </SuspenseLoader>
    </>
  );
}
