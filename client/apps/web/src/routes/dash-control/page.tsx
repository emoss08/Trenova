import { SuspenseLoader } from "@/components/component-loader";
import { AdminPageLayout } from "@/components/navigation/sidebar-layout";
import { PageHeader } from "@/components/page-header";
import { lazy } from "react";

const DashControlForm = lazy(() => import("./_components/dash-control-form"));

export function DashControlPage() {
  return (
    <AdminPageLayout>
      <PageHeader
        title="Dash Control"
        description="Choose what drivers can see and do in the Dash driver portal"
      />
      <SuspenseLoader>
        <div className="p-4">
          <DashControlForm />
        </div>
      </SuspenseLoader>
    </AdminPageLayout>
  );
}
