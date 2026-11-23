import { WorkPaneHost, WorkPaneProvider } from "@/components/work-pane";
import { CLOSED_WORKER_OPEN_STATE, type WorkerOpenState } from "@/routes/worker/use-worker-state";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";
import WorkerDocuments from "../worker-documents";

const mocks = vi.hoisted(() => ({
  getByResource: vi.fn(),
  getPacketSummary: vi.fn(),
  getVersions: vi.fn(),
  remove: vi.fn(),
  getViewUrl: vi.fn(),
  getDownloadUrl: vi.fn(),
  uploadFiles: vi.fn(),
  uploads: [] as unknown[],
}));

vi.mock("@/services/document", () => ({
  documentContentUrl: (id: string, action: string) => `/api/v1/documents/${id}/${action}/`,
  documentService: {
    getByResource: mocks.getByResource,
    getPacketSummary: mocks.getPacketSummary,
    getVersions: mocks.getVersions,
    delete: mocks.remove,
    getViewUrl: mocks.getViewUrl,
    getDownloadUrl: mocks.getDownloadUrl,
    approve: vi.fn(),
    restoreVersion: vi.fn(),
    bulkDelete: vi.fn(),
  },
}));

vi.mock("@/hooks/use-document-upload", () => ({
  useDocumentUpload: () => ({
    uploads: mocks.uploads,
    uploadFiles: mocks.uploadFiles,
    cancelUpload: vi.fn(),
    retryUpload: vi.fn(),
    removeUpload: vi.fn(),
    clearCompleted: vi.fn(),
    clearAll: vi.fn(),
    isUploading: false,
  }),
}));

vi.mock("@/lib/queries/user", () => ({
  user: {
    avatar: (id: string) => ({
      queryKey: ["avatar", id],
      queryFn: async () => ({ id, name: id === "usr_dana" ? "Dana Whitfield" : "Marcus Bell" }),
    }),
  },
}));

vi.mock("@/hooks/use-permission", () => ({
  usePermission: () => ({ allowed: true, isLoading: false }),
}));

vi.mock("@trenova/shared/hooks/use-plan-feature", () => ({
  usePlanFeature: () => ({ available: false }),
}));

vi.mock("@/components/documents/capture/capture-panel", () => ({
  useCaptureKind: () => null,
  CaptureButton: () => null,
  CaptureRequestsPanel: () => null,
  CaptureStacksPanel: () => null,
}));

vi.mock("@/components/documents/document-storage-alert", () => ({
  DocumentStorageAlert: () => null,
}));
vi.mock("@/components/documents/extraction-needs-provider-alert", () => ({
  ExtractionNeedsProviderAlert: () => null,
}));

function document(overrides: Record<string, unknown>) {
  return {
    id: "doc",
    organizationId: "org_1",
    businessUnitId: "bu_1",
    lineageId: "lin",
    versionNumber: 1,
    isCurrentVersion: true,
    fileName: "file.pdf",
    originalName: "file.pdf",
    fileSize: 1_258_291,
    fileType: "application/pdf",
    storagePath: "x",
    status: "Active",
    resourceId: "wrk_1",
    resourceType: "worker",
    processingProfile: "none",
    isPublic: false,
    uploadedById: "usr_dana",
    previewStatus: "Ready",
    hasExtractedText: false,
    shipmentDraftStatus: "Unavailable",
    signatureStatus: "Unknown",
    signatureSource: null,
    documentType: null,
    version: 1,
    createdAt: 1_791_000_000,
    updatedAt: 1,
    ...overrides,
  };
}

const docs = [
  document({
    id: "doc_w4",
    originalName: "w4-2026.pdf",
    createdAt: 1_767_700_000,
    uploadedById: "usr_marcus",
  }),
  document({
    id: "doc_insp",
    originalName: "inspection-IL4417002931.pdf",
    documentType: { id: "dt_1", name: "Inspection report" },
  }),
  document({
    id: "doc_twic",
    originalName: "twic.jpg",
    fileType: "image/jpeg",
    createdAt: 1_713_000_000,
  }),
  document({ id: "doc_old", originalName: "old.pdf", isCurrentVersion: false }),
];

