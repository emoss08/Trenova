import { useT } from "@trenova/shared/i18n/use-t";
import { SuspenseLoader } from "@trenova/shared/components/component-loader";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { lazy } from "react";

const DataEntryControlForm = lazy(() => import("./_components/data-entry-control-form"));

export function DataEntryControlPage() {
  const t = useT();

  return (
    <PageLayout
      pageHeaderProps={{
        title: t("Data Entry Control"),
        description: t("Configure case formatting rules for data entry across the system"),
      }}
    >
      <SuspenseLoader>
        <DataEntryControlForm />
      </SuspenseLoader>
    </PageLayout>
  );
}
