import { LazyComponent } from "@/components/error-boundary";
import { MetaTags } from "@/components/meta-tags";
import { lazy } from "react";

const TractorTable = lazy(() => import("./_components/tractor-table"));

export function Tractor() {
  return (
    <>
      <MetaTags title="Tractors" description="Tractors" />
      <LazyComponent>
        <TractorTable />
      </LazyComponent>
    </>
  );
}
