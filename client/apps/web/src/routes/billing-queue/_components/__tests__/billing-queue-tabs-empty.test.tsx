import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MessageScrollerProvider } from "@trenova/shared/components/ui/message-scroller";
import { setLocale } from "@trenova/shared/i18n/runtime";
import type { ReactElement } from "react";
import { MemoryRouter } from "react-router";
import { afterEach, describe, expect, it, vi } from "vitest";
import AuditTab from "@/components/audit-tab";
import { CommentStream } from "@/routes/shipment/_components/comments/comment-stream";
import { BillingQueueDocumentsTab } from "../billing-queue-documents-tab";

const mocks = vi.hoisted(() => ({
  documents: vi.fn(),
  readiness: vi.fn(),
  audit: vi.fn(),
}));

vi.mock("@/services/document", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/services/document")>()),
  documentService: { getByResource: mocks.documents, delete: vi.fn() },
}));
vi.mock("@/lib/queries/shipment", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/lib/queries/shipment")>()),
  shipment: {
    billingReadiness: (id: string) => ({
      queryKey: ["billing-readiness", id],
      queryFn: mocks.readiness,
    }),
  },
}));
vi.mock("@/lib/queries/audit", () => ({
  audit: { history: (id: string) => ({ queryKey: ["audit-history", id] }) },
}));
vi.mock("@/components/autocomplete-fields", () => ({ DocumentTypeAutocompleteField: () => null }));
vi.mock("@/components/documents/upload-panel", () => ({
  UploadPanel: ({ isOpen }: { isOpen: boolean }) =>
    isOpen ? <div data-testid="upload-panel" /> : null,
}));
vi.mock("@/hooks/use-document-upload", () => ({
  useDocumentUpload: () => ({
    uploads: [],
    uploadFiles: vi.fn(),
    cancelUpload: vi.fn(),
    retryUpload: vi.fn(),
    removeUpload: vi.fn(),
    clearCompleted: vi.fn(),
  }),
}));
vi.mock("@/lib/graphql/record-activity", () => ({ fetchRecordActivity: mocks.audit }));
vi.mock("@/hooks/use-permission", () => ({
  usePermission: () => ({ allowed: true, isLoading: false }),
}));
vi.mock("@/components/image", () => ({
  LazyImage: ({ src, alt }: { src: string; alt: string }) => <img src={src} alt={alt} />,
}));
vi.mock("@/hooks/shipment-comments/use-new-comments-pill", () => ({
  useNewCommentsPill: () => ({ unseenCount: 0, clearUnseen: vi.fn() }),
}));

function renderWithin(ui: ReactElement) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <MemoryRouter>
      <QueryClientProvider client={client}>{ui}</QueryClientProvider>
    </MemoryRouter>,
  );
}

function renderStream(props: { isFiltered?: boolean; onClearFilters?: () => void } = {}) {
  renderWithin(
    <MessageScrollerProvider>
      <CommentStream
        shipmentId="shp_1"
        comments={[]}
        isLoading={false}
        isError={false}
        onRetry={vi.fn()}
        isFiltered={props.isFiltered ?? false}
        hasNextPage={false}
        isFetchingNextPage={false}
        fetchNextPage={vi.fn()}
        jumpRequest={null}
        highlightedCommentId={null}
        onHighlight={vi.fn()}
        onClearFilters={props.onClearFilters ?? vi.fn()}
        renderComment={() => null}
      />
    </MessageScrollerProvider>,
  );
}

function heading(name: string) {
  return screen.findByRole("heading", { level: 3, name });
}

afterEach(async () => {
  cleanup();
  vi.clearAllMocks();
  await act(() => setLocale("en"));
});

