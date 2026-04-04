import { api } from "@/lib/api";
import type { Document } from "@/types/document";
import type {
  UploadErrorType,
  UploadState,
} from "@/types/upload";
import { documentSchema } from "@/types/document";
import { useQueryClient } from "@tanstack/react-query";
import { nanoid } from "nanoid";
import { useCallback, useRef, useState } from "react";
export type { UploadErrorType, UploadState, UploadStatus } from "@/types/upload";

const MAX_AUTO_RETRIES = 3;

interface UseUploadWithProgressOptions {
  resourceId: string;
  resourceType: string;
  maxConcurrent?: number;
  uploadEndpoint?: string;
  uploadMetadata?: Record<string, string>;
  parseResponse?: (response: unknown) => unknown;
  invalidateQueryKey?: readonly unknown[];
  transformFile?: (file: File) => Promise<File>;
  onSuccess?: (result: unknown) => void;
  onError?: (error: Error, file: File) => void;
}

interface UseUploadWithProgressReturn {
  uploads: UploadState[];
  uploadFiles: (files: File[]) => void;
  cancelUpload: (id: string) => void;
  retryUpload: (id: string) => void;
  removeUpload: (id: string) => void;
  clearCompleted: () => void;
  clearAll: () => void;
  isUploading: boolean;
}

