import { SuspenseLoader } from "@/components/component-loader";
import { AdminPageLayout } from "@/components/navigation/sidebar-layout";
import { PageHeader } from "@/components/page-header";
import { lazy } from "react";

const DispatchControlForm = lazy(() => import("./_components/dispatch-control-form"));

export function DispatchControlPage() {
  return (
    <AdminPageLayout>
      <PageHeader
        title="Dispatch Control"
        description="Configure and manage your dispatch control settings"
      />
      <SuspenseLoader>
        <div className="p-4">
          <DispatchControlForm />
        </div>
      </SuspenseLoader>
    </AdminPageLayout>
  );
}
