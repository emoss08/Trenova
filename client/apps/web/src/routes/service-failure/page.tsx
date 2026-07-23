import { DataTableLazyComponent } from "@trenova/shared/components/error-boundary";
import { PageHeader } from "@/components/page-header";
import { lazy } from "react";
import { useSearchParams } from "react-router";

const Table = lazy(() => import("./_components/service-failure-table"));

export function ServiceFailuresPage() {
  const [searchParams] = useSearchParams();
  const shipmentId = searchParams.get("shipmentId") ?? undefined;

  return (
    <div className="flex h-full flex-col">
      <PageHeader
        title="Service Failures"
        description="Review unresolved pickup and delivery service failures"
      />
      <div className="p-4">
        <DataTableLazyComponent>
          <Table shipmentId={shipmentId} />
        </DataTableLazyComponent>
      </div>
    </div>
  );
}
