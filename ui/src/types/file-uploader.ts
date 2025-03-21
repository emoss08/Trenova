import type { TableSheetProps } from "./data-table";

export type DocumentType = {
  value: string;
  label: string;
};

export type FileUploadProps = {
  // Required props
  resourceType: string;
  resourceId: string;

  // Optional props
  allowMultiple?: boolean;
  documentTypes?: DocumentType[];
  maxFileSizeMB?: number;
  acceptedFileTypes?: string;
  onUploadComplete?: (response: any) => void;
  onUploadError?: (error: any) => void;
  onCancel?: () => void;
  showDocumentTypeSelection?: boolean;
  requireApproval?: boolean;
};

export type UploadingFile = {
  file: File;
  progress: number;
  documentType: string;
  status: "uploading" | "success" | "error";
  error?: string;
  fileSize?: string; // Formatted file size for display
};

export type UploadFileParams = {
  file: File;
  resourceId: string;
  resourceType: string;
  documentType: string;
  description: string;
  onProgress: (progress: number) => void;
};

export type UploadError = {
  fileName: string;
  message: string;
  details?: string;
  code?: string;
};

type FileStats = {
  total: number;
  pending: number;
  success: number;
  error: number;
};

export type FileUploadDockProps = {
  fileStats: FileStats;
  isDirty: boolean;
  isSubmitting: boolean;
  onCancel?: () => void;
  onUpload?: () => void;
};

export type FileUploadErrorDialogProps = {
  errorsByType: Record<string, UploadError[]>;
  clearErrors: () => void;
} & TableSheetProps;

export type FileUploadCardProps = {
  fileInfo: UploadingFile;
  index: number;
  removeFile: (index: number) => void;
};

export type FileUploadErrorCardProps = {
  errors: UploadError[];
  onOpenChange: (open: boolean) => void;
};
