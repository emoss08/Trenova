import { useOpenedOnce } from "@/hooks/use-opened-once";
import { lazyComponent } from "@/lib/lazy-component";
import { LazyComponent } from "@trenova/shared/components/error-boundary";
import type { Document } from "@trenova/shared/types/document";
import { WORKER_DOCUMENT_RESOURCE } from "./documents-model";
import type { useWorkerDocuments } from "./use-worker-documents";

const IntelligenceDialog = lazyComponent(
  () => import("@/components/documents/document-intelligence-dialog"),
  (module) => module.DocumentIntelligenceDialog,
);
const RejectDialog = lazyComponent(
  () => import("@/components/documents/reject-document-dialog"),
  (module) => module.RejectDocumentDialog,
);
const SignatureDialog = lazyComponent(
  () => import("@/components/documents/document-signature-dialog"),
  (module) => module.DocumentSignatureDialog,
);
const VersionDialog = lazyComponent(
  () => import("@/components/documents/document-version-dialog"),
  (module) => module.DocumentVersionDialog,
);

/** Reads each dialog's code ahead of its menu item being chosen (`RowAction.preload`). */
export const DOCUMENT_DIALOG_PRELOADS = {
  inspect: IntelligenceDialog.preload,
  reject: RejectDialog.preload,
  signature: SignatureDialog.preload,
  versions: VersionDialog.preload,
} as const;

type DocumentDialogsProps = {
  workerId: string;
  inspecting: Document | null;
  onInspectingChange: (document: Document | null) => void;
  rejecting: Document | null;
  onRejectingChange: (document: Document | null) => void;
  signing: Document | null;
  onSigningChange: (document: Document | null) => void;
  versionOf: Document | null;
  onVersionOfChange: (document: Document | null) => void;
  documents: Pick<
    ReturnType<typeof useWorkerDocuments>,
    "refresh" | "restore" | "versions" | "versionsLoading"
  >;
  onUploadNewVersion: (document: Document) => void;
};

/**
 * The shared document dialogs a row's menu opens (what was read, reject, record the signature,
 * versions). Each is read from the network the first time its menu item is chosen, and stays
 * mounted after that so closing can play out.
 */
export function DocumentDialogs({
  workerId,
  inspecting,
  onInspectingChange,
  rejecting,
  onRejectingChange,
  signing,
  onSigningChange,
  versionOf,
  onVersionOfChange,
  documents,
  onUploadNewVersion,
}: DocumentDialogsProps) {
  const inspectOpened = useOpenedOnce(inspecting !== null);
  const rejectOpened = useOpenedOnce(rejecting !== null);
  const signOpened = useOpenedOnce(signing !== null);
  const versionsOpened = useOpenedOnce(versionOf !== null);

  return (
    <>
      {inspectOpened ? (
        <LazyComponent fallback={null}>
          <IntelligenceDialog.Component
            open={inspecting !== null}
            onOpenChange={(next) => !next && onInspectingChange(null)}
            document={inspecting}
            resourceType={WORKER_DOCUMENT_RESOURCE}
            resourceId={workerId}
          />
        </LazyComponent>
      ) : null}
      {rejectOpened ? (
        <LazyComponent fallback={null}>
          <RejectDialog.Component
            open={rejecting !== null}
            onOpenChange={(next) => !next && onRejectingChange(null)}
            document={rejecting}
            onRejected={() => void documents.refresh()}
          />
        </LazyComponent>
      ) : null}
      {signOpened ? (
        <LazyComponent fallback={null}>
          <SignatureDialog.Component
            open={signing !== null}
            onOpenChange={(next) => !next && onSigningChange(null)}
            document={signing}
            onSaved={() => void documents.refresh()}
          />
        </LazyComponent>
      ) : null}
      {versionsOpened ? (
        <LazyComponent fallback={null}>
          <VersionDialog.Component
            open={versionOf !== null}
            onOpenChange={(next) => !next && onVersionOfChange(null)}
            document={versionOf}
            versions={documents.versions}
            isLoading={documents.versionsLoading}
            isRestoring={documents.restore.isPending}
            onRestore={(document) =>
              void documents.restore.mutateAsync(document).catch(() => undefined)
            }
            onUploadNewVersion={onUploadNewVersion}
          />
        </LazyComponent>
      ) : null}
    </>
  );
}