describe("billing queue documents tab", () => {
  it("offers the upload itself when the item can still take documents", async () => {
    mocks.documents.mockResolvedValue([]);
    mocks.readiness.mockResolvedValue(null);
    const user = userEvent.setup();
    renderWithin(
      <BillingQueueDocumentsTab shipmentId="shp_1" onDocumentSelect={vi.fn()} isEditable />,
    );

    expect(await heading("No documents yet")).toBeInTheDocument();
    expect(screen.getByText(/proof of delivery, bill of lading/)).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Upload a document" }));
    expect(screen.getByTestId("upload-panel")).toBeInTheDocument();
  });

  // Out of review the toolbar is gone, so the empty state must not offer an
  // upload the item cannot take; it says where the documents come from instead.
  it("says where documents come from once the item is out of review", async () => {
    mocks.documents.mockResolvedValue([]);
    mocks.readiness.mockResolvedValue(null);
    renderWithin(
      <BillingQueueDocumentsTab shipmentId="shp_1" onDocumentSelect={vi.fn()} isEditable={false} />,
    );

    expect(await heading("No documents yet")).toBeInTheDocument();
    expect(screen.getByText(/while the item is in review/)).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Upload a document" })).not.toBeInTheDocument();
  });

  it("names the invoice's documents as supporting ones", async () => {
    mocks.documents.mockResolvedValue([]);
    mocks.readiness.mockResolvedValue(null);
    renderWithin(
      <BillingQueueDocumentsTab
        shipmentId="shp_1"
        onDocumentSelect={vi.fn()}
        isEditable={false}
        context="invoice"
      />,
    );

    expect(await heading("No supporting documents")).toBeInTheDocument();
  });

  it("renders in the reader's language", async () => {
    mocks.documents.mockResolvedValue([]);
    mocks.readiness.mockResolvedValue(null);
    await act(() => setLocale("es"));
    renderWithin(
      <BillingQueueDocumentsTab shipmentId="shp_1" onDocumentSelect={vi.fn()} isEditable />,
    );

    expect(await heading("Todavía no hay documentos")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Cargar un documento" })).toBeInTheDocument();
  });
});

describe("comments tab", () => {
  it("says what comments are for when the shipment has none", async () => {
    renderStream();

    expect(await heading("No comments yet")).toBeInTheDocument();
    expect(screen.getByText(/@mention a teammate/)).toBeInTheDocument();
  });

  it("offers to clear the filters when they are what emptied the stream", async () => {
    const onClearFilters = vi.fn();
    const user = userEvent.setup();
    renderStream({ isFiltered: true, onClearFilters });

    expect(await heading("Nothing matches")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Clear filters" }));
    expect(onClearFilters).toHaveBeenCalledOnce();
  });

  it("renders in the reader's language", async () => {
    await act(() => setLocale("zh-TW"));
    renderStream();

    expect(await heading("尚無留言")).toBeInTheDocument();
  });
});

describe("activity tab", () => {
  it("says what the history will hold when nothing has been recorded", async () => {
    mocks.audit.mockResolvedValue({ rows: [], total: 0, endCursor: null, hasNextPage: false });
    renderWithin(<AuditTab resourceId="shp_1" />);

    expect(await heading("No activity yet")).toBeInTheDocument();
    expect(screen.getByText(/who made it, when, and what it changed/)).toBeInTheDocument();
  });

  it("renders in the reader's language", async () => {
    mocks.audit.mockResolvedValue({ rows: [], total: 0, endCursor: null, hasNextPage: false });
    await act(() => setLocale("zh-CN"));
    renderWithin(<AuditTab resourceId="shp_1" />);

    expect(await heading("暂无活动记录")).toBeInTheDocument();
  });
});

describe("billing queue documents tab rows", () => {
  const paper = (id: string, previewStatus: string, name: string) => ({
    id,
    lineageId: `lin_${id}`,
    documentTypeId: null,
    documentType: null,
    originalName: name,
    fileName: name,
    fileType: "application/pdf",
    fileSize: 48_100,
    status: "Active",
    signatureStatus: "NotRequired",
    previewStatus,
    isCurrentVersion: true,
    createdAt: Math.floor(Date.now() / 1000) - 3600,
  });

  it("shows each document's first page once the server has drawn it, and a blank page until then", async () => {
    mocks.documents.mockResolvedValue([
      paper("doc_ratecon", "Ready", "ratecon.pdf"),
      paper("doc_pod", "Pending", "pod.pdf"),
    ]);
    mocks.readiness.mockResolvedValue(null);
    renderWithin(
      <BillingQueueDocumentsTab shipmentId="shp_1" onDocumentSelect={vi.fn()} isEditable={false} />,
    );

    await screen.findByRole("button", { name: "ratecon.pdf" });
    const pages = document.querySelectorAll("img");
    expect(pages).toHaveLength(1);
    expect(pages[0]!.getAttribute("src")).toContain("/documents/doc_ratecon/preview/");
  });
});
