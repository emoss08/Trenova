import { useT } from "@trenova/shared/i18n/use-t";
import { SuspenseLoader } from "@trenova/shared/components/component-loader";
import { AdminPageLayout } from "@/components/navigation/sidebar-layout";
import { PageHeader } from "@/components/page-header";
import { lazy } from "react";

const SequenceConfigForm = lazy(() => import("./_components/sequence-config-form"));

export function SequenceConfigPage() {
  const t = useT();

  return (
    <AdminPageLayout>
      <PageHeader
        title={t("Sequence Configuration")}
        description={t("Configure sequence generation formats for shipments and billing workflows")}
      />
      <div className="p-4">
        <SuspenseLoader>
          <SequenceConfigForm />
        </SuspenseLoader>
      </div>
    </AdminPageLayout>
  );
}
