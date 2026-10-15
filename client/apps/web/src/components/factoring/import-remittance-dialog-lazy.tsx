import { lazyComponent } from "@/lib/lazy-component";
import { LazyComponent } from "@trenova/shared/components/error-boundary";

const importDialog = lazyComponent(
  () => import("./import-remittance-dialog"),
  (module) => module.ImportRemittanceDialog,
);
const ImportDialog = importDialog.Component;

/**
 * Reads the import dialog's code ahead of the click: called when the pointer
 * or focus reaches the button that opens it. Repeat calls share the read.
 */
export function preloadImportRemittanceDialog(): void {
  void importDialog.preload().catch(() => undefined);
}

/** The remittance import, whose code is read only once someone opens it. */
export function LazyImportRemittanceDialog({
  open,
  onOpenChange,
  factoringCompanyId,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  factoringCompanyId?: string;
}) {
  if (!open) return null;
  return (
    <LazyComponent fallback={null}>
      <ImportDialog open onOpenChange={onOpenChange} factoringCompanyId={factoringCompanyId} />
    </LazyComponent>
  );
}
