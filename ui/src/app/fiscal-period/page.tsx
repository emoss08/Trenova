import { DataTableLazyComponent } from "@/components/error-boundary";
import { MetaTags } from "@/components/meta-tags";
import { lazy } from "react";

const DataTable = lazy(() => import("./_components/fiscal-period-table"));

export function FiscalPeriods() {
  return (
    <>
      <MetaTags title="Fiscal Periods" description="Fiscal Periods" />
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
    <div className="flex items-center justify-between">
      <div>
        <h1 className="text-3xl font-bold tracking-tight">Fiscal Periods</h1>
        <p className="text-muted-foreground">
          Manage and configure fiscal periods for your organization
        </p>
      </div>
    </div>
  );
}