function renderView(open: Partial<WorkerOpenState> = {}) {
  const onOpen = vi.fn();
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  render(
    <QueryClientProvider client={client}>
      <WorkPaneProvider>
        <WorkPaneHost>
          <WorkerDocuments
            workerId="wrk_1"
            worker={{ id: "wrk_1", firstName: "Maria" } as never}
            open={{ ...CLOSED_WORKER_OPEN_STATE, ...open }}
            onOpen={onOpen}
          />
        </WorkPaneHost>
      </WorkPaneProvider>
    </QueryClientProvider>,
  );
  return onOpen;
}

async function loadedPane(name: string) {
  await screen.findByRole("group", { name: "Files to upload" });
  return screen.getByRole("dialog", { name });
}

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
  mocks.uploads = [];
});

describe("WorkerDocuments", () => {
  it("lists current documents newest first with their type, who added them and their size", async () => {
    mocks.getByResource.mockResolvedValue(docs);
    mocks.getPacketSummary.mockResolvedValue({ totalRules: 0 });
    renderView();

    const rows = await screen.findAllByTestId(/^document-/);
    expect(rows.map((row) => row.getAttribute("data-testid"))).toEqual([
      "document-doc_insp",
      "document-doc_w4",
      "document-doc_twic",
    ]);
    const inspection = within(rows[0]);
    expect(inspection.getByText("Inspection report")).toBeTruthy();
    expect(await inspection.findByText("Dana Whitfield")).toBeTruthy();
    expect(inspection.getByText("1.2 MB")).toBeTruthy();
    expect(within(rows[2]).getByText("Image")).toBeTruthy();
    expect(mocks.getByResource).toHaveBeenCalledWith("worker", "wrk_1", "", {
      includeDocumentType: "true",
    });
  });

  it("searches names, types and people, and offers to clear the filters when nothing matches", async () => {
    mocks.getByResource.mockResolvedValue(docs);
    mocks.getPacketSummary.mockResolvedValue({ totalRules: 0 });
    renderView();
    const user = userEvent.setup();

    await screen.findByTestId("document-doc_w4");
    const search = screen.getByRole("textbox", { name: "Search documents" });
    await user.type(search, "marcus");
    await waitFor(() => expect(screen.queryByTestId("document-doc_insp")).toBeNull());
    expect(screen.getByTestId("document-doc_w4")).toBeTruthy();
    expect(screen.getByText("1 of 3")).toBeTruthy();

    await user.clear(search);
    await user.type(search, "nothing like this");
    expect(screen.getByText("No documents match")).toBeTruthy();
    await user.click(screen.getByRole("button", { name: "Clear filters" }));
    expect(screen.getAllByTestId(/^document-/)).toHaveLength(3);
  });

  it("opens the upload in the page's work pane", async () => {
    mocks.getByResource.mockResolvedValue(docs);
    mocks.getPacketSummary.mockResolvedValue({ totalRules: 0 });
    const onOpen = renderView();

    await userEvent.setup().click(await screen.findByRole("button", { name: "Upload" }));
    expect(onOpen).toHaveBeenCalledWith({
      tab: "files",
      view: "documents",
      form: "upload-document",
    });
  });

  it("lists chosen files in the pane and uploads them together", async () => {
    mocks.getByResource.mockResolvedValue(docs);
    mocks.getPacketSummary.mockResolvedValue({ totalRules: 0 });
    renderView({ form: "upload-document" });
    const user = userEvent.setup();

    const pane = await loadedPane("Upload documents");
    const file = new File(["x"], "med-card-2026.pdf", { type: "application/pdf" });
    const input = pane.querySelector('input[type="file"]') as HTMLInputElement;
    fireEvent.change(input, { target: { files: [file] } });

    expect(await within(pane).findByText("med-card-2026.pdf")).toBeTruthy();
    expect(mocks.uploadFiles).not.toHaveBeenCalled();
    await user.click(within(pane).getByRole("button", { name: "Upload" }));
    expect(mocks.uploadFiles).toHaveBeenCalledWith([file], undefined);
  });

  it("uploads a new version under the document's lineage, one file only", async () => {
    mocks.getByResource.mockResolvedValue(docs);
    mocks.getPacketSummary.mockResolvedValue({ totalRules: 0 });
    renderView({ form: "upload-document", kind: "lin_w4" });
    const user = userEvent.setup();

    const pane = await loadedPane("Upload a new version");
    const input = pane.querySelector('input[type="file"]') as HTMLInputElement;
    expect(input.multiple).toBe(false);
    const file = new File(["x"], "w4-2026-v2.pdf", { type: "application/pdf" });
    fireEvent.change(input, { target: { files: [file] } });
    await user.click(await within(pane).findByRole("button", { name: "Upload" }));
    expect(mocks.uploadFiles).toHaveBeenCalledWith([file], { lineageId: "lin_w4" });
  });

  it("does not read the upload pane until it is first opened", async () => {
    mocks.getByResource.mockResolvedValue(docs);
    mocks.getPacketSummary.mockResolvedValue({ totalRules: 0 });
    renderView();

    await screen.findByTestId("document-doc_w4");
    expect(screen.queryByRole("dialog", { name: "Upload documents" })).toBeNull();
    expect(screen.queryByLabelText("Files to upload")).toBeNull();
  });

  it("asks for a file before uploading", async () => {
    mocks.getByResource.mockResolvedValue(docs);
    mocks.getPacketSummary.mockResolvedValue({ totalRules: 0 });
    renderView({ form: "upload-document" });

    const pane = await loadedPane("Upload documents");
    await userEvent.setup().click(within(pane).getByRole("button", { name: "Upload" }));
    expect(within(pane).getByText("Add a file before you upload.")).toBeTruthy();
    expect(mocks.uploadFiles).not.toHaveBeenCalled();
  });

  it("deletes a document only after the confirmation", async () => {
    mocks.getByResource.mockResolvedValue(docs);
    mocks.getPacketSummary.mockResolvedValue({ totalRules: 0 });
    mocks.remove.mockResolvedValue(undefined);
    renderView();
    const user = userEvent.setup();

    const row = within(await screen.findByTestId("document-doc_w4"));
    await user.click(row.getByRole("button", { name: "Actions for w4-2026.pdf" }));
    await user.click(await screen.findByRole("menuitem", { name: "Delete…" }));
    expect(mocks.remove).not.toHaveBeenCalled();

    const dialog = await screen.findByRole("alertdialog");
    expect(
      within(dialog).getByText("w4-2026.pdf comes off Maria's file. This cannot be undone."),
    ).toBeTruthy();
    await user.click(within(dialog).getByRole("button", { name: "Delete document" }));
    await waitFor(() => expect(mocks.remove).toHaveBeenCalledWith("doc_w4"));
  });

  it("opens a document beside the table when its row is chosen", async () => {
    mocks.getByResource.mockResolvedValue(docs);
    mocks.getPacketSummary.mockResolvedValue({ totalRules: 0 });
    renderView();

    await userEvent.setup().click(await screen.findByTestId("document-doc_insp"));
    const pane = await screen.findByRole("dialog", { name: "inspection-IL4417002931.pdf" });
    expect(within(pane).getByRole("button", { name: "Download" })).toBeTruthy();
  });

  it("asks for the first upload on an empty file and says when the read fails", async () => {
    mocks.getByResource.mockResolvedValueOnce([]);
    mocks.getPacketSummary.mockResolvedValue({ totalRules: 0 });
    renderView();
    expect(await screen.findByText("No documents on file")).toBeTruthy();

    cleanup();
    mocks.getByResource.mockRejectedValue(new Error("boom"));
    renderView();
    expect(await screen.findByText("The documents could not be loaded")).toBeTruthy();
  });
});
