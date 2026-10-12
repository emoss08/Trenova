import { documentContentUrl } from "@/services/document";
import { clearPermissions, seedPermissions } from "@/test/permissions";
import { Operation, Resource } from "@trenova/shared/types/permission";
import {
  FakeIntersectionObserver,
  installIntersectionObserver,
} from "@/test/intersection-observer";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, fireEvent, render, screen, within } from "@testing-library/react";
import type { Document } from "@trenova/shared/types/document";
import { MemoryRouter } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { CardDocuments } from "../card-documents";
import { DocumentRail, RAIL_BATCH } from "../document-rail";
import type { DocumentTile } from "../record-documents";
import { CardDocumentsSkeleton } from "../../skeletons/record-card-skeleton";

const documentService = vi.hoisted(() => ({ getByResource: vi.fn() }));

vi.mock("@/services/api", () => ({ apiService: { documentService } }));

function attached(fields: Partial<Document> & { id: string }): Document {
  return {
    originalName: `${fields.id}.pdf`,
    fileType: "application/pdf",
    fileSize: 120_000,
    createdAt: 1_790_000_000,
    isCurrentVersion: true,
    status: "Active",
    approvedAt: null,
    previewStatus: "Ready",
    contentStatus: "Indexed",
    shipmentDraftStatus: "Unavailable",
    processingProfile: "none",
    documentType: null,
    ...fields,
  } as Document;
}

function tile(index: number): DocumentTile {
  return {
    id: `doc_${index}`,
    name: `doc_${index}.pdf`,
    type: "",
    code: "",
    fileType: "application/pdf",
    category: "pdf",
    meta: "PDF",
    preview: false,
    pending: false,
    review: "unreviewed",
  };
}

function draw(ui: React.ReactElement) {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter>{ui}</MemoryRouter>
    </QueryClientProvider>,
  );
}

beforeEach(() => {
  seedPermissions({ [Resource.Document]: Operation.Read });
  installIntersectionObserver();
  documentService.getByResource.mockReset();
  documentService.getByResource.mockResolvedValue([]);
});

afterEach(() => {
  clearPermissions();
  vi.unstubAllGlobals();
});

/*
A card's documents cost nothing until the reader scrolls to them: the section
holds the rail's outline and reads neither the documents nor the rail's code
until it comes within reach of the screen.
*/
describe("CardDocuments", () => {
  it("reads nothing until the section comes near the screen, then draws the rail", async () => {
    documentService.getByResource.mockResolvedValue([
      attached({
        id: "doc_pod",
        originalName: "pod-1042.jpg",
        fileType: "image/jpeg",
        approvedAt: 1_790_000_100,
        documentType: { name: "Proof of delivery", code: "POD" } as Document["documentType"],
      }),
    ]);
    const { container } = draw(<CardDocuments resourceType="shipment" resourceId="shp_1" />);

    expect(container.querySelector(".dk-rec-docs-sk")).not.toBeNull();
    expect(documentService.getByResource).not.toHaveBeenCalled();
    const body = container.querySelector(".dk-rec-docs-body");
    expect(
      FakeIntersectionObserver.instances.some(
        (observer) => observer.options.rootMargin === "320px 0px",
      ),
    ).toBe(true);

    act(() => FakeIntersectionObserver.show(body as Element));

    const pod = await screen.findByRole("link", { name: /Proof of delivery/ });
    expect(documentService.getByResource).toHaveBeenCalledWith("shipment", "shp_1", "", {
      includeDocumentType: "true",
    });
    expect(pod).toHaveAttribute("href", documentContentUrl("doc_pod", "view"));
    expect(screen.getByText("1 document")).toBeInTheDocument();
    expect(container.querySelector(".dk-rec-docs-sk")).toBeNull();
  });

  it("says when the record has no documents, and leads to where they are managed", async () => {
    const { container } = draw(
      <CardDocuments resourceType="shipment" resourceId="shp_1" manageTo="/shipments/shp_1" />,
    );
    act(() => FakeIntersectionObserver.show(container.querySelector(".dk-rec-docs-body")!));

    expect(await screen.findByText("No documents yet.")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: /Manage/ })).toHaveAttribute(
      "href",
      "/shipments/shp_1",
    );
  });

  it("says why the documents could not be read", async () => {
    documentService.getByResource.mockRejectedValue(new Error("Document storage is unavailable"));
    const { container } = draw(<CardDocuments resourceType="shipment" resourceId="shp_1" />);
    act(() => FakeIntersectionObserver.show(container.querySelector(".dk-rec-docs-body")!));

    expect(await screen.findByText("Document storage is unavailable")).toBeInTheDocument();
  });

  // The card is the record's; the documents are another resource. Someone who
  // may read the record but not documents sees the card without the section,
  // and the documents are never asked for.
  it("is left out, and reads nothing, for someone who may not read documents", () => {
    seedPermissions({ [Resource.Shipment]: Operation.Read });
    const { container } = draw(<CardDocuments resourceType="shipment" resourceId="shp_1" />);

    expect(container).toBeEmptyDOMElement();
    expect(FakeIntersectionObserver.instances).toHaveLength(0);
    expect(documentService.getByResource).not.toHaveBeenCalled();
  });

  it("leaves its outline out of a card's skeleton for someone who may not read documents", () => {
    seedPermissions({ [Resource.Shipment]: Operation.Read });
    const { container } = draw(<CardDocumentsSkeleton />);

    expect(container).toBeEmptyDOMElement();
  });
});

