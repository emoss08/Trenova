import { formatMessage } from "@trenova/shared/i18n/format-message";
import type { TranslateFn } from "@trenova/shared/i18n/runtime";
import { formatFileSize } from "@trenova/shared/lib/utils";
import type { Document } from "@trenova/shared/types/document";
import { describe, expect, it } from "vitest";
import { cardDay } from "../record-card-values";
import { documentReviewLabel, documentTiles } from "../record-documents";

const t: TranslateFn = (message, ...args) => formatMessage("en", message ?? "", args);

/** A wall-clock moment in October 2026 in the zone the tests run in, as unix seconds. */
function at(day: number, hour: number): number {
  return new Date(2026, 9, day, hour).getTime() / 1000;
}

/** A document as the documents endpoint returns it, with its type when it has one. */
function attached(fields: Partial<Document> & { id: string }): Document {
  return {
    originalName: `${fields.id}.pdf`,
    fileType: "application/pdf",
    fileSize: 245_760,
    createdAt: at(12, 9),
    isCurrentVersion: true,
    status: "Active",
    approvedAt: null,
    previewStatus: "Ready",
    contentStatus: "Indexed",
    shipmentDraftStatus: "Unavailable",
    processingProfile: "none",
    documentType: null,
    ...fields,
  } as Document;
}

describe("documentTiles", () => {
  it("shows the current version of each document, newest first", () => {
    const tiles = documentTiles(
      [
        attached({ id: "doc_bol", createdAt: at(10, 8) }),
        attached({ id: "doc_old_pod", createdAt: at(13, 8), isCurrentVersion: false }),
        attached({ id: "doc_pod", createdAt: at(13, 9) }),
      ],
      t,
    );

    expect(tiles.map((tile) => tile.id)).toEqual(["doc_pod", "doc_bol"]);
  });

  it("names a document for its type and says what the file is, how big and when", () => {
    const [tile] = documentTiles(
      [
        attached({
          id: "doc_pod",
          originalName: "pod-1042.jpg",
          fileType: "image/jpeg",
          documentType: { name: "Proof of delivery", code: "POD" } as Document["documentType"],
        }),
      ],
      t,
    );

    expect(tile).toMatchObject({
      name: "pod-1042.jpg",
      type: "Proof of delivery",
      code: "POD",
      category: "image",
      fileType: "image/jpeg",
      preview: true,
      pending: false,
    });
    expect(tile.meta).toBe(`Image · ${formatFileSize(245_760)} · ${cardDay(at(12, 9))}`);
  });

  it("leaves a document without a type unnamed, and an unknown file's kind unsaid", () => {
    const [tile] = documentTiles(
      [attached({ id: "doc_x", originalName: "notes.bin", fileType: "application/octet-stream" })],
      t,
    );

    expect(tile.type).toBe("");
    expect(tile.code).toBe("");
    expect(tile.meta).toBe(`${formatFileSize(245_760)} · ${cardDay(at(12, 9))}`);
  });

  it("waits on a preview still being drawn, and draws none that failed", () => {
    const tiles = documentTiles(
      [
        attached({ id: "doc_wait", previewStatus: "Pending", createdAt: at(12, 10) }),
        attached({ id: "doc_failed", previewStatus: "Failed" }),
      ],
      t,
    );

    expect(tiles.map(({ preview, pending }) => ({ preview, pending }))).toEqual([
      { preview: false, pending: true },
      { preview: false, pending: false },
    ]);
  });

  it("reads where each document's review stands", () => {
    const reviews = documentTiles(
      [
        attached({ id: "a", status: "Active", approvedAt: at(12, 11), createdAt: at(12, 4) }),
        attached({ id: "b", status: "Rejected", createdAt: at(12, 3) }),
        attached({ id: "c", status: "PendingApproval", createdAt: at(12, 2) }),
        attached({ id: "d", status: "Active", createdAt: at(12, 1) }),
        attached({ id: "e", status: "Archived", createdAt: at(12, 0) }),
      ],
      t,
    ).map((tile) => tile.review);

    expect(reviews).toEqual(["approved", "rejected", "awaiting", "unreviewed", "unreviewed"]);
    expect(reviews.map((review) => documentReviewLabel(review, t))).toEqual([
      "Approved",
      "Rejected",
      "Awaiting review",
      "",
      "",
    ]);
  });
});
