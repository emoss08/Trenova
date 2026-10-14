import { useT } from "@trenova/shared/i18n/use-t";
import { LazyComponent } from "@trenova/shared/components/error-boundary";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { lazy } from "react";
import { DataEntryControlSkeleton } from "./skeleton";

const DataEntryControlForm = lazy(() => import("./_components/data-entry-control-form"));

export function DataEntryControlPage() {
  const t = useT();

  return (
    <PageLayout
      pageHeaderProps={{
        title: t("Data entry control"),
        description: t("Configure case formatting rules for data entry across the system"),
      }}
    >
      <LazyComponent fallback={<DataEntryControlSkeleton />}>
        <DataEntryControlForm />
      </LazyComponent>
    </PageLayout>
  );
}
