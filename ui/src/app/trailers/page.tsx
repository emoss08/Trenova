import { LazyComponent } from "@/components/error-boundary";
import { MetaTags } from "@/components/meta-tags";
import { lazy } from "react";

const TrailerTable = lazy(() => import("./_components/trailer-table"));

export function Trailers() {
  return (
    <>
      <MetaTags title="Trailers" description="Trailers" />
      <LazyComponent>
        <TrailerTable />
      </LazyComponent>
    </>
  );
}
