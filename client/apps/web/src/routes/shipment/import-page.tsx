import { LazyComponent } from "@trenova/shared/components/error-boundary";
import { lazy } from "react";
import { ImportWorkspaceSkeleton } from "./_components/rate-confirmation-import/import-workspace-skeleton";

const ImportWorkspace = lazy(() =>
  import("./_components/rate-confirmation-import/import-workspace").then((m) => ({
    default: m.ImportWorkspace,
  })),
);

export function ShipmentImportPage() {
  return (
    <div className="h-full">
      <LazyComponent fallback={<ImportWorkspaceSkeleton />}>
        <ImportWorkspace />
      </LazyComponent>
    </div>
  );
}
