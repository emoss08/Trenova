import { describe, expect, it, vi } from "vitest";
import { DocumentService } from "./document";

// The base URL is resolved once at module load from import.meta.env.VITE_API_URL, which
// vite reads out of apps/web/.env — a gitignored file that exists on some machines and
// not others. Asserting the unconfigured default made this suite pass or fail on where
// it ran. Pinning the constant tests what documentContentUrl actually does: compose the
// configured base with an encoded id, whatever that base happens to be.
vi.mock("@trenova/shared/lib/constants", async (importActual) => {
  const actual = await importActual<typeof import("@trenova/shared/lib/constants")>();
  return { ...actual, API_BASE_URL: "/api/v1" };
});

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
