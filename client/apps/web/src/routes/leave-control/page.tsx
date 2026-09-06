import { AdminPageLayout } from "@/components/navigation/sidebar-layout";
import { PageHeader } from "@/components/page-header";
import { SuspenseLoader } from "@trenova/shared/components/component-loader";
import { lazy } from "react";

const LeaveControlForm = lazy(() => import("./_components/leave-control-form"));

export function LeaveControlPage() {
  return (
    <AdminPageLayout>
      <PageHeader
        title="Leave Settings"
        description="How family and medical leave is measured, and what an employee must do to qualify"
      />
      <SuspenseLoader>
        <div className="p-4">
          <LeaveControlForm />
        </div>
      </SuspenseLoader>
    </AdminPageLayout>
  );
}
