import type { PersistedUploadRecord } from "@/lib/document-upload-store";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, renderHook, waitFor } from "@testing-library/react";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import type { DocumentUploadSession } from "@trenova/shared/types/document";
import type { User } from "@trenova/shared/types/user";
import type { ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { useDocumentUpload } from "../use-document-upload";

const mocks = vi.hoisted(() => ({
  listActiveUploadSessions: vi.fn(),
  createUploadSession: vi.fn(),
  getUploadSession: vi.fn(),
  listPersisted: vi.fn(),
  persist: vi.fn(),
  remove: vi.fn(),
}));

vi.mock("@/services/api", () => ({
  apiService: {
    documentService: {
      listActiveUploadSessions: mocks.listActiveUploadSessions,
      createUploadSession: mocks.createUploadSession,
      getUploadSession: mocks.getUploadSession,
    },
  },
}));

vi.mock("@/lib/document-upload-store", () => ({
  listPersistedDocumentUploadSessions: mocks.listPersisted,
  persistDocumentUploadSession: mocks.persist,
  removeDocumentUploadSession: mocks.remove,
}));

const JANE = "usr_jane:org_1";

function session(overrides: Partial<DocumentUploadSession> = {}): DocumentUploadSession {
  return {
    id: "dus_01",
    organizationId: "org_1",
    businessUnitId: "bu_1",
    documentId: null,
    lineageId: null,
    resourceId: "shp_01",
    resourceType: "shipment",
    processingProfile: "none",
    documentTypeId: null,
    originalName: "bol.pdf",
    contentType: "application/pdf",
    fileSize: 8,
    storagePath: "org_1/shipment/bol.pdf",
    storageProviderUploadId: "",
    strategy: "multipart",
    status: "Paused",
    description: null,
    tags: [],
    uploadedParts: [],
    partSize: 4,
    failureCode: null,
    failureMessage: null,
    malwareScan: false,
    threatName: null,
    expiresAt: 1_800_000_000,
    lastActivityAt: 1_790_000_000,
    version: 1,
    createdAt: 1_790_000_000,
    updatedAt: 1_790_000_000,
    ...overrides,
  };
}

function pdf(): File {
  return new File(["%PDF-1.7"], "bol.pdf", { type: "application/pdf" });
}

function persisted(id: string): PersistedUploadRecord {
  return {
    sessionId: id,
    scope: JANE,
    resourceId: "shp_01",
    resourceType: "shipment",
    session: session({ id }),
    file: pdf(),
  };
}

function signIn(user: Pick<User, "id" | "currentOrganizationId"> | null) {
  useAuthStore.setState({ user: user as User | null, isAuthenticated: user !== null });
}

function renderUpload() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const wrapper = ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={client}>{children}</QueryClientProvider>
  );
  return renderHook(() => useDocumentUpload({ resourceId: "shp_01", resourceType: "shipment" }), {
    wrapper,
  });
}

beforeEach(() => {
  vi.clearAllMocks();
  mocks.persist.mockResolvedValue(undefined);
  mocks.remove.mockResolvedValue(undefined);
  signIn({ id: "usr_jane", currentOrganizationId: "org_1" });
});

describe("useDocumentUpload's kept uploads", () => {
  it("resumes only the signed-in person's uploads to this resource", async () => {
    mocks.listPersisted.mockResolvedValue([persisted("dus_01")]);
    mocks.listActiveUploadSessions.mockResolvedValue([session({ id: "dus_01" })]);

    const { result } = renderUpload();

    await waitFor(() => expect(result.current.uploads).toHaveLength(1));
    expect(mocks.listPersisted).toHaveBeenCalledWith(JANE, "shipment", "shp_01");
    expect(result.current.uploads[0]).toMatchObject({ sessionId: "dus_01", status: "paused" });
  });

  it("keeps and resumes nothing while nobody is signed in", async () => {
    signIn(null);
    mocks.createUploadSession.mockResolvedValue(session({ status: "Initiated" }));
    mocks.getUploadSession.mockReturnValue(new Promise(() => undefined));

    const { result } = renderUpload();
    act(() => result.current.uploadFiles([pdf()]));

    await waitFor(() => expect(mocks.createUploadSession).toHaveBeenCalledTimes(1));
    expect(mocks.listPersisted).not.toHaveBeenCalled();
    expect(mocks.persist).not.toHaveBeenCalled();
  });

  it("records a new upload under the person who started it", async () => {
    mocks.listPersisted.mockResolvedValue([]);
    mocks.listActiveUploadSessions.mockResolvedValue([]);
    mocks.createUploadSession.mockResolvedValue(session({ status: "Initiated" }));
    mocks.getUploadSession.mockReturnValue(new Promise(() => undefined));

    const { result } = renderUpload();
    act(() => result.current.uploadFiles([pdf()]));

    await waitFor(() => expect(mocks.persist).toHaveBeenCalledTimes(1));
    expect(mocks.persist.mock.calls[0]?.[0]).toMatchObject({
      sessionId: "dus_01",
      scope: JANE,
      resourceId: "shp_01",
      resourceType: "shipment",
    });
  });

  it("deletes the kept file of a paused upload when every upload is cleared", async () => {
    mocks.listPersisted.mockResolvedValue([persisted("dus_01"), persisted("dus_02")]);
    mocks.listActiveUploadSessions.mockResolvedValue([
      session({ id: "dus_01" }),
      session({ id: "dus_02" }),
    ]);

    const { result } = renderUpload();
    await waitFor(() => expect(result.current.uploads).toHaveLength(2));

    act(() => result.current.clearAll());

    expect(result.current.uploads).toEqual([]);
    const removed: string[] = mocks.remove.mock.calls.map(([id]) => id as string);
    expect(removed.sort()).toEqual(["dus_01", "dus_02"]);
  });
});
