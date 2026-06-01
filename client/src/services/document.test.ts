import { describe, expect, it } from "vitest";
import { DocumentService } from "./document";

describe("DocumentService document content URLs", () => {
  it("uses the configured API base URL for document view URLs", async () => {
    const service = new DocumentService();

    await expect(service.getViewUrl("doc_01KSXRKXGW7TRBAYHYSCQD16RW")).resolves.toBe(
      "/api/v1/documents/doc_01KSXRKXGW7TRBAYHYSCQD16RW/view/",
    );
  });

  it("encodes document IDs in content URLs", async () => {
    const service = new DocumentService();

    await expect(service.getDownloadUrl("doc/id with spaces")).resolves.toBe(
      "/api/v1/documents/doc%2Fid%20with%20spaces/download/",
    );
  });
});
