import { QueryLazyComponent } from "@/components/error-boundary";
import { MetaTags } from "@/components/meta-tags";
import { queries } from "@/lib/queries";
import { lazy, memo } from "react";

const AccountingControlForm = lazy(
  () => import("./_components/accounting-control-form"),
);

export function AccountingControl() {
  return (
    <div className="flex flex-col space-y-6">
      <MetaTags title="Accounting Control" description="Accounting Control" />
      <Header />
      <QueryLazyComponent
        queryKey={queries.organization.getAccountingControl._def}
      >
        <AccountingControlForm />
      </QueryLazyComponent>
    </div>
  );
}

const Header = memo(() => {
  return (
    <div className="flex justify-between items-center">
      <div>
        <h1 className="text-3xl font-bold tracking-tight">
          Accounting Control
        </h1>
        <p className="text-muted-foreground">
          Configure and manage your accounting control settings
        </p>
      </div>
    </div>
  );
});
Header.displayName = "Header";
