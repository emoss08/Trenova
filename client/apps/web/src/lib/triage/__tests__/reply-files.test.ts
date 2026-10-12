import type { InboundReplyAttachable } from "@/lib/graphql/inbox";
import { describe, expect, it } from "vitest";
import {
  attachableSections,
  fileExtension,
  fileFromAttachable,
  fileFromDocument,
  filesKey,
  fitsBudget,
  isImageExtension,
  overBudget,
  planUploads,
  remainingBytes,
  sameFiles,
  totalBytes,
  uploadTarget,
  withFile,
  withoutFile,
  type ReplyFile,
} from "../reply-files";

const ELD: ReplyFile = {
  documentId: "doc_eld",
  fileName: "ELD_SEED-PAY-002_Oct3.pdf",
  contentType: "application/pdf",
  byteSize: 48_000,
};
const BOL: ReplyFile = {
  documentId: "doc_bol",
  fileName: "BOL.pdf",
  contentType: "application/pdf",
  byteSize: 12_000,
};

function attachable(
  id: string,
  source: InboundReplyAttachable["source"],
  byteSize = 1_000,
): InboundReplyAttachable {
  return {
    id,
    fileName: `${id}.pdf`,
    contentType: "application/pdf",
    documentType: "",
    byteSize,
    createdAt: 1_700_000_000,
    source,
  };
}

describe("reply files", () => {
  it("adds a file once and removes it by document", () => {
    const once = withFile([ELD], ELD);
    expect(once).toEqual([ELD]);

    const both = withFile(once, BOL);
    expect(both.map((file) => file.documentId)).toEqual(["doc_eld", "doc_bol"]);
    expect(withoutFile(both, "doc_eld")).toEqual([BOL]);
  });

  it("adds up the files against the limit", () => {
    expect(totalBytes([ELD, BOL])).toBe(60_000);
    expect(remainingBytes([ELD, BOL], 100_000)).toBe(40_000);
    expect(remainingBytes([ELD, BOL], 50_000)).toBe(0);
    expect(fitsBudget([ELD], 52_000, 100_000)).toBe(true);
    expect(fitsBudget([ELD], 52_001, 100_000)).toBe(false);
    expect(overBudget([ELD, BOL], 60_000)).toBe(false);
    expect(overBudget([ELD, BOL], 59_999)).toBe(true);
  });

  it("compares sets of files by document, in order", () => {
    expect(filesKey([ELD, BOL])).toBe("doc_eld,doc_bol");
    expect(sameFiles([ELD, BOL], [{ ...ELD, fileName: "renamed.pdf" }, BOL])).toBe(true);
    expect(sameFiles([ELD, BOL], [BOL, ELD])).toBe(false);
    expect(sameFiles([], [])).toBe(true);
  });

  it("groups what can be attached by where it is stored, message first", () => {
    const sections = attachableSections([
      attachable("doc_cus", "Customer"),
      attachable("doc_bol", "Shipment"),
      attachable("doc_msg", "Message"),
      attachable("doc_pod", "Shipment"),
    ]);

    expect(sections.map((section) => section.source)).toEqual([
      "Message",
      "Shipment",
      "Customer",
    ]);
    expect(sections[1]?.documents.map((document) => document.id)).toEqual([
      "doc_bol",
      "doc_pod",
    ]);
  });

  it("reads a file from an attachable and from an uploaded document", () => {
    expect(fileFromAttachable(attachable("doc_msg", "Message", 2_048))).toEqual({
      documentId: "doc_msg",
      fileName: "doc_msg.pdf",
      contentType: "application/pdf",
      byteSize: 2_048,
    });
    expect(
      fileFromDocument({
        id: "doc_up",
        originalName: "gate-log.jpg",
        fileType: "image/jpeg",
        fileSize: 900,
      }),
    ).toEqual({
      documentId: "doc_up",
      fileName: "gate-log.jpg",
      contentType: "image/jpeg",
      byteSize: 900,
    });
  });

  it("uploads to the matched shipment, else to the message", () => {
    expect(uploadTarget({ id: "imsg_1", matchedShipment: { id: "shp_1" } })).toEqual({
      resourceType: "shipment",
      resourceId: "shp_1",
    });
    expect(uploadTarget({ id: "imsg_1", matchedShipment: null })).toEqual({
      resourceType: "inbound_message",
      resourceId: "imsg_1",
    });
  });

  it("names a file's kind by its extension", () => {
    expect(fileExtension("ELD_SEED-PAY-002_Oct3.PDF")).toBe("pdf");
    expect(fileExtension("scan")).toBe("file");
    expect(isImageExtension("jpeg")).toBe(true);
    expect(isImageExtension("pdf")).toBe(false);
  });

  it("uploads what fits and refuses the rest, keeping the order they were chosen", () => {
    const small = { name: "gate-log.jpg", size: 30 };
    const big = { name: "scan.pdf", size: 80 };
    const tiny = { name: "note.txt", size: 10 };

    const plan = planUploads([small, big, tiny], 50);

    expect(plan.accepted).toEqual([small, tiny]);
    expect(plan.refused).toEqual([{ file: big, remainingBytes: 20 }]);
    expect(planUploads([small], -5)).toEqual({
      accepted: [],
      refused: [{ file: small, remainingBytes: 0 }],
    });
  });
});