/*
However many documents a record has, the rail draws a batch and the next as
the reader nears its end, and each preview loads only as its tile comes near.
*/
describe("DocumentRail", () => {
  it("draws a batch of tiles and the next as the reader scrolls toward the end", () => {
    const { container } = draw(
      <DocumentRail tiles={Array.from({ length: 30 }, (_, index) => tile(index))} label="Docs" />,
    );
    const links = () => container.querySelectorAll("a.dk-rec-doc");
    const end = () => container.querySelector(".dk-rec-docs-end");

    expect(links()).toHaveLength(RAIL_BATCH);
    act(() => FakeIntersectionObserver.show(end()!));
    expect(links()).toHaveLength(2 * RAIL_BATCH);
    act(() => FakeIntersectionObserver.show(end()!));
    expect(links()).toHaveLength(30);
    expect(end()).toBeNull();
  });

  it("lays the tiles out in one list, previews loading lazily, each marked as it stands", () => {
    draw(
      <DocumentRail
        tiles={[
          { ...tile(1), preview: true, code: "BOL", review: "approved" },
          { ...tile(2), pending: true, review: "rejected" },
        ]}
        label="Shipment documents"
      />,
    );

    const list = screen.getByRole("list", { name: "Shipment documents" });
    const [first, second] = within(list).getAllByRole("link");
    const preview = first.querySelector("img");
    expect(preview).toHaveAttribute("src", documentContentUrl("doc_1", "preview"));
    expect(preview).toHaveAttribute("loading", "lazy");
    expect(within(first).getByText("BOL")).toBeInTheDocument();
    expect(within(first).getByText("Approved")).toBeInTheDocument();
    expect(second.querySelector("img")).toBeNull();
    expect(within(second).getByText("Preparing…")).toBeInTheDocument();
    expect(within(second).getByText("Rejected")).toBeInTheDocument();
  });

  it("gives every tile the same three lines, whatever it knows, and marks its review on the preview", () => {
    const { container } = draw(
      <DocumentRail
        tiles={[
          { ...tile(1), type: "Proof of delivery", review: "approved" },
          { ...tile(2), review: "unreviewed" },
        ]}
        label="Docs"
      />,
    );

    const bodies = Array.from(container.querySelectorAll(".dk-rec-doc-b"));
    expect(bodies.map((body) => Array.from(body.children).map((line) => line.textContent))).toEqual(
      [
        ["Proof of delivery", "doc_1.pdf", "PDF"],
        ["doc_2.pdf", "Unclassified", "PDF"],
      ],
    );
    expect(container.querySelector(".dk-rec-doc-p .dk-rec-doc-rv")).toHaveTextContent("Approved");
    expect(container.querySelector(".dk-rec-doc-b .dk-rec-doc-rv")).toBeNull();
  });

  it("offers its arrows only toward the ends that lie beyond, measuring once a frame", async () => {
    const frame = vi.spyOn(window, "requestAnimationFrame");
    draw(
      <DocumentRail tiles={Array.from({ length: 6 }, (_, index) => tile(index))} label="Docs" />,
    );
    const rail = screen.getByRole("list", { name: "Docs" });
    let left = 0;
    Object.defineProperties(rail, {
      clientWidth: { configurable: true, get: () => 400 },
      scrollWidth: { configurable: true, get: () => 1200 },
      scrollLeft: { configurable: true, get: () => left },
    });
    const nextFrame = () =>
      act(() => new Promise((done) => requestAnimationFrame(() => done(null))));

    expect(screen.queryByRole("button", { name: "More documents" })).not.toBeInTheDocument();
    frame.mockClear();
    fireEvent.scroll(rail);
    fireEvent.scroll(rail);
    fireEvent.scroll(rail);
    expect(frame).toHaveBeenCalledTimes(1);
    await nextFrame();
    expect(screen.getByRole("button", { name: "More documents" })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Earlier documents" })).not.toBeInTheDocument();

    left = 800;
    fireEvent.scroll(rail);
    await nextFrame();
    expect(screen.getByRole("button", { name: "Earlier documents" })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "More documents" })).not.toBeInTheDocument();
    frame.mockRestore();
  });
});
