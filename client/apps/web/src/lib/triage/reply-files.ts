import type {
  InboundReplyAttachable,
  InboundReplyAttachableSource,
  InboundReplyAttachment,
} from "@/lib/graphql/inbox";

/**
 * The files a reply carries, as the composer holds them: each is a stored
 * document, named and sized as it was when it was chosen, so the chip and the
 * running total agree with what the server will send.
 */
export type ReplyFile = InboundReplyAttachment;

/** What a section of the Attach menu holds: the documents stored on one record. */
export type AttachableSection = {
  source: InboundReplyAttachableSource;
  documents: InboundReplyAttachable[];
};

/** Where an upload from the computer is stored, so it is on file and not only in the email. */
export type UploadTarget = { resourceType: string; resourceId: string };

const SOURCE_ORDER: readonly InboundReplyAttachableSource[] = [
  "Message",
  "Shipment",
  "Customer",
  "Carrier",
];

const EXTENSION = /\.([a-z0-9]{2,5})$/i;
const IMAGE_EXTENSIONS: ReadonlySet<string> = new Set([
  "jpg",
  "jpeg",
  "png",
  "gif",
  "heic",
  "webp",
  "tif",
  "tiff",
]);

/** A file's extension, lower case, for its badge; "file" when it has none. */
export function fileExtension(fileName: string): string {
  return EXTENSION.exec(fileName)?.[1]?.toLowerCase() ?? "file";
}

export function isImageExtension(extension: string): boolean {
  return IMAGE_EXTENSIONS.has(extension);
}

export function withFile(files: readonly ReplyFile[], file: ReplyFile): ReplyFile[] {
  return files.some((existing) => existing.documentId === file.documentId)
    ? [...files]
    : [...files, file];
}

export function withoutFile(files: readonly ReplyFile[], documentId: string): ReplyFile[] {
  return files.filter((file) => file.documentId !== documentId);
}

export function totalBytes(files: readonly ReplyFile[]): number {
  return files.reduce((sum, file) => sum + file.byteSize, 0);
}

export function remainingBytes(files: readonly ReplyFile[], limitBytes: number): number {
  return Math.max(limitBytes - totalBytes(files), 0);
}

/** Whether a file of this size still fits beside the files already chosen. */
export function fitsBudget(
  files: readonly ReplyFile[],
  byteSize: number,
  limitBytes: number,
): boolean {
  return totalBytes(files) + byteSize <= limitBytes;
}

export function overBudget(files: readonly ReplyFile[], limitBytes: number): boolean {
  return totalBytes(files) > limitBytes;
}

/** The files' identity in order: what the server stores and what a save compares. */
export function filesKey(files: readonly ReplyFile[]): string {
  return files.map((file) => file.documentId).join(",");
}

export function sameFiles(a: readonly ReplyFile[], b: readonly ReplyFile[]): boolean {
  return filesKey(a) === filesKey(b);
}

/** The documents a reply can carry, grouped by the record they are stored on, message first. */
export function attachableSections(
  documents: readonly InboundReplyAttachable[],
): AttachableSection[] {
  return SOURCE_ORDER.map((source) => ({
    source,
    documents: documents.filter((document) => document.source === source),
  })).filter((section) => section.documents.length > 0);
}

export function fileFromAttachable(document: InboundReplyAttachable): ReplyFile {
  return {
    documentId: document.id,
    fileName: document.fileName,
    contentType: document.contentType,
    byteSize: document.byteSize,
  };
}

/** A document the upload route stored, as a file the reply carries. */
export function fileFromDocument(document: {
  id: string;
  originalName: string;
  fileType: string;
  fileSize: number;
}): ReplyFile {
  return {
    documentId: document.id,
    fileName: document.originalName,
    contentType: document.fileType,
    byteSize: document.fileSize,
  };
}

/** A new file is filed on the message's shipment when it has one, otherwise on the message. */
export function uploadTarget(message: {
  id: string;
  matchedShipment: { id: string } | null;
}): UploadTarget {
  return message.matchedShipment === null
    ? { resourceType: "inbound_message", resourceId: message.id }
    : { resourceType: "shipment", resourceId: message.matchedShipment.id };
}

/** Which chosen files fit in what is left of the budget, in the order they were chosen. */
export type UploadPlan<T> = {
  accepted: T[];
  refused: { file: T; remainingBytes: number }[];
};

export function planUploads<T extends { size: number }>(
  chosen: readonly T[],
  roomBytes: number,
): UploadPlan<T> {
  const plan: UploadPlan<T> = { accepted: [], refused: [] };
  let room = Math.max(roomBytes, 0);
  for (const file of chosen) {
    if (file.size <= room) {
      plan.accepted.push(file);
      room -= file.size;
    } else {
      plan.refused.push({ file, remainingBytes: room });
    }
  }

  return plan;
}
