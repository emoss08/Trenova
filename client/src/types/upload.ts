export type UploadStatus =
  | "pending"
  | "processing"
  | "uploading"
  | "uploaded"
  | "verifying"
  | "paused"
  | "retrying"
  | "completing"
  | "quarantined"
  | "success"
  | "error";

export type UploadErrorType = "network" | "validation" | "server" | "unknown";

export interface UploadState {
  id: string;
  file: File;
  progress: number;
  status: UploadStatus;
  error?: string;
  errorType?: UploadErrorType;
  retryCount?: number;
  metadata?: Record<string, string>;
  sessionId?: string;
}
