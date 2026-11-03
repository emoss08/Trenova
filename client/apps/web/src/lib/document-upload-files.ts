export const DOCUMENT_UPLOAD_ACCEPT =
  ".pdf,.jpg,.jpeg,.png,.webp,.gif,.doc,.docx,.xls,.xlsx,.txt,.csv";

export const DOCUMENT_UPLOAD_MAX_SIZE = 50 * 1024 * 1024;

export interface RejectedFile {
  file: File;
  reason: "size" | "type";
}

/** Whether a file matches an `accept` list of extensions, MIME types and `type/*` wildcards. */
export function isAcceptedFileType(file: File, accept: string): boolean {
  const acceptedTypes = accept
    .split(",")
    .map((entry) => entry.trim().toLowerCase())
    .filter(Boolean);

  if (acceptedTypes.length === 0) {
    return true;
  }

  const lowerName = file.name.toLowerCase();
  const lowerType = file.type.toLowerCase();

  return acceptedTypes.some((entry) => {
    if (entry.startsWith(".")) {
      return lowerName.endsWith(entry);
    }
    if (entry.endsWith("/*")) {
      return lowerType.startsWith(entry.slice(0, -1));
    }
    return lowerType === entry;
  });
}

/**
 * Splits picked or dropped files into those that can be uploaded and those that
 * cannot, with why. A single-file picker keeps only the first file it accepts.
 */
export function sortUploadFiles(
  files: Iterable<File>,
  {
    accept = DOCUMENT_UPLOAD_ACCEPT,
    maxFileSize = DOCUMENT_UPLOAD_MAX_SIZE,
    multiple = true,
  }: { accept?: string; maxFileSize?: number; multiple?: boolean } = {},
): { accepted: File[]; rejected: RejectedFile[] } {
  const accepted: File[] = [];
  const rejected: RejectedFile[] = [];

  for (const file of files) {
    if (!isAcceptedFileType(file, accept)) {
      rejected.push({ file, reason: "type" });
    } else if (file.size > maxFileSize) {
      rejected.push({ file, reason: "size" });
    } else {
      accepted.push(file);
    }
  }

  return { accepted: multiple ? accepted : accepted.slice(0, 1), rejected };
}
