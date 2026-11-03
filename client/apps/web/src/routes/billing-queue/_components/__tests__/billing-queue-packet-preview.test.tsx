import { stubLayout, stubScrollTo } from "@/test/layout";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { ApiRequestError } from "@trenova/shared/lib/api";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { BillingQueuePacketPreview, turnedOnce } from "../billing-queue-packet-preview";

const mocks = vi.hoisted(() => ({
  getPacketPreview: vi.fn(),
  getPacketChecks: vi.fn(),
  getById: vi.fn(),
  updateInvoiceDetails: vi.fn(),
  download: vi.fn(),
  downloadBlob: vi.fn(),
  toast: Object.assign(vi.fn(), { error: vi.fn(), success: vi.fn() }),
}));

vi.mock("@/services/api", () => ({
  apiService: {
    billingQueueService: {
      getPacketPreview: mocks.getPacketPreview,
      getPacketChecks: mocks.getPacketChecks,
      getById: mocks.getById,
      updateInvoiceDetails: mocks.updateInvoiceDetails,
    },
  },
}));
vi.mock("@trenova/shared/components/ui/toast", () => ({ toast: mocks.toast }));
vi.mock("@trenova/shared/lib/utils", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@trenova/shared/lib/utils")>();
  return { ...actual, downloadBlob: mocks.downloadBlob };
});
vi.mock("@trenova/shared/lib/api", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@trenova/shared/lib/api")>();
  return { ...actual, api: { ...actual.api, download: mocks.download } };
});

let restore: Array<() => void> = [];

beforeEach(() => {
  restore = [stubLayout(800, 600), stubScrollTo()];
  mocks.getById.mockRejectedValue(new Error("the item is not read in this test"));
  mocks.getPacketChecks.mockResolvedValue({ build: "a1b2c3d4e5f60718", blankPages: [] });
});

afterEach(() => {
  for (const undo of restore) undo();
  cleanup();
  vi.clearAllMocks();
});

const PACKET = "/billing-queue/bqi_1/packet-preview/";

function pageEndpoint(page: number) {
  return `${PACKET}pages/${page}/image/`;
}

function drawsEveryPage() {
  mocks.download.mockImplementation(async (endpoint: string) => ({
    blob: new Blob([endpoint], {
      type: endpoint.endsWith("/image/") ? "image/webp" : "application/pdf",
    }),
  }));
}

function renderPreview() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <BillingQueuePacketPreview itemId="bqi_1" open onOpenChange={() => {}} />
    </QueryClientProvider>,
  );
}

function requestedPages(): string[] {
  return mocks.download.mock.calls
    .map(([endpoint]) => endpoint as string)
    .filter((endpoint) => endpoint.endsWith("/image/"));
}

