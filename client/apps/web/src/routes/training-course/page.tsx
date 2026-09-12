import { useT } from "@trenova/shared/i18n/use-t";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { DataTableLazyComponent } from "@trenova/shared/components/error-boundary";
import { lazy } from "react";

const Table = lazy(() => import("./_components/training-course-table"));

export function TrainingCoursesPage() {
  const t = useT();

  return (
    <PageLayout
      pageHeaderProps={{
        title: t("Training Courses"),
        description: t(
          "The courses workers can be put through — which are required for each driver type, how they are delivered, what passes, and how long a completion stays valid.",
        ),
      }}
    >
      <div className="flex flex-col gap-4">
        <DataTableLazyComponent>
          <Table />
        </DataTableLazyComponent>
      </div>
    </PageLayout>
  );
}
