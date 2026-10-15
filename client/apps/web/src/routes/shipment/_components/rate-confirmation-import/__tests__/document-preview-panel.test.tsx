import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import DocumentPreviewPanel from "../document-preview-panel";

const mocks = vi.hoisted(() => ({ download: vi.fn() }));

vi.mock("@trenova/shared/lib/api", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@trenova/shared/lib/api")>();
  return { ...actual, api: { ...actual.api, download: mocks.download } };
});

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

function renderPanel(fileName: string) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <QueryClientProvider client={client}>
      <DocumentPreviewPanel documentId="doc_1" fileName={fileName} />
    </QueryClientProvider>,
  );
}

describe("rate confirmation document preview", () => {
  // The workspace draws this same outline while the panel's code arrives, so the file
  // downloading inside the panel must not swap it for a different shape.
  it("draws the page's outline under the file name while the PDF downloads", () => {
    mocks.download.mockImplementation(() => new Promise<never>(() => {}));
    renderPanel("ratecon.pdf");

    expect(screen.getByText("ratecon.pdf")).toBeInTheDocument();
    expect(screen.getByLabelText("Loading the document")).toHaveAttribute("aria-busy", "true");
    expect(mocks.download).toHaveBeenCalledWith("/documents/doc_1/view/", "document.pdf", {
      signal: expect.any(AbortSignal),
    });
  });

  it("shows an image confirmation as the image, without downloading it as a PDF", () => {
    renderPanel("ratecon.png");

    expect(screen.getByRole("img", { name: "ratecon.png" })).toHaveAttribute(
      "src",
      expect.stringContaining("/documents/doc_1/view/"),
    );
    expect(mocks.download).not.toHaveBeenCalled();
  });
});
