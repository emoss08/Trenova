import { SuspenseLoader } from "@trenova/shared/components/component-loader";
import { AdminPageLayout } from "@/components/navigation/sidebar-layout";
import { PageHeader } from "@/components/page-header";
import { lazy } from "react";

const AgentControlForm = lazy(() => import("./_components/agent-control-form"));

export function AgentControlPage() {
  return (
    <AdminPageLayout>
      <PageHeader
        title="Agent Control"
        description="Configure the billing exception agent — enablement, shadow mode, and how long proposals wait for a human decision"
      />
      <SuspenseLoader>
        <div className="p-4">
          <AgentControlForm />
        </div>
      </SuspenseLoader>
    </AdminPageLayout>
  );
}
