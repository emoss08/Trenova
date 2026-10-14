import { DataTableLazyComponent, LazyComponent } from "@trenova/shared/components/error-boundary";
import { lazy } from "react";
import type { AuditView } from "../../ai-control-tabs";
import { AuditTrailSkeleton, EXPORT_COLUMNS } from "./audit-skeleton";
import { ChainStatusHeader } from "./chain-status";

const AuditTrailView = lazy(() => import("./audit-trail-view"));
const AuditExportsView = lazy(() => import("./audit-exports-view"));

/**
 * The AI audit trail: Nova's sentence on the signed chain every run, model call, tool
 * call and decision is written into, with a way to check it now, and below it the trail
 * itself or the files it has been exported to, as the view switch picks.
 */
export default function AuditTab({
  view,
  onOpenExports,
}: {
  view: AuditView;
  onOpenExports: () => void;
}) {
  return (
    <div className="tabp">
      <ChainStatusHeader />
      {view === "trail" && (
        <LazyComponent fallback={<AuditTrailSkeleton />}>
          <AuditTrailView onOpenExports={onOpenExports} />
        </LazyComponent>
      )}
      {view === "exports" && (
        <DataTableLazyComponent columnCount={EXPORT_COLUMNS}>
          <AuditExportsView />
        </DataTableLazyComponent>
      )}
    </div>
  );
}
