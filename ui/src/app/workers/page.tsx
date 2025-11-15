import { DataTableLazyComponent } from "@/components/error-boundary";
import { FormSaveProvider } from "@/components/form";
import { MetaTags } from "@/components/meta-tags";
import { lazy, memo } from "react";
import { PTOContent } from "./_components/pto/pto-content";

const WorkersDataTable = lazy(() => import("./_components/workers-table"));

export function Workers() {
  return (
    <>
      <MetaTags title="Workers" description="Workers" />
      <FormSaveProvider>
        <div className="flex flex-col gap-4">
          <Header />
          <WorkersContent>
            <PTOContent />
            <DataTableLazyComponent>
              <WorkersDataTable />
            </DataTableLazyComponent>
          </WorkersContent>
        </div>
      </FormSaveProvider>
    </>
  );
}

function WorkersContent({ children }: { children: React.ReactNode }) {
  return <div className="flex flex-col gap-4">{children}</div>;
}

const Header = memo(() => {
  return (
    <div className="flex items-center justify-between">
      <div>
        <h1 className="text-3xl font-bold tracking-tight">Workers</h1>
        <p className="text-muted-foreground">
          Manage and track all worker along with their paid time off records
        </p>
      </div>
    </div>
  );
});
Header.displayName = "Header";
