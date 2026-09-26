import { isCaptureRecordKind, type CaptureRecordKind } from "@/lib/capture";
import type {
  CaptureBatchDetail,
  CaptureItem,
  FileCaptureItemsEntryInput,
} from "@/lib/graphql/capture";

/** Where a person means to file one document. Blank until they say. */
export type Destination = {
  kind: CaptureRecordKind;
  recordId: string;
  documentTypeId: string;
};

type BatchTarget = Pick<CaptureBatchDetail, "targetType" | "targetId" | "documentTypeId">;

/**
 * The destination a document starts with. What the stack's own reading
 * suggested for it comes first — a cover sheet, the record it was scanned
 * into, or what the document itself says — then the record the whole stack
 * was sent to, and a shipment otherwise, because that is where most paper in
 * a freight office goes.
 */
export function initialDestination(item: CaptureItem | undefined, batch: BatchTarget): Destination {
  if (item?.suggestedId && isCaptureRecordKind(item.suggestedType)) {
    return {
      kind: item.suggestedType,
      recordId: item.suggestedId,
      documentTypeId: item.suggestedDocumentTypeId ?? batch.documentTypeId ?? "",
    };
  }
  if (batch.targetId && isCaptureRecordKind(batch.targetType)) {
    return {
      kind: batch.targetType,
      recordId: batch.targetId,
      documentTypeId: item?.suggestedDocumentTypeId ?? batch.documentTypeId ?? "",
    };
  }

  return { kind: "shipment", recordId: "", documentTypeId: item?.suggestedDocumentTypeId ?? "" };
}

/**
 * A destination is kept by the document's first page, not by the item's id.
 * Saving a new split gives the items new ids, but the server carries a
 * document's route by its first page, and so does this, so a destination a
 * person chose survives them splitting the stack around it.
 */
export function destinationKey(pageIds: readonly string[]): string {
  return pageIds[0] ?? "";
}

/** Changing the kind of record clears the record: an id means nothing across kinds. */
export function withKind(destination: Destination, kind: CaptureRecordKind): Destination {
  if (kind === destination.kind) {
    return destination;
  }

  return { kind, recordId: "", documentTypeId: destination.documentTypeId };
}

export function isFileable(destination: Destination): boolean {
  return destination.recordId !== "";
}

export function filingEntry(
  item: Pick<CaptureItem, "id" | "version">,
  destination: Destination,
): FileCaptureItemsEntryInput {
  return {
    itemId: item.id,
    targetType: destination.kind,
    targetId: destination.recordId,
    documentTypeId: destination.documentTypeId === "" ? null : destination.documentTypeId,
    version: item.version,
  };
}
