import { DataTableLazyComponent } from "@/components/error-boundary";
import { MetaTags } from "@/components/meta-tags";
import { lazy } from "react";

const GLAccountTable = lazy(() => import("./_components/gl-account-table"));

export function GLAccounts() {
  return (
    <>
      <MetaTags title="GL Accounts" description="GL Accounts" />
      <div className="flex flex-col gap-y-3">
        <Header />
        <DataTableLazyComponent>
          <GLAccountTable />
        </DataTableLazyComponent>
      </div>
    </>
  );
}

function Header() {
  return (
    <div className="flex items-start justify-between">
      <div>
        <h1 className="text-3xl font-bold tracking-tight">GL Accounts</h1>
        <p className="text-muted-foreground">
          Manage and configure GL accounts for your organization
        </p>
      </div>
    </div>
  );
}
