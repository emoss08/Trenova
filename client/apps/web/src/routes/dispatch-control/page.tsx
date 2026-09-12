import { useT } from "@trenova/shared/i18n/use-t";
import { SuspenseLoader } from "@trenova/shared/components/component-loader";
import { AdminPageLayout } from "@/components/navigation/sidebar-layout";
import { PageHeader } from "@/components/page-header";
import { lazy } from "react";

const DispatchControlForm = lazy(() => import("./_components/dispatch-control-form"));

export function DispatchControlPage() {
  const t = useT();

  return (
    <AdminPageLayout>
      <PageHeader
        title={t("Dispatch Control")}
        description={t("Configure and manage your dispatch control settings")}
      />
      <SuspenseLoader>
        <div className="p-4">
          <DispatchControlForm />
        </div>
      </SuspenseLoader>
    </AdminPageLayout>
  );
}
