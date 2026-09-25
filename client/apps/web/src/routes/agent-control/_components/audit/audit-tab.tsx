import { DataTableLazyComponent } from "@trenova/shared/components/error-boundary";
import { lazy } from "react";
import type { AuditView } from "../../ai-control-tabs";

const AuditTrailView = lazy(() => import("./audit-trail-view"));
const AuditExportsView = lazy(() => import("./audit-exports-view"));

/**
 * The AI audit trail: every run, model call, tool call and decision, signed
 * into a chain that shows when a row was changed, and the files it has been
 * exported to. Which one is showing is chosen on the rail.
 */
export default function AuditTab({
  view,
  onOpenExports,
}: {
  view: AuditView;
  onOpenExports: () => void;
}) {
  return (
    <DataTableLazyComponent>
      {view === "trail" && <AuditTrailView onOpenExports={onOpenExports} />}
      {view === "exports" && <AuditExportsView />}
    </DataTableLazyComponent>
  );
}