describe("BillingQueuePacketPreview", () => {
  it("shows the packet's first page in the page viewer, not the browser's", async () => {
    mocks.getPacketPreview.mockResolvedValue({
      fileName: "INV-1001.pdf",
      pageCount: 3,
      documents: 2,
      build: "a1b2c3d4e5f60718",
    });
    drawsEveryPage();

    renderPreview();

    expect(await screen.findByAltText("Page 1 of INV-1001.pdf")).toBeInTheDocument();
    expect(screen.getByText("1 of 3")).toBeInTheDocument();
    expect(screen.queryByTitle("Invoice packet")).not.toBeInTheDocument();
    expect(mocks.getPacketPreview).toHaveBeenCalledWith("bqi_1", expect.anything());
    expect(requestedPages()).toContain(pageEndpoint(1));
  });

  it("lists every page in the rail and turns to one clicked there", async () => {
    mocks.getPacketPreview.mockResolvedValue({
      fileName: "INV-1001.pdf",
      pageCount: 3,
      documents: 2,
      build: "a1b2c3d4e5f60718",
    });
    drawsEveryPage();
    renderPreview();
    await screen.findByAltText("Page 1 of INV-1001.pdf");

    expect(screen.getByRole("group", { name: "Pages" })).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Page 3" }));

    expect(screen.getByText("3 of 3")).toBeInTheDocument();
    await waitFor(() => expect(requestedPages()).toContain(pageEndpoint(3)));
    expect(requestedPages().filter((endpoint) => endpoint === pageEndpoint(3))).toHaveLength(1);
  });

  it("draws a hundred-page packet a screen at a time", async () => {
    mocks.getPacketPreview.mockResolvedValue({
      fileName: "INV-1001.pdf",
      pageCount: 100,
      documents: 12,
      build: "a1b2c3d4e5f60718",
    });
    drawsEveryPage();
    renderPreview();

    expect(await screen.findByAltText("Page 1 of INV-1001.pdf")).toBeInTheDocument();
    expect(screen.getByText("1 of 100")).toBeInTheDocument();
    expect(requestedPages().length).toBeLessThan(10);
    expect(requestedPages()).not.toContain(pageEndpoint(100));
  });

  it("saves the whole packet as one PDF under the name the server gives it", async () => {
    mocks.getPacketPreview.mockResolvedValue({
      fileName: "INV-1001.pdf",
      pageCount: 1,
      documents: 0,
      build: "a1b2c3d4e5f60718",
    });
    drawsEveryPage();
    const packet = new Blob(["%PDF packet"], { type: "application/pdf" });
    mocks.download.mockImplementation(async (endpoint: string) =>
      endpoint === PACKET
        ? { blob: packet, fileName: "INV-1001 packet.pdf" }
        : { blob: new Blob([endpoint], { type: "image/webp" }), fileName: "page.webp" },
    );
    renderPreview();
    await screen.findByAltText("Page 1 of INV-1001.pdf");

    fireEvent.click(screen.getByRole("button", { name: "Download" }));

    await waitFor(() =>
      expect(mocks.downloadBlob).toHaveBeenCalledWith(packet, "INV-1001 packet.pdf"),
    );
    expect(mocks.download).toHaveBeenCalledWith(PACKET, "INV-1001.pdf");
  });

  it("holds the Download button while the packet is fetched and says when it fails", async () => {
    mocks.getPacketPreview.mockResolvedValue({
      fileName: "INV-1001.pdf",
      pageCount: 1,
      documents: 0,
      build: "a1b2c3d4e5f60718",
    });
    drawsEveryPage();
    let fail: (reason: Error) => void = () => {};
    mocks.download.mockImplementation((endpoint: string) =>
      endpoint === PACKET
        ? new Promise((_, reject) => {
            fail = reject;
          })
        : Promise.resolve({
            blob: new Blob([endpoint], { type: "image/webp" }),
            fileName: "p.webp",
          }),
    );
    renderPreview();
    await screen.findByAltText("Page 1 of INV-1001.pdf");
    const button = screen.getByRole("button", { name: "Download" });

    fireEvent.click(button);
    fireEvent.click(button);

    expect(button).toBeDisabled();
    expect(mocks.download.mock.calls.filter(([endpoint]) => endpoint === PACKET)).toHaveLength(1);
    fail(new Error("the packet could not be built"));
    await waitFor(() =>
      expect(mocks.toast.error).toHaveBeenCalledWith("The file couldn't be downloaded"),
    );
    expect(button).not.toBeDisabled();
    expect(mocks.downloadBlob).not.toHaveBeenCalled();
  });

  it("says why a packet cannot be built and offers no retry the server would refuse", async () => {
    mocks.getPacketPreview.mockRejectedValue(
      new ApiRequestError(422, {
        type: "business-rule-violation",
        title: "Business Rule Violation",
        status: 422,
        detail:
          "This customer is billed on a statement; its packet is built when the statement closes",
      }),
    );
    drawsEveryPage();

    renderPreview();

    expect(await screen.findByRole("alert")).toHaveTextContent(
      "This customer is billed on a statement; its packet is built when the statement closes",
    );
    expect(screen.getByText("The packet couldn't be built.")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Try again" })).not.toBeInTheDocument();
    expect(requestedPages()).toEqual([]);
  });

  it("offers to try again when the server failed to build the packet", async () => {
    mocks.getPacketPreview
      .mockRejectedValueOnce(
        new ApiRequestError(500, {
          type: "internal-error",
          title: "Internal Server Error",
          status: 500,
        }),
      )
      .mockResolvedValueOnce({
        fileName: "INV-1001.pdf",
        pageCount: 1,
        documents: 0,
        build: "a1b2c3d4e5f60718",
      });
    drawsEveryPage();
    renderPreview();

    fireEvent.click(await screen.findByRole("button", { name: "Try again" }));

    expect(await screen.findByAltText("Page 1 of INV-1001.pdf")).toBeInTheDocument();
    expect(mocks.getPacketPreview).toHaveBeenCalledTimes(2);
  });
});

const BUILD = "b0b1b2b3b4b5b6b7";
const POD = "doc_pod";
const BOL = "doc_bol";

/** A packet as the server describes it: the invoice, a two-page POD that needs a signature it lacks, and a BOL. */
function packetWithDocuments(build = BUILD, documentIds = [POD, BOL]) {
  const named: Record<string, { name: string; pages: number; unsigned: boolean }> = {
    [POD]: { name: "Proof of delivery", pages: 2, unsigned: true },
    [BOL]: { name: "Bill of lading", pages: 1, unsigned: false },
  };
  let next = 2;
  const documents = documentIds.map((documentId) => {
    const { name, pages, unsigned } = named[documentId];
    const part = {
      kind: "Document",
      documentId,
      name,
      firstPage: next,
      lastPage: next + pages - 1,
      unsigned,
    };
    next += pages;
    return part;
  });
  return {
    fileName: "INV-1001.pdf",
    pageCount: next - 1,
    documents: documents.length,
    build,
    parts: [
      { kind: "Invoice", documentId: null, name: "", firstPage: 1, lastPage: 1, unsigned: false },
      ...documents,
    ],
  };
}

function editableItem(
  status = "InReview",
  packetPageRotations: { documentId: string; page: number; rotation: number }[] = [],
) {
  return {
    id: "bqi_1",
    version: 3,
    status,
    billType: "Invoice",
    poNumber: "",
    invoiceMemo: "",
    paymentTerm: "Net30",
    invoiceDate: null,
    packetDocumentIds: [],
    packetPageRotations,
  };
}

describe("BillingQueuePacketPreview, documents in the packet", () => {
  it("names each document in the rail and over its first page", async () => {
    mocks.getPacketPreview.mockResolvedValue(packetWithDocuments());
    drawsEveryPage();

    renderPreview();

    await screen.findByAltText("Page 1 of INV-1001.pdf");
    expect(screen.getAllByText("Invoice").length).toBeGreaterThan(0);
    expect(screen.getAllByText("Proof of delivery").length).toBeGreaterThan(0);
    expect(screen.getByText("2–3")).toBeInTheDocument();
    expect(screen.getByText("Pages 2–3")).toBeInTheDocument();
  });

  it("flags a document missing its signature and pages the server found blank", async () => {
    mocks.getPacketPreview.mockResolvedValue(packetWithDocuments());
    mocks.getPacketChecks.mockResolvedValue({ build: BUILD, blankPages: [3] });
    drawsEveryPage();

    renderPreview();

    expect(await screen.findByRole("button", { name: "2 to check" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Page 2, to check" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Page 3, to check" })).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "2 to check" }));

    expect(screen.getByText("2 of 4")).toBeInTheDocument();
    expect(screen.getByText("No signature found")).toBeInTheDocument();
    expect(screen.getByText("Proof of delivery needs a signature to count.")).toBeInTheDocument();
  });

  it("ignores checks made of another assembly of the packet", async () => {
    mocks.getPacketPreview.mockResolvedValue(packetWithDocuments(BUILD, [BOL]));
    mocks.getPacketChecks.mockResolvedValue({ build: "ffffffffffffffff", blankPages: [2] });
    drawsEveryPage();

    renderPreview();

    await screen.findByAltText("Page 1 of INV-1001.pdf");
    await waitFor(() => expect(mocks.getPacketChecks).toHaveBeenCalled());
    expect(screen.queryByText(/to check/)).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Page 2" })).toBeInTheDocument();
  });

  it("draws the rail from the packet's thumbnails, not its full pages", async () => {
    mocks.getPacketPreview.mockResolvedValue(packetWithDocuments());
    drawsEveryPage();

    renderPreview();

    await screen.findByAltText("Page 1 of INV-1001.pdf");
    await waitFor(() =>
      expect(mocks.download.mock.calls.map(([endpoint]) => endpoint)).toContain(
        `${PACKET}pages/1/thumbnail/`,
      ),
    );
  });

  it("takes a document out of the packet, assembles it again and offers Undo", async () => {
    mocks.getById.mockResolvedValue(editableItem());
    mocks.getPacketPreview
      .mockResolvedValueOnce(packetWithDocuments())
      .mockResolvedValue(packetWithDocuments("c0c1c2c3c4c5c6c7", [BOL]));
    mocks.updateInvoiceDetails.mockImplementation(
      async (_id: string, input: { version: number }) => ({
        ...editableItem(),
        ...input,
        version: input.version + 1,
      }),
    );
    drawsEveryPage();
    renderPreview();
    await screen.findByAltText("Page 1 of INV-1001.pdf");

    fireEvent.click(
      await screen.findByRole("button", { name: "Remove Proof of delivery from the packet" }),
    );

    await waitFor(() =>
      expect(mocks.updateInvoiceDetails).toHaveBeenCalledWith(
        "bqi_1",
        expect.objectContaining({ packetDocumentIds: [BOL] }),
      ),
    );
    await waitFor(() => expect(mocks.getPacketPreview).toHaveBeenCalledTimes(2));
    await waitFor(() =>
      expect(mocks.toast).toHaveBeenCalledWith(
        "Proof of delivery removed from the packet",
        expect.objectContaining({ action: expect.objectContaining({ label: "Undo" }) }),
      ),
    );

    const [, options] = mocks.toast.mock.calls.at(-1) as [
      string,
      { action: { onClick: () => void } },
    ];
    options.action.onClick();

    await waitFor(() =>
      expect(mocks.updateInvoiceDetails).toHaveBeenLastCalledWith(
        "bqi_1",
        expect.objectContaining({ packetDocumentIds: [POD, BOL] }),
      ),
    );
  });

  it("moves a document with the arrow keys on its handle", async () => {
    mocks.getById.mockResolvedValue(editableItem());
    mocks.getPacketPreview.mockResolvedValue(packetWithDocuments());
    mocks.updateInvoiceDetails.mockImplementation(
      async (_id: string, input: { version: number }) => ({
        ...editableItem(),
        ...input,
      }),
    );
    drawsEveryPage();
    renderPreview();
    await screen.findByAltText("Page 1 of INV-1001.pdf");

    fireEvent.keyDown(
      await screen.findByRole("button", {
        name: "Move Proof of delivery. Use the arrow keys to move it.",
      }),
      { key: "ArrowDown" },
    );

    await waitFor(() =>
      expect(mocks.updateInvoiceDetails).toHaveBeenCalledWith(
        "bqi_1",
        expect.objectContaining({ packetDocumentIds: [BOL, POD] }),
      ),
    );
  });

  it("offers to remove the document a flagged page belongs to, named only to screen readers", async () => {
    mocks.getById.mockResolvedValue(editableItem());
    mocks.getPacketPreview.mockResolvedValue(packetWithDocuments());
    mocks.getPacketChecks.mockResolvedValue({ build: BUILD, blankPages: [3] });
    drawsEveryPage();
    renderPreview();

    const note = (await screen.findByText("This page looks blank")).closest("[role=note]");
    expect(note).not.toBeNull();
    const remove = within(note as HTMLElement).getByRole("button", {
      name: "Remove Proof of delivery from the packet",
    });

    expect(remove).toHaveTextContent("Remove from packet");
  });

  it("removes a document once however many times its button is pressed", async () => {
    mocks.getById.mockResolvedValue(editableItem());
    mocks.getPacketPreview.mockResolvedValue(packetWithDocuments());
    mocks.getPacketChecks.mockResolvedValue({ build: BUILD, blankPages: [3] });
    let finish: (value: unknown) => void = () => {};
    mocks.updateInvoiceDetails.mockImplementation(
      () =>
        new Promise((resolve) => {
          finish = resolve;
        }),
    );
    drawsEveryPage();
    renderPreview();
    const note = (await screen.findByText("This page looks blank")).closest("[role=note]");
    const remove = within(note as HTMLElement).getByRole("button", {
      name: "Remove Proof of delivery from the packet",
    });

    const inRail = screen
      .getAllByRole("button", { name: "Remove Proof of delivery from the packet" })
      .find((button) => !note?.contains(button));

    fireEvent.click(remove);
    fireEvent.click(remove);
    fireEvent.click(inRail as HTMLElement);

    await waitFor(() => expect(mocks.updateInvoiceDetails).toHaveBeenCalledTimes(1));
    expect(remove).toBeDisabled();
    finish({ ...editableItem(), version: 4, packetDocumentIds: [BOL] });
    await waitFor(() => expect(mocks.toast).toHaveBeenCalledTimes(1));
    expect(mocks.updateInvoiceDetails).toHaveBeenCalledTimes(1);
  });

  it("keeps the invoice first, and takes the last document out as a pick of none", async () => {
    mocks.getById.mockResolvedValue(editableItem());
    mocks.getPacketPreview.mockResolvedValue(packetWithDocuments(BUILD, [BOL]));
    mocks.updateInvoiceDetails.mockResolvedValue({ ...editableItem(), version: 4 });
    drawsEveryPage();
    renderPreview();
    await screen.findByAltText("Page 1 of INV-1001.pdf");
    await waitFor(() => expect(mocks.getById).toHaveBeenCalled());

    expect(screen.queryByRole("button", { name: /^Move / })).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: /^Remove / }));

    await waitFor(() =>
      expect(mocks.updateInvoiceDetails).toHaveBeenCalledWith(
        "bqi_1",
        expect.objectContaining({ packetDocumentIds: [], packetPicked: true }),
      ),
    );
  });

  it("offers no changes once the invoice is approved", async () => {
    mocks.getById.mockResolvedValue(editableItem("Approved"));
    mocks.getPacketPreview.mockResolvedValue(packetWithDocuments());
    drawsEveryPage();
    renderPreview();
    await screen.findByAltText("Page 1 of INV-1001.pdf");
    await waitFor(() => expect(mocks.getById).toHaveBeenCalled());

    expect(screen.queryByRole("button", { name: /^Remove / })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /^Move / })).not.toBeInTheDocument();
  });
});

