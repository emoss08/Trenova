import { useApiMutation } from "@/hooks/use-api-mutation";
import { openInNewTab } from "@/lib/open-in-new-tab";
import {
  hasPendingDocumentWork,
  PENDING_DOCUMENTS_REFETCH_MS,
  resourceDocumentsQuery,
} from "@/lib/resource-documents";
import { documentService } from "@/services/document";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "@trenova/shared/components/ui/toast";
import { useT } from "@trenova/shared/i18n/use-t";
import type { Document, DocumentPacketSummary } from "@trenova/shared/types/document";

export const WORKER_DOCUMENT_RESOURCE = "worker";

/** Every key a change to the worker's documents makes stale, the overview's file count included. */
export function workerDocumentKeys(workerId: string, versionOf?: string | null) {
  return [
    ["documents", WORKER_DOCUMENT_RESOURCE, workerId],
    ["document-packet-summary", WORKER_DOCUMENT_RESOURCE, workerId],
    ["document-versions", versionOf ?? null],
    ["worker"],
  ] as const;
}

/**
 * The worker's documents and what can be done to them, on the same reads, keys and services the
 * shared documents tab uses, so an upload, a review or a delete here refreshes every other view of
 * the same files.
 */
export function useWorkerDocuments(workerId: string, versionOf: Document | null) {
  const t = useT();
  const queryClient = useQueryClient();

  const documentsQuery = useQuery({
    ...resourceDocumentsQuery(WORKER_DOCUMENT_RESOURCE, workerId),
    refetchInterval: (query) =>
      query.state.data?.some(hasPendingDocumentWork) ? PENDING_DOCUMENTS_REFETCH_MS : false,
  });
  const packetQuery = useQuery<DocumentPacketSummary>({
    queryKey: ["document-packet-summary", WORKER_DOCUMENT_RESOURCE, workerId],
    queryFn: () => documentService.getPacketSummary(WORKER_DOCUMENT_RESOURCE, workerId),
  });
  const versionId = versionOf?.id ?? null;
  const versionsQuery = useQuery({
    queryKey: ["document-versions", versionId],
    queryFn: () => (versionId ? documentService.getVersions(versionId) : Promise.resolve([])),
    enabled: versionId !== null,
  });

  const refresh = () =>
    Promise.all(
      workerDocumentKeys(workerId, versionOf?.id).map((queryKey) =>
        queryClient.invalidateQueries({ queryKey }),
      ),
    );

  const remove = useApiMutation({
    scope: { id: `worker-documents:${workerId}` },
    resourceName: "Document",
    mutationFn: (document: Document) => documentService.delete(document.id),
    onSuccess: () => void refresh(),
  });
  const removeMany = useApiMutation({
    scope: { id: `worker-documents:${workerId}` },
    resourceName: "Documents",
    mutationFn: (ids: string[]) => documentService.bulkDelete(ids),
    onSuccess: (result) => {
      void refresh();
      toast.success(
        t("{0, plural, one {# document deleted} other {# documents deleted}}", result.deletedCount),
      );
    },
  });
  const approve = useApiMutation({
    scope: { id: `worker-documents-review:${workerId}` },
    resourceName: "Document",
    mutationFn: (document: Document) => documentService.approve(document.id),
    onSuccess: () => void refresh(),
  });
  const restore = useApiMutation({
    scope: { id: `worker-documents-review:${workerId}` },
    resourceName: "Document",
    mutationFn: (document: Document) => documentService.restoreVersion(document.id),
    onSuccess: () => void refresh(),
  });

  const open = async (document: Document) => {
    try {
      openInNewTab(await documentService.getViewUrl(document.id));
    } catch {
      toast.error(t("The document could not be opened"));
    }
  };
  const download = async (document: Document) => {
    try {
      openInNewTab(await documentService.getDownloadUrl(document.id));
    } catch {
      toast.error(t("The download could not be started"));
    }
  };

  return {
    documentsQuery,
    packetSummary: packetQuery.data ?? null,
    versions: versionsQuery.data ?? [],
    versionsLoading: versionsQuery.isLoading,
    refresh,
    remove,
    removeMany,
    approve,
    restore,
    open,
    download,
  };
}
