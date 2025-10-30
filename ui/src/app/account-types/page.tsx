import { DataTableLazyComponent } from "@/components/error-boundary";
import { MetaTags } from "@/components/meta-tags";
import { lazy } from "react";

const DataTable = lazy(() => import("./_components/account-type-table"));

export function AccountTypes() {
  return (
    <>
      <MetaTags title="Account Types" description="Account Types" />
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
        <h1 className="text-3xl font-bold tracking-tight">Account Types</h1>
        <p className="text-muted-foreground">
          Manage and configure account types for your organization
        </p>
      </div>
    </div>
  );
}
