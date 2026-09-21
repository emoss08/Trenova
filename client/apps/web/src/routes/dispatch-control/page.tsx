import { useT } from "@trenova/shared/i18n/use-t";
import { SuspenseLoader } from "@trenova/shared/components/component-loader";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { lazy } from "react";

const DispatchControlForm = lazy(() => import("./_components/dispatch-control-form"));

export function DispatchControlPage() {
  const t = useT();

  return (
    <PageLayout
      pageHeaderProps={{
        title: t("Dispatch Control"),
        description: t("Configure and manage your dispatch control settings"),
      }}
    >
      <SuspenseLoader>
        <div className="p-4">
          <DispatchControlForm />
        </div>
      </SuspenseLoader>
    </PageLayout>
  );
}
