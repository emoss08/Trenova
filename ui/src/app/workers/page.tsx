import { FormSaveProvider } from "@/components/form";
import { MetaTags } from "@/components/meta-tags";
import { SuspenseLoader } from "@/components/ui/component-loader";
import WorkersDataTable from "./_components/workers-table";

export function Workers() {
  return (
    <>
      <MetaTags title="Workers" description="Workers" />
      <SuspenseLoader>
        <FormSaveProvider>
          <WorkersDataTable />
        </FormSaveProvider>
      </SuspenseLoader>
    </>
  );
}
