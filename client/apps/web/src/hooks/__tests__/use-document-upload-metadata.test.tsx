import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, renderHook, waitFor } from "@testing-library/react";
import type { ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { useDocumentUpload } from "../use-document-upload";

const mocks = vi.hoisted(() => ({
  createUploadSession: vi.fn(),
}));

vi.mock("@/services/document", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/services/document")>()),
  documentService: {
    createUploadSession: mocks.createUploadSession,
    getUploadSession: vi.fn(),
    getUploadPartTargets: vi.fn(),
    completeUploadSession: vi.fn(),
    cancelUploadSession: vi.fn().mockResolvedValue(undefined),
    getById: vi.fn(),
  },
}));

vi.mock("@/lib/document-upload-store", () => ({
  listPersistedDocumentUploadSessions: vi.fn().mockResolvedValue([]),
  persistDocumentUploadSession: vi.fn().mockResolvedValue(undefined),
  removeDocumentUploadSession: vi.fn().mockResolvedValue(undefined),
}));

function renderUpload(uploadMetadata: Record<string, string>) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const wrapper = ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={client}>{children}</QueryClientProvider>
  );
  return renderHook(
    () => useDocumentUpload({ resourceId: "shp_01", resourceType: "shipment", uploadMetadata }),
    { wrapper },
  );
}

const pdf = () => new File(["%PDF-1.7"], "bol.pdf", { type: "application/pdf" });

describe("useDocumentUpload metadata", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mocks.createUploadSession.mockRejectedValue(new Error("stop after the session request"));
  });

  it("files an upload with the hook's type and version target when the call names none", async () => {
    const { result } = renderUpload({ documentTypeId: "dt_pod", lineageId: "lin_pod" });

    act(() => result.current.uploadFiles([pdf()]));

    await waitFor(() => expect(mocks.createUploadSession).toHaveBeenCalledTimes(1));
    expect(mocks.createUploadSession).toHaveBeenCalledWith(
      expect.objectContaining({ documentTypeId: "dt_pod", lineageId: "lin_pod" }),
    );
  });

  it("files an upload with only what the call names, never as a new version of another document", async () => {
    const { result } = renderUpload({ documentTypeId: "dt_pod", lineageId: "lin_pod" });

    act(() => result.current.uploadFiles([pdf()], { documentTypeId: "dt_bol" }));

    await waitFor(() => expect(mocks.createUploadSession).toHaveBeenCalledTimes(1));
    expect(mocks.createUploadSession).toHaveBeenCalledWith(
      expect.objectContaining({ documentTypeId: "dt_bol", lineageId: undefined }),
    );
  });
});
