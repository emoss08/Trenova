import { api } from "@/lib/api";
import {
  listPersistedDocumentUploadSessions,
  persistDocumentUploadSession,
  removeDocumentUploadSession,
} from "@/lib/document-upload-store";
import { apiService } from "@/services/api";
import type {
  Document,
  DocumentProcessingProfile,
  DocumentUploadSession,
  DocumentUploadSessionState,
} from "@/types/document";
import type { UploadErrorType, UploadState } from "@/types/upload";
import { useQueryClient } from "@tanstack/react-query";
import { nanoid } from "nanoid";
import { useCallback, useEffect, useRef, useState } from "react";

const MAX_FILE_CONCURRENCY = 2;
const MAX_PART_CONCURRENCY = 4;
const MAX_PART_RETRIES = 5;

interface UseDocumentUploadOptions {
  resourceId: string;
  resourceType: string;
  processingProfile?: DocumentProcessingProfile;
  uploadMetadata?: Record<string, string>;
  invalidateQueryKey?: readonly unknown[];
  onSuccess?: (document: Document) => void;
  onError?: (error: Error, file: File) => void;
}

interface UseDocumentUploadReturn {
  uploads: UploadState[];
  uploadFiles: (files: File[]) => void;
  cancelUpload: (id: string) => void;
  retryUpload: (id: string) => void;
  removeUpload: (id: string) => void;
  clearCompleted: () => void;
  clearAll: () => void;
  isUploading: boolean;
}

function isAbortError(error: unknown): boolean {
  return error instanceof DOMException && error.name === "AbortError";
}

function isNetworkError(error: unknown): boolean {
  return (
    error instanceof TypeError ||
    (error instanceof Error &&
      (error.message.toLowerCase().includes("network") ||
        error.message.toLowerCase().includes("failed to connect") ||
        error.message.toLowerCase().includes("failed to fetch")))
  );
}

function getErrorType(error: unknown): UploadErrorType {
  if (isNetworkError(error)) {
    return "network";
  }
  if (
    error instanceof Error &&
    (error.message.toLowerCase().includes("validation") ||
      error.message.toLowerCase().includes("invalid") ||
      error.message.toLowerCase().includes("unsupported"))
  ) {
    return "validation";
  }
  if (
    error instanceof Error &&
    (error.message.toLowerCase().includes("server") ||
      error.message.toLowerCase().includes("internal"))
  ) {
    return "server";
  }

  return "unknown";
}

function sleep(ms: number): Promise<void> {
  return new Promise((resolve) => window.setTimeout(resolve, ms));
}

function buildUploadMetadata(session: DocumentUploadSession): Record<string, string> {
  const metadata: Record<string, string> = {};

  if (session.documentTypeId) {
    metadata.documentTypeId = session.documentTypeId;
  }
  if (session.lineageId) {
    metadata.lineageId = session.lineageId;
  }

  return metadata;
}

function isTerminalUploadStatus(status: DocumentUploadSession["status"]): boolean {
  return (
    status === "Completed" ||
    status === "Available" ||
    status === "Quarantined" ||
    status === "Failed" ||
    status === "Canceled" ||
    status === "Expired"
  );
}

function mapSessionStatusToUploadStatus(
  status: DocumentUploadSession["status"],
): UploadState["status"] {
  switch (status) {
    case "Uploaded":
      return "uploaded";
    case "Verifying":
      return "verifying";
    case "Finalizing":
    case "Completing":
      return "completing";
    case "Available":
    case "Completed":
      return "success";
    case "Quarantined":
      return "quarantined";
    case "Failed":
    case "Canceled":
    case "Expired":
      return "error";
    case "Paused":
      return "paused";
    default:
      return "uploading";
  }
}