export function useUploadWithProgress({
  resourceId,
  resourceType,
  maxConcurrent = 3,
  uploadEndpoint = "/documents/upload/",
  uploadMetadata = {},
  parseResponse = (response) => documentSchema.parse(response as Document),
  invalidateQueryKey,
  transformFile,
  onSuccess,
  onError,
}: UseUploadWithProgressOptions): UseUploadWithProgressReturn {
  const queryClient = useQueryClient();
  const [uploads, setUploads] = useState<UploadState[]>([]);
  const abortControllersRef = useRef<Map<string, AbortController>>(new Map());
  const activeUploadsRef = useRef(0);
  const pendingQueueRef = useRef<UploadState[]>([]);

  const processQueue = useCallback(() => {
    while (
      activeUploadsRef.current < maxConcurrent &&
      pendingQueueRef.current.length > 0
    ) {
      const nextUpload = pendingQueueRef.current.shift();
      if (nextUpload) {
        void startUpload(nextUpload);
      }
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [maxConcurrent]);

  const startUpload = useCallback(
    async (uploadState: UploadState) => {
      activeUploadsRef.current++;

      let uploadFile = uploadState.file;

      if (transformFile) {
        setUploads((prev) =>
          prev.map((u) =>
            u.id === uploadState.id
              ? { ...u, status: "processing" as const }
              : u,
          ),
        );

        try {
          uploadFile = await transformFile(uploadState.file);
          setUploads((prev) =>
            prev.map((u) =>
              u.id === uploadState.id ? { ...u, file: uploadFile } : u,
            ),
          );
        } catch (error) {
          const errorMessage =
            error instanceof Error ? error.message : "Failed to process file";
          setUploads((prev) =>
            prev.map((u) =>
              u.id === uploadState.id
                ? {
                    ...u,
                    status: "error" as const,
                    error: errorMessage,
                    errorType: "validation" as const,
                    retryCount: uploadState.retryCount ?? 0,
                  }
                : u,
            ),
          );
          onError?.(
            error instanceof Error ? error : new Error(errorMessage),
            uploadState.file,
          );
          activeUploadsRef.current--;
          processQueue();
          return;
        }
      }

      setUploads((prev) =>
        prev.map((u) =>
          u.id === uploadState.id ? { ...u, status: "uploading" as const } : u,
        ),
      );

      const abortController = new AbortController();
      abortControllersRef.current.set(uploadState.id, abortController);

      const formData = new FormData();
      formData.append("file", uploadFile);
      formData.append("resourceId", resourceId);
      formData.append("resourceType", resourceType);
      const effectiveMetadata = uploadState.metadata ?? uploadMetadata;
      for (const [key, value] of Object.entries(effectiveMetadata)) {
        formData.append(key, value);
      }

      try {
        const response = await api.uploadWithProgress<unknown>(
          uploadEndpoint,
          formData,
          (percent) => {
            setUploads((prev) =>
              prev.map((u) =>
                u.id === uploadState.id ? { ...u, progress: percent } : u,
              ),
            );
          },
          abortController.signal,
        );

        const parsed = parseResponse(response);

        setUploads((prev) =>
          prev.map((u) =>
            u.id === uploadState.id
              ? { ...u, status: "success" as const, progress: 100 }
              : u,
          ),
        );

        void queryClient.invalidateQueries({
          queryKey: invalidateQueryKey || [
            "documents",
            resourceType,
            resourceId,
          ],
        });

        onSuccess?.(parsed);
      } catch (error) {
        if (error instanceof DOMException && error.name === "AbortError") {
          setUploads((prev) => prev.filter((u) => u.id !== uploadState.id));
        } else {
          const errorMessage =
            error instanceof Error ? error.message : "Upload failed";

          let errorType: UploadErrorType = "unknown";
          if (
            error instanceof TypeError ||
            errorMessage.toLowerCase().includes("network") ||
            errorMessage.toLowerCase().includes("failed to fetch")
          ) {
            errorType = "network";
          } else if (
            errorMessage.toLowerCase().includes("validation") ||
            errorMessage.toLowerCase().includes("invalid") ||
            errorMessage.toLowerCase().includes("unsupported")
          ) {
            errorType = "validation";
          } else if (
            errorMessage.toLowerCase().includes("server") ||
            errorMessage.toLowerCase().includes("500") ||
            errorMessage.toLowerCase().includes("internal")
          ) {
            errorType = "server";
          }

          const currentRetryCount = uploadState.retryCount ?? 0;

          setUploads((prev) =>
            prev.map((u) =>
              u.id === uploadState.id
                ? {
                    ...u,
                    status: "error" as const,
                    error: errorMessage,
                    errorType,
                    retryCount: currentRetryCount,
                  }
                : u,
            ),
          );
          onError?.(
            error instanceof Error ? error : new Error(errorMessage),
            uploadState.file,
          );
        }
      } finally {
        activeUploadsRef.current--;
        abortControllersRef.current.delete(uploadState.id);
        processQueue();
      }
    },
    [
      resourceId,
      resourceType,
      uploadMetadata,
      uploadEndpoint,
      parseResponse,
      invalidateQueryKey,
      transformFile,
      queryClient,
      onSuccess,
      onError,
      processQueue,
    ],
  );

  const uploadFiles = useCallback(
    (files: File[]) => {
      const newUploads: UploadState[] = files.map((file) => ({
        id: nanoid(),
        file,
        progress: 0,
        status: "pending" as const,
        metadata: { ...uploadMetadata },
      }));

      setUploads((prev) => [...prev, ...newUploads]);
      pendingQueueRef.current.push(...newUploads);
      processQueue();
    },
    [processQueue, uploadMetadata],
  );

  const cancelUpload = useCallback((id: string) => {
    const controller = abortControllersRef.current.get(id);
    if (controller) {
      controller.abort();
    }
    pendingQueueRef.current = pendingQueueRef.current.filter(
      (u) => u.id !== id,
    );
    setUploads((prev) => prev.filter((u) => u.id !== id));
  }, []);

  const retryUpload = useCallback(
    (id: string) => {
      const currentUploads = uploads;
      const upload = currentUploads.find((u) => u.id === id);
      if (!upload || upload.status !== "error") return;

      const newRetryCount = (upload.retryCount ?? 0) + 1;
      if (newRetryCount > MAX_AUTO_RETRIES) {
        return;
      }

      const resetUpload: UploadState = {
        ...upload,
        status: "pending",
        progress: 0,
        error: undefined,
        errorType: undefined,
        retryCount: newRetryCount,
        metadata: upload.metadata,
      };

      setUploads((prev) => prev.map((u) => (u.id === id ? resetUpload : u)));
      pendingQueueRef.current.push(resetUpload);
      processQueue();
    },
    [uploads, processQueue],
  );

  const removeUpload = useCallback((id: string) => {
    const controller = abortControllersRef.current.get(id);
    if (controller) {
      controller.abort();
    }
    pendingQueueRef.current = pendingQueueRef.current.filter(
      (u) => u.id !== id,
    );
    setUploads((prev) => prev.filter((u) => u.id !== id));
  }, []);

  const clearCompleted = useCallback(() => {
    setUploads((prev) =>
      prev.filter((u) => u.status !== "success" && u.status !== "error"),
    );
  }, []);

  const clearAll = useCallback(() => {
    abortControllersRef.current.forEach((controller) => controller.abort());
    abortControllersRef.current.clear();
    pendingQueueRef.current = [];
    activeUploadsRef.current = 0;
    setUploads([]);
  }, []);

  const isUploading = uploads.some(
    (u) =>
      u.status === "uploading" ||
      u.status === "pending" ||
      u.status === "processing",
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
