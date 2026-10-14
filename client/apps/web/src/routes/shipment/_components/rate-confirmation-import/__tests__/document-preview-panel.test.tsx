import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import DocumentPreviewPanel from "../document-preview-panel";

const mocks = vi.hoisted(() => ({ getViewUrl: vi.fn() }));

vi.mock("@/services/api", () => ({
  apiService: { documentService: { getViewUrl: mocks.getViewUrl } },
}));
vi.mock("@/lib/pdf-worker", () => ({}));
vi.mock("react-pdf", () => ({ Document: () => null, Page: () => null }));

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe("rate confirmation document preview", () => {
  // The workspace draws this same outline while the panel's code arrives, so
  // the document's address being read must not swap it for a different shape.
  it("draws the viewer's outline while the document's address is read", () => {
    mocks.getViewUrl.mockImplementation(() => new Promise<never>(() => {}));
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    render(
      <QueryClientProvider client={client}>
        <DocumentPreviewPanel documentId="doc_1" fileName="ratecon.pdf" />
      </QueryClientProvider>,
    );

    expect(screen.getByLabelText("Loading the document")).toHaveAttribute("aria-busy", "true");
    expect(screen.queryByText("Preview unavailable")).not.toBeInTheDocument();
    expect(mocks.getViewUrl).toHaveBeenCalledWith("doc_1");
  });
});
