import { DataTableLazyComponent } from "@/components/error-boundary";
import { PageHeader } from "@/components/page-header";
import { lazy } from "react";

const Table = lazy(() => import("./_components/service-failure-table"));

export function ServiceFailuresPage() {
  return (
    <div className="flex h-full flex-col">
      <PageHeader
        title="Service Failures"
        description="Review unresolved pickup and delivery service failures"
      />
      <div className="p-4">
        <DataTableLazyComponent>
          <Table />
        </DataTableLazyComponent>
      </div>
    </div>
  );
}
