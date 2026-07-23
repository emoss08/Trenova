import { SuspenseLoader } from "@/components/component-loader";
import { AdminPageLayout } from "@/components/navigation/sidebar-layout";
import { PageHeader } from "@/components/page-header";
import { lazy } from "react";

const SequenceConfigForm = lazy(() => import("./_components/sequence-config-form"));

export function SequenceConfigPage() {
  return (
    <AdminPageLayout>
      <PageHeader
        title="Sequence Configuration"
        description="Configure sequence generation formats for shipments and billing workflows"
      />
      <div className="p-4">
        <SuspenseLoader>
          <SequenceConfigForm />
        </SuspenseLoader>
      </div>
    </AdminPageLayout>
  );
}
