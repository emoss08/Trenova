import { LazyComponent } from "@/components/error-boundary";
import { FormSaveProvider } from "@/components/form";
import { MetaTags } from "@/components/meta-tags";
import { lazy } from "react";

const WorkersDataTable = lazy(() => import("./_components/workers-table"));

export function Workers() {
  return (
    <>
      <MetaTags title="Workers" description="Workers" />
      <FormSaveProvider>
        <LazyComponent>
          <WorkersDataTable />
        </LazyComponent>
      </FormSaveProvider>
    </>
  );
}
