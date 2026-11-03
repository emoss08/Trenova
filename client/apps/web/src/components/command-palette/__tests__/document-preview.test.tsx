import { documentContentUrl } from "@/services/document";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import type { Document } from "@trenova/shared/types/document";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { DocumentPreview, expiryRow } from "../_components/preview/document-preview";
import { record } from "./palette-test-fixtures";

const { documentService, download } = vi.hoisted(() => ({
  documentService: { getById: vi.fn(), getShipmentDraft: vi.fn() },
  download: vi.fn(),
}));

vi.mock("@/services/api", () => ({ apiService: { documentService } }));

vi.mock("@trenova/shared/lib/api", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@trenova/shared/lib/api")>();
  return { ...actual, api: { ...actual.api, download } };
});

const DAY = 86_400;

/** A stored document as `GET /documents/:id/` returns it (the `documentSchema` contract). */
function stored(fields: Partial<Document> = {}): Document {
  return {
    id: "doc_1",
    originalName: "rate-confirmation.pdf",
    fileName: "rate-confirmation.pdf",
    fileType: "application/pdf",
    fileSize: 245_760,
    createdAt: 1_790_000_000,
    isCurrentVersion: true,
    status: "Active",
    approvedAt: null,
    rejectionReason: null,
    previewStatus: "Ready",
    contentStatus: "Indexed",
    shipmentDraftStatus: "Unavailable",
    processingProfile: "none",
    signatureStatus: "Unknown",
    resourceType: "shipment",
    resourceId: "shp_1",
    documentType: null,
    expirationDate: null,
    ...fields,
  } as Document;
}

function pageEndpoint(page: number): string {
  return `/documents/doc_1/pages/${page}/image/`;
}

/** The server draws pages 1 to `count`, and answers a page past the end with not found. */
function serverDraws(count: number) {
  download.mockImplementation(async (endpoint: string) => {
    for (let page = 1; page <= count; page += 1) {
      if (endpoint === pageEndpoint(page)) {
        return { blob: new Blob([`page ${page}`], { type: "image/webp" }), filename: "page.webp" };
      }
    }
    throw new Error("This page is not in the document");
  });
}

function draw() {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={queryClient}>
      <DocumentPreview
        record={record({
          entityType: "document",
          id: "doc_1",
          title: "rate-confirmation.pdf",
          metadata: { attachedLabel: "PRO-1001" },
        })}
        onNavigate={vi.fn()}
      />
    </QueryClientProvider>,
  );
}

let urls = 0;

beforeEach(() => {
  urls = 0;
  URL.createObjectURL = vi.fn(() => `blob:page-${++urls}`);
  URL.revokeObjectURL = vi.fn();
});

afterEach(() => {
  vi.clearAllMocks();
});

describe("DocumentPreview", () => {
  it("draws the page the server renders for a document whose stored preview never finished", async () => {
    documentService.getById.mockResolvedValue(stored({ previewStatus: "Failed" }));
    serverDraws(1);

    draw();

    const sheet = await screen.findByRole("img", { name: "Page 1 of rate-confirmation.pdf" });
    expect(sheet.getAttribute("src")).toBe("blob:page-1");
    expect(download).toHaveBeenCalledWith(pageEndpoint(1), "page.webp", expect.anything());
  });

  it("steps to the next page once the server has drawn it, and stops at the last", async () => {
    documentService.getById.mockResolvedValue(stored());
    serverDraws(2);

    draw();

    const next = await screen.findByRole("button", { name: "Next page" });
    await waitFor(() => expect(next).toBeEnabled());
    expect(screen.getByRole("button", { name: "Previous page" })).toBeDisabled();

    fireEvent.click(next);

    expect(
      await screen.findByRole("img", { name: "Page 2 of rate-confirmation.pdf" }),
    ).toBeInTheDocument();
    expect(screen.getByText("Page 2")).toBeInTheDocument();
    await waitFor(() => expect(screen.getByRole("button", { name: "Next page" })).toBeDisabled());
    expect(screen.getByRole("button", { name: "Previous page" })).toBeEnabled();
  });

  it("shows no pager for a one-page file", async () => {
    documentService.getById.mockResolvedValue(stored());
    serverDraws(1);

    draw();

    await screen.findByRole("img", { name: "Page 1 of rate-confirmation.pdf" });
    await waitFor(() =>
      expect(download).toHaveBeenCalledWith(pageEndpoint(2), "page.webp", expect.anything()),
    );
    expect(screen.queryByRole("button", { name: "Next page" })).not.toBeInTheDocument();
  });

  it("asks for no page of a file the server cannot draw", async () => {
    documentService.getById.mockResolvedValue(
      stored({ originalName: "rates.csv", fileName: "rates.csv", fileType: "text/csv" }),
    );

    draw();

    expect(await screen.findByText("rates.csv")).toBeInTheDocument();
    expect(download).not.toHaveBeenCalled();
    expect(screen.queryByRole("img", { name: /^Page / })).not.toBeInTheDocument();
  });

  it("opens the whole file in a new tab from its page", async () => {
    documentService.getById.mockResolvedValue(stored());
    serverDraws(1);

    draw();

    const link = await screen.findByRole("link", { name: /rate-confirmation\.pdf/ });
    expect(link.getAttribute("href")).toBe(documentContentUrl("doc_1", "view"));
    expect(link.getAttribute("target")).toBe("_blank");
    expect(link.getAttribute("rel")).toBe("noopener noreferrer");
  });

  it("marks a rejected document and says why it was rejected", async () => {
    documentService.getById.mockResolvedValue(
      stored({ status: "Rejected", rejectionReason: "Signature is missing" }),
    );
    serverDraws(1);

    draw();

    expect(await screen.findByText("Rejected")).toBeInTheDocument();
    expect(screen.getByText("Rejected: Signature is missing")).toBeInTheDocument();
  });

  it("names the file's kind and size", async () => {
    documentService.getById.mockResolvedValue(stored());
    serverDraws(1);

    draw();

    expect(await screen.findByText("PDF · 240 KB")).toBeInTheDocument();
  });
});

describe("expiryRow", () => {
  const t: TranslateFn = (message, ...args) =>
    args.reduce<string>(
      (text, arg, index) => text.replace(`{${index}}`, String(arg)),
      message ?? "",
    );
  const now = 1_800_000_000;

  it("says nothing for a document that does not expire", () => {
    expect(expiryRow(null, now, t)).toBeNull();
    expect(expiryRow(0, now, t)).toBeNull();
  });

  it("marks a lapsed document as bad", () => {
    expect(expiryRow(now - DAY, now, t)).toMatchObject({ label: "Expires", state: "bad" });
    expect(expiryRow(now - DAY, now, t)?.value).toMatch(/^Expired /);
  });

  it("marks a document expiring within thirty days as worth a look", () => {
    expect(expiryRow(now + 30 * DAY, now, t)?.state).toBe("warn");
  });

  it("leaves a document good for longer unmarked", () => {
    expect(expiryRow(now + 31 * DAY, now, t)?.state).toBeUndefined();
  });
});