export function useDocumentUpload({
  resourceId,
  resourceType,
  processingProfile,
  uploadMetadata = {},
  invalidateQueryKey,
  onSuccess,
  onError,
}: UseDocumentUploadOptions): UseDocumentUploadReturn {
  const queryClient = useQueryClient();
  const [uploads, setUploads] = useState<UploadState[]>([]);
  const pendingQueueRef = useRef<UploadState[]>([]);
  const activeUploadsRef = useRef(0);
  const abortControllersRef = useRef<Map<string, AbortController>>(new Map());
  const sessionByUploadIdRef = useRef<Map<string, DocumentUploadSession>>(new Map());
  const restoredSessionIdsRef = useRef<Set<string>>(new Set());
  const processQueueRef = useRef<() => void>(() => undefined);

  const setUploadState = useCallback((id: string, updater: (state: UploadState) => UploadState) => {
    setUploads((prev) => prev.map((upload) => (upload.id === id ? updater(upload) : upload)));
  }, []);

  const syncSessionToStore = useCallback(
    async (uploadId: string, file: File) => {
      const session = sessionByUploadIdRef.current.get(uploadId);
      if (!session) {
        return;
      }

      await persistDocumentUploadSession({
        sessionId: session.id,
        resourceId,
        resourceType,
        session,
        file,
      });
    },
    [resourceId, resourceType],
  );

  const uploadPartWithRetry = useCallback(
    async (
      uploadId: string,
      session: DocumentUploadSession,
      file: File,
      partNumber: number,
      uploadedBytes: number,
      onBytesUploaded: (part: number, bytes: number) => void,
      signal: AbortSignal,
    ) => {
      const partSize = session.partSize;
      const start = (partNumber - 1) * partSize;
      const end = Math.min(start + partSize, file.size);
      const blob = file.slice(start, end, file.type || "application/octet-stream");

      for (let attempt = 0; attempt < MAX_PART_RETRIES; attempt++) {
        const [target] = await apiService.documentService.getUploadPartTargets(session.id, [
          partNumber,
        ]);

        try {
          await api.putFileWithProgress(
            target.url,
            blob,
            (percent) => {
              const bytes = Math.round((blob.size * percent) / 100);
              onBytesUploaded(partNumber, uploadedBytes + bytes);
            },
            signal,
            file.type || "application/octet-stream",
          );
          onBytesUploaded(partNumber, uploadedBytes + blob.size);
          return;
        } catch (error) {
          if (isAbortError(error)) {
            throw error;
          }

          if (attempt === MAX_PART_RETRIES - 1) {
            throw error;
          }

          setUploadState(uploadId, (current) => ({
            ...current,
            status: "retrying",
            retryCount: attempt + 1,
          }));
          await sleep(500 * 2 ** attempt);
        }
      }
    },
    [setUploadState],
  );

  const uploadMultipart = useCallback(
    async (
      uploadId: string,
      session: DocumentUploadSession,
      file: File,
      state: DocumentUploadSessionState,
      signal: AbortSignal,
    ) => {
      const uploadedPartMap = new Map<number, number>();
      state.parts.forEach((part) => uploadedPartMap.set(part.partNumber, part.size));

      const totalParts = Math.ceil(file.size / session.partSize);
      const partNumbers = Array.from({ length: totalParts }, (_, index) => index + 1).filter(
        (partNumber) => !uploadedPartMap.has(partNumber),
      );

      if (partNumbers.length === 0) {
        return;
      }

      const inFlightBytes = new Map<number, number>();
      const updateProgress = () => {
        const committed = Array.from(uploadedPartMap.values()).reduce((sum, size) => sum + size, 0);
        const inFlight = Array.from(inFlightBytes.values()).reduce((sum, size) => sum + size, 0);
        const progress = Math.min(
          99,
          Math.round(((committed + inFlight) / Math.max(file.size, 1)) * 100),
        );

        setUploadState(uploadId, (current) => ({
          ...current,
          status: current.status === "retrying" ? "retrying" : "uploading",
          progress,
        }));
      };

      let cursor = 0;
      const worker = async () => {
        while (cursor < partNumbers.length) {
          const partNumber = partNumbers[cursor];
          cursor += 1;

          await uploadPartWithRetry(
            uploadId,
            session,
            file,
            partNumber,
            Array.from(uploadedPartMap.values()).reduce((sum, size) => sum + size, 0) -
              Array.from(uploadedPartMap.entries())
                .filter(([existingPart]) => existingPart !== partNumber)
                .reduce((sum, [, size]) => sum + size, 0),
            (_part, bytes) => {
              if (bytes >= file.size) {
                return;
              }

              const start = (partNumber - 1) * session.partSize;
              const end = Math.min(start + session.partSize, file.size);
              const relativeBytes = Math.min(bytes, end - start);
              if (relativeBytes >= end - start) {
                inFlightBytes.delete(partNumber);
                uploadedPartMap.set(partNumber, end - start);
              } else {
                inFlightBytes.set(partNumber, relativeBytes);
              }
              updateProgress();
            },
            signal,
          );
        }
      };

      await Promise.all(
        Array.from({ length: Math.min(MAX_PART_CONCURRENCY, partNumbers.length) }, () =>
          worker(),
        ),
      );
    },
    [setUploadState, uploadPartWithRetry],
  );

  const createFreshSession = useCallback(
    async (uploadState: UploadState) => {
      const session = await apiService.documentService.createUploadSession({
        resourceId,
        resourceType,
        processingProfile,
        fileName: uploadState.file.name,
        fileSize: uploadState.file.size,
        contentType: uploadState.file.type || "application/octet-stream",
        documentTypeId: uploadState.metadata?.documentTypeId,
        lineageId: uploadState.metadata?.lineageId,
      });
      sessionByUploadIdRef.current.set(uploadState.id, session);
      await syncSessionToStore(uploadState.id, uploadState.file);
      return session;
    },
    [processingProfile, resourceId, resourceType, syncSessionToStore],
  );

  const discardSession = useCallback(async (uploadId: string) => {
    const session = sessionByUploadIdRef.current.get(uploadId);
    if (session?.id) {
      restoredSessionIdsRef.current.delete(session.id);
      await removeDocumentUploadSession(session.id);
    }
    sessionByUploadIdRef.current.delete(uploadId);
  }, []);

  const waitForSessionFinalization = useCallback(
    async (uploadId: string, sessionId: string, signal: AbortSignal) => {
      while (!signal.aborted) {
        const state = await apiService.documentService.getUploadSession(sessionId);
        sessionByUploadIdRef.current.set(uploadId, state.session);

        setUploadState(uploadId, (current) => ({
          ...current,
          sessionId,
          status: mapSessionStatusToUploadStatus(state.session.status),
          progress:
            state.session.status === "Available" || state.session.status === "Completed"
              ? 100
              : current.progress,
          error:
            state.session.status === "Failed" ||
            state.session.status === "Expired" ||
            state.session.status === "Canceled" ||
            state.session.status === "Quarantined"
              ? state.session.failureMessage || current.error
              : undefined,
        }));

        if (
          state.session.status === "Available" ||
          state.session.status === "Completed"
        ) {
          if (!state.session.documentId) {
            throw new Error("Upload finalized without a document record");
          }
          return apiService.documentService.getById(state.session.documentId);
        }

        if (
          state.session.status === "Failed" ||
          state.session.status === "Expired" ||
          state.session.status === "Canceled" ||
          state.session.status === "Quarantined"
        ) {
          throw new Error(state.session.failureMessage || `Upload ${state.session.status.toLowerCase()}`);
        }

        await sleep(1500);
      }

      throw new DOMException("Upload aborted", "AbortError");
    },
    [setUploadState],
  );

  const startUpload = useCallback(
    async (uploadState: UploadState) => {
      activeUploadsRef.current += 1;

      const abortController = new AbortController();
      abortControllersRef.current.set(uploadState.id, abortController);

      try {
        let session = sessionByUploadIdRef.current.get(uploadState.id);
        if (!session) {
          session = await createFreshSession(uploadState);
        }
        if (!session) {
          throw new Error("Failed to initialize upload session");
        }
        const activeSession = session;

        setUploadState(uploadState.id, (current) => ({
          ...current,
          sessionId: activeSession.id,
          status: "uploading",
          progress: 0,
          error: undefined,
          errorType: undefined,
        }));

        let state = await apiService.documentService.getUploadSession(activeSession.id);
        if (
          isTerminalUploadStatus(state.session.status) ||
          state.session.fileSize !== uploadState.file.size ||
          state.session.originalName !== uploadState.file.name
        ) {
          await discardSession(uploadState.id);
          session = await createFreshSession(uploadState);
          state = await apiService.documentService.getUploadSession(session.id);
        }
        sessionByUploadIdRef.current.set(uploadState.id, state.session);
        await syncSessionToStore(uploadState.id, uploadState.file);

        if (
          state.session.status === "Uploaded" ||
          state.session.status === "Verifying" ||
          state.session.status === "Finalizing" ||
          state.session.status === "Completing"
        ) {
          const document = await waitForSessionFinalization(
            uploadState.id,
            state.session.id,
            abortController.signal,
          );
          restoredSessionIdsRef.current.delete(state.session.id);
          await removeDocumentUploadSession(state.session.id);
          sessionByUploadIdRef.current.delete(uploadState.id);

          setUploadState(uploadState.id, (current) => ({
            ...current,
            status: "success",
            progress: 100,
          }));

          void queryClient.invalidateQueries({
            queryKey: invalidateQueryKey || ["documents", resourceType, resourceId],
          });

          onSuccess?.(document);
          return;
        }

        if (state.session.strategy === "single") {
          const [target] = await apiService.documentService.getUploadPartTargets(state.session.id, [1]);
          await api.putFileWithProgress(
            target.url,
            uploadState.file,
            (percent) => {
              setUploadState(uploadState.id, (current) => ({
                ...current,
                status: "uploading",
                progress: Math.min(99, percent),
              }));
            },
            abortController.signal,
            uploadState.file.type || "application/octet-stream",
          );
        } else {
          await uploadMultipart(
            uploadState.id,
            state.session,
            uploadState.file,
            state,
            abortController.signal,
          );
        }

        setUploadState(uploadState.id, (current) => ({
          ...current,
          status: "uploaded",
          progress: 100,
        }));

        const completionSession = await apiService.documentService.completeUploadSession(state.session.id);
        sessionByUploadIdRef.current.set(uploadState.id, completionSession);
        await syncSessionToStore(uploadState.id, uploadState.file);
        const document = await waitForSessionFinalization(
          uploadState.id,
          completionSession.id,
          abortController.signal,
        );
        restoredSessionIdsRef.current.delete(completionSession.id);
        await removeDocumentUploadSession(state.session.id);
        sessionByUploadIdRef.current.delete(uploadState.id);

        setUploadState(uploadState.id, (current) => ({
          ...current,
          status: "success",
          progress: 100,
        }));

        void queryClient.invalidateQueries({
          queryKey: invalidateQueryKey || ["documents", resourceType, resourceId],
        });

        onSuccess?.(document);
      } catch (error) {
        if (isAbortError(error)) {
          await discardSession(uploadState.id);
          setUploads((prev) => prev.filter((upload) => upload.id !== uploadState.id));
          return;
        }

        const message = error instanceof Error ? error.message : "Upload failed";
        const errorType = getErrorType(error);

        setUploadState(uploadState.id, (current) => ({
          ...current,
          status: errorType === "network" ? "paused" : "error",
          error: message,
          errorType,
        }));
        if (errorType !== "network") {
          await discardSession(uploadState.id);
        }
        onError?.(error instanceof Error ? error : new Error(message), uploadState.file);
      } finally {
        activeUploadsRef.current -= 1;
        abortControllersRef.current.delete(uploadState.id);
        while (
          activeUploadsRef.current < MAX_FILE_CONCURRENCY &&
          pendingQueueRef.current.length > 0
        ) {
          const next = pendingQueueRef.current.shift();
          if (next) {
            void startUpload(next);
          }
        }
      }
    },
    [
      invalidateQueryKey,
      onError,
      onSuccess,
      queryClient,
      resourceId,
      resourceType,
      setUploadState,
      syncSessionToStore,
      createFreshSession,
      discardSession,
      waitForSessionFinalization,
      uploadMultipart,
    ],
  );

  const processQueue = useCallback(() => {
    while (
      activeUploadsRef.current < MAX_FILE_CONCURRENCY &&
      pendingQueueRef.current.length > 0
    ) {
      const next = pendingQueueRef.current.shift();
      if (next) {
        void startUpload(next);
      }
    }
  }, [startUpload]);

  processQueueRef.current = processQueue;

  const uploadFiles = useCallback(
    (files: File[]) => {
      const nextUploads = files.map((file) => ({
        id: nanoid(),
        file,
        progress: 0,
        status: "pending" as const,
        metadata: { ...uploadMetadata },
      }));

      setUploads((prev) => [...prev, ...nextUploads]);
      pendingQueueRef.current.push(...nextUploads);
      processQueue();
    },
    [processQueue, uploadMetadata],
  );

  const cancelUpload = useCallback(
    (id: string) => {
      const controller = abortControllersRef.current.get(id);
      const session = sessionByUploadIdRef.current.get(id);

      if (controller) {
        controller.abort();
      }

      if (session?.id) {
        restoredSessionIdsRef.current.delete(session.id);
        void apiService.documentService.cancelUploadSession(session.id).catch(() => undefined);
        void removeDocumentUploadSession(session.id);
      }

      sessionByUploadIdRef.current.delete(id);
      pendingQueueRef.current = pendingQueueRef.current.filter((upload) => upload.id !== id);
      setUploads((prev) => prev.filter((upload) => upload.id !== id));
    },
    [],
  );

  const retryUpload = useCallback(
    (id: string) => {
      const upload = uploads.find((entry) => entry.id === id);
      if (!upload) {
        return;
      }

      const nextUpload: UploadState = {
        ...upload,
        status: "pending",
        progress: 0,
        retryCount: (upload.retryCount ?? 0) + 1,
        error: undefined,
        errorType: undefined,
      };

      setUploads((prev) => prev.map((entry) => (entry.id === id ? nextUpload : entry)));
      void discardSession(id);
      pendingQueueRef.current.push(nextUpload);
      processQueue();
    },
    [discardSession, processQueue, uploads],
  );

  const removeUpload = useCallback(
    (id: string) => {
      const session = sessionByUploadIdRef.current.get(id);
      if (session?.id) {
        restoredSessionIdsRef.current.delete(session.id);
        void removeDocumentUploadSession(session.id);
      }
      sessionByUploadIdRef.current.delete(id);
      setUploads((prev) => prev.filter((upload) => upload.id !== id));
    },
    [],
  );

  const clearCompleted = useCallback(() => {
    setUploads((prev) => {
      prev
        .filter((upload) => upload.status === "success")
        .forEach((upload) => {
          const session = sessionByUploadIdRef.current.get(upload.id);
          if (session?.id) {
            restoredSessionIdsRef.current.delete(session.id);
          }
          sessionByUploadIdRef.current.delete(upload.id);
        });
      return prev.filter((upload) => upload.status !== "success");
    });
  }, []);

  const clearAll = useCallback(() => {
    abortControllersRef.current.forEach((controller) => controller.abort());
    abortControllersRef.current.clear();
    restoredSessionIdsRef.current.clear();
    sessionByUploadIdRef.current.clear();
    pendingQueueRef.current = [];
    activeUploadsRef.current = 0;
    setUploads([]);
  }, []);

  useEffect(() => {
    let cancelled = false;

    const restore = async () => {
      try {
        const [persisted, activeSessions] = await Promise.all([
          listPersistedDocumentUploadSessions(resourceType, resourceId),
          apiService.documentService.listActiveUploadSessions(resourceType, resourceId),
        ]);

        if (cancelled) {
          return;
        }

        const activeSessionMap = new Map(activeSessions.map((session) => [session.id, session]));
        const restoredUploads: UploadState[] = [];

        for (const record of persisted) {
          const activeSession = activeSessionMap.get(record.sessionId);
          if (!activeSession) {
            await removeDocumentUploadSession(record.sessionId);
            restoredSessionIdsRef.current.delete(record.sessionId);
            continue;
          }
          if (restoredSessionIdsRef.current.has(activeSession.id)) {
            continue;
          }

          const id = nanoid();
          sessionByUploadIdRef.current.set(id, activeSession);
          restoredSessionIdsRef.current.add(activeSession.id);
          restoredUploads.push({
            id,
            file: record.file,
            progress: Math.round(
              (activeSession.uploadedParts.reduce((sum, part) => sum + part.size, 0) /
                Math.max(activeSession.fileSize, 1)) *
                100,
            ),
            status:
              activeSession.status === "Paused"
                ? "paused"
                : mapSessionStatusToUploadStatus(activeSession.status),
            metadata: buildUploadMetadata(activeSession),
            sessionId: activeSession.id,
          });
        }

        if (restoredUploads.length > 0) {
          setUploads((prev) => {
            const existingKeys = new Set(prev.map((upload) => upload.sessionId).filter(Boolean));
            return [
              ...prev,
              ...restoredUploads.filter((upload) => !existingKeys.has(upload.sessionId)),
            ];
          });
          const queuedSessionIDs = new Set(
            pendingQueueRef.current.map((upload) => upload.sessionId).filter(Boolean),
          );
          pendingQueueRef.current.push(
            ...restoredUploads.filter(
              (upload) =>
                ["uploading", "uploaded", "verifying", "completing"].includes(upload.status) &&
                !queuedSessionIDs.has(upload.sessionId),
            ),
          );
          processQueueRef.current();
        }
      } catch {
        return;
      }
    };

    void restore();

    return () => {
      cancelled = true;
    };
  }, [resourceId, resourceType]);

  const isUploading = uploads.some((upload) =>
    ["pending", "uploading", "processing", "uploaded", "verifying", "retrying", "completing"].includes(
      upload.status,
    ),
  );

  return {
    uploads,
    uploadFiles,
    cancelUpload,
    retryUpload,
    removeUpload,
    clearCompleted,
    clearAll,
    isUploading,
  };
}
