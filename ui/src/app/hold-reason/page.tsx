import { LazyComponent } from "@/components/error-boundary";
import { MetaTags } from "@/components/meta-tags";
import { lazy } from "react";

const HoldReasonTable = lazy(() => import("./_components/hold-reason-table"));

export function HoldReasons() {
  return (
    <>
      <MetaTags title="Hold Reasons" description="Hold Reasons" />
      <Header />
      <LazyComponent>
        <HoldReasonTable />
      </LazyComponent>
    </>
  );
}

function Header() {
  return (
    <div className="flex justify-between items-center">
      <div>
        <h1 className="text-3xl font-bold tracking-tight">Hold Reasons</h1>
        <p className="text-muted-foreground">
          Manage and configure hold reasons for your organization
        </p>
      </div>
    </div>
  );
}