describe("BillingQueuePacketPreview, turning pages", () => {
  it("turns a document page in the packet, kept by the document's own page", async () => {
    mocks.getById.mockResolvedValue(editableItem());
    mocks.getPacketPreview.mockResolvedValue(packetWithDocuments());
    mocks.updateInvoiceDetails.mockImplementation(
      async (_id: string, input: { version: number }) => ({
        ...editableItem(),
        ...input,
      }),
    );
    drawsEveryPage();
    renderPreview();
    await screen.findByAltText("Page 1 of INV-1001.pdf");

    fireEvent.click(await screen.findByRole("button", { name: "Rotate page 3" }));

    await waitFor(() =>
      expect(mocks.updateInvoiceDetails).toHaveBeenCalledWith(
        "bqi_1",
        expect.objectContaining({
          packetDocumentIds: [POD, BOL],
          packetPageRotations: [{ documentId: POD, page: 2, rotation: 90 }],
        }),
      ),
    );
    await waitFor(() => expect(mocks.getPacketPreview).toHaveBeenCalledTimes(2));
  });

  it("does not turn the invoice's own pages, nor every page at once", async () => {
    mocks.getById.mockResolvedValue(editableItem());
    mocks.getPacketPreview.mockResolvedValue(packetWithDocuments());
    drawsEveryPage();
    renderPreview();
    await screen.findByRole("button", { name: "Rotate page 2" });

    expect(screen.queryByRole("button", { name: "Rotate page 1" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Rotate all pages" })).not.toBeInTheDocument();
  });

  it("turns pages only in the view once the invoice is approved", async () => {
    mocks.getById.mockResolvedValue(editableItem("Approved"));
    mocks.getPacketPreview.mockResolvedValue(packetWithDocuments());
    drawsEveryPage();
    renderPreview();
    await screen.findByAltText("Page 1 of INV-1001.pdf");
    await waitFor(() => expect(mocks.getById).toHaveBeenCalled());

    fireEvent.click(screen.getByRole("button", { name: "Rotate page 1" }));

    expect(screen.getByRole("button", { name: "Rotate all pages" })).toBeInTheDocument();
    expect(mocks.updateInvoiceDetails).not.toHaveBeenCalled();
  });

  it("puts a removed document's turns back with Undo", async () => {
    const turns = [{ documentId: POD, page: 1, rotation: 180 }];
    mocks.getById.mockResolvedValue(editableItem("InReview", turns));
    mocks.getPacketPreview
      .mockResolvedValueOnce(packetWithDocuments())
      .mockResolvedValue(packetWithDocuments("c0c1c2c3c4c5c6c7", [BOL]));
    mocks.updateInvoiceDetails.mockImplementation(
      async (_id: string, input: { version: number }) => ({
        ...editableItem(),
        ...input,
        version: input.version + 1,
      }),
    );
    drawsEveryPage();
    renderPreview();
    await screen.findByAltText("Page 1 of INV-1001.pdf");

    fireEvent.click(
      await screen.findByRole("button", { name: "Remove Proof of delivery from the packet" }),
    );
    await waitFor(() => expect(mocks.toast).toHaveBeenCalled());
    const [, options] = mocks.toast.mock.calls.at(-1) as [
      string,
      { action: { onClick: () => void } },
    ];
    options.action.onClick();

    await waitFor(() =>
      expect(mocks.updateInvoiceDetails).toHaveBeenLastCalledWith(
        "bqi_1",
        expect.objectContaining({ packetDocumentIds: [POD, BOL], packetPageRotations: turns }),
      ),
    );
  });
});

describe("turnedOnce", () => {
  it("adds a quarter turn to a page and drops the page after a full turn", () => {
    const once = turnedOnce([], "doc_a", 2);
    expect(once).toEqual([{ documentId: "doc_a", page: 2, rotation: 90 }]);

    const full = turnedOnce([{ documentId: "doc_a", page: 2, rotation: 270 }], "doc_a", 2);
    expect(full).toEqual([]);
  });

  it("leaves every other page's turn as it was", () => {
    const turns = [
      { documentId: "doc_a", page: 1, rotation: 90 },
      { documentId: "doc_b", page: 2, rotation: 180 },
    ];

    expect(turnedOnce(turns, "doc_b", 2)).toEqual([
      { documentId: "doc_a", page: 1, rotation: 90 },
      { documentId: "doc_b", page: 2, rotation: 270 },
    ]);
  });
});
