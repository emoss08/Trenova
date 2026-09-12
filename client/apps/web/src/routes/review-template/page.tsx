import { useT } from "@trenova/shared/i18n/use-t";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { DataTableLazyComponent } from "@trenova/shared/components/error-boundary";
import { lazy } from "react";

const Table = lazy(() => import("./_components/review-template-table"));

export function ReviewTemplatesPage() {
  const t = useT();

  return (
    <PageLayout
      pageHeaderProps={{
        title: t("Review Templates"),
        description: t(
          "What a performance review rates and how much each item counts. Items are copied onto every review, so editing a template never rewrites history.",
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
