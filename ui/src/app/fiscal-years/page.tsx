import { DataTableLazyComponent } from "@/components/error-boundary";
import { MetaTags } from "@/components/meta-tags";
import { lazy } from "react";

const DataTable = lazy(() => import("./_components/fiscal-year-table"));

export function FiscalYears() {
  return (
    <>
      <MetaTags title="Fiscal Years" description="Fiscal Years" />
      <div className="flex flex-col gap-y-3">
        <Header />
        <DataTableLazyComponent>
          <DataTable />
        </DataTableLazyComponent>
      </div>
    </>
  );
}

function Header() {
  return (
    <div className="flex justify-between items-center">
      <div>
        <h1 className="text-3xl font-bold tracking-tight">Fiscal Years</h1>
        <p className="text-muted-foreground">
          Manage and configure fiscal years for your organization
        </p>
      </div>
    </div>
  );
}
