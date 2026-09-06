import { PageLayout } from "@/components/navigation/sidebar-layout";
import { DataTableLazyComponent } from "@trenova/shared/components/error-boundary";
import { lazy } from "react";

const Table = lazy(() => import("./_components/review-template-table"));

export function ReviewTemplatesPage() {
  return (
    <PageLayout
      pageHeaderProps={{
        title: "Review Templates",
        description:
          "What a performance review rates and how much each item counts. Items are copied onto every review, so editing a template never rewrites history.",
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
