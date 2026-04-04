import { LazyComponent } from "@/components/error-boundary";
import { lazy } from "react";

const ImportWorkspace = lazy(() =>
  import("./_components/rate-confirmation-import/import-workspace").then((m) => ({
    default: m.ImportWorkspace,
  })),
);

export function ShipmentImportPage() {
  return (
    <div className="h-full">
      <LazyComponent>
        <ImportWorkspace />
      </LazyComponent>
    </div>
  );
}
