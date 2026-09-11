import { useT } from "@trenova/shared/i18n/use-t";
import { SuspenseLoader } from "@trenova/shared/components/component-loader";
import { AdminPageLayout } from "@/components/navigation/sidebar-layout";
import { PageHeader } from "@/components/page-header";
import { lazy } from "react";

const DashControlForm = lazy(() => import("./_components/dash-control-form"));

export function DashControlPage() {
  const t = useT();

  return (
    <AdminPageLayout>
      <PageHeader
        title={t("Dash Control")}
        description={t("Choose what drivers can see and do in the Dash driver portal")}
      />
      <SuspenseLoader>
        <div className="p-4">
          <DashControlForm />
        </div>
      </SuspenseLoader>
    </AdminPageLayout>
  );
}
